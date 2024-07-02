package op

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"runtime"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/blob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/bloberror"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/pageblob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/service"
	"github.com/coreos/pkg/multierror"

	"github.com/flatcar/azure-vhd-utils/upload"
	"github.com/flatcar/azure-vhd-utils/upload/metadata"
	"github.com/flatcar/azure-vhd-utils/vhdcore/common"
	"github.com/flatcar/azure-vhd-utils/vhdcore/diskstream"
	"github.com/flatcar/azure-vhd-utils/vhdcore/validator"
)

type Error int

const (
	MissingVHDSuffix Error = iota
	BlobAlreadyExists
	MissingUploadMetadata
)

func (e Error) Error() string {
	switch e {
	case MissingVHDSuffix:
		return "missing .vhd suffix in blob name"
	case BlobAlreadyExists:
		return "blob already exists"
	case MissingUploadMetadata:
		return "blob has no upload metadata"
	default:
		return "unknown upload error"
	}
}

func ErrorIsAnyOf(err error, errs ...Error) bool {
	var opError Error
	if !errors.As(err, &opError) {
		return false
	}

	for _, e := range errs {
		if opError == e {
			return true
		}
	}

	return false
}

type UploadOptions struct {
	Overwrite   bool
	Parallelism int
	Logger      func(string)
}

func noopLogger(s string) {
}

func Upload(ctx context.Context, blobServiceClient *service.Client, container, blob, vhd string, opts *UploadOptions) error {
	const PageBlobPageSize int64 = 512
	const PageBlobPageSetSize int64 = 4 * 1024 * 1024

	if !strings.HasSuffix(strings.ToLower(blob), ".vhd") {
		return MissingVHDSuffix
	}

	if opts == nil {
		opts = &UploadOptions{
			Logger: noopLogger,
		}
	}

	parallelism := 8 * runtime.NumCPU()
	if opts.Parallelism > 0 {
		parallelism = opts.Parallelism
	}
	overwrite := opts.Overwrite
	logger := opts.Logger

	if err := ensureVHDSanity(vhd); err != nil {
		return err
	}

	diskStream, err := diskstream.CreateNewDiskStream(vhd)
	if err != nil {
		return err
	}
	defer diskStream.Close()

	containerClient := blobServiceClient.NewContainerClient(container)
	pageblobClient := containerClient.NewPageBlobClient(blob)
	blobClient := pageblobClient.BlobClient()

	_, err = containerClient.Create(ctx, nil)
	if err != nil && !bloberror.HasCode(err, bloberror.ContainerAlreadyExists, bloberror.ResourceAlreadyExists) {
		return err
	}

	blobExists := true
	blobProperties, err := blobClient.GetProperties(ctx, nil)
	if err != nil {
		if !bloberror.HasCode(err, bloberror.BlobNotFound, bloberror.ResourceNotFound) {
			return err
		}
		blobExists = false
	}

	resume := false
	var blobMetadata *metadata.Metadata
	if blobExists {
		if !overwrite {
			if len(blobProperties.ContentMD5) > 0 {
				return BlobAlreadyExists
			}
			blobMetadata, err = metadata.NewMetadataFromBlobMetadata(blobProperties.Metadata)
			if err != nil {
				return err
			}
			if blobMetadata == nil {
				return MissingUploadMetadata
			}
		}
		resume = true
		logger(fmt.Sprintf("Blob with name '%s' already exists, checking upload can be resumed", blob))
	}

	localMetadata, err := metadata.NewMetadataFromLocalVHD(vhd)
	if err != nil {
		return err
	}

	var rangesToSkip []*common.IndexRange
	if resume {
		if errs := metadata.CompareMetadata(blobMetadata, localMetadata); len(errs) > 0 {
			return multierror.Error(errs)
		}
		ranges, err := getAlreadyUploadedBlobRanges(ctx, pageblobClient)
		if err != nil {
			return err
		}
		rangesToSkip = ranges
	} else {
		if err := createBlob(ctx, pageblobClient, diskStream.GetSize(), localMetadata); err != nil {
			return err
		}
	}

	uploadableRanges, err := upload.LocateUploadableRanges(diskStream, rangesToSkip, PageBlobPageSize, PageBlobPageSetSize)
	if err != nil {
		return err
	}

	uploadableRanges, err = upload.DetectEmptyRanges(diskStream, uploadableRanges)
	if err != nil {
		return err
	}

	uploadContext := &upload.DiskUploadContext{
		VhdStream:             diskStream,
		AlreadyProcessedBytes: diskStream.GetSize() - common.TotalRangeLength(uploadableRanges),
		UploadableRanges:      uploadableRanges,
		PageblobClient:        pageblobClient,
		Parallelism:           parallelism,
		Resume:                resume,
	}

	err = upload.Upload(ctx, uploadContext)
	if err != nil {
		return err
	}

	if err := setBlobMD5Hash(ctx, blobClient, localMetadata); err != nil {
		return err
	}
	logger("Upload completed")
	return nil
}

// ensureVHDSanity ensure is VHD is valid for Azure.
func ensureVHDSanity(vhd string) error {
	if err := validator.ValidateVhd(vhd); err != nil {
		return err
	}

	if err := validator.ValidateVhdSize(vhd); err != nil {
		return err
	}

	return nil
}

// createBlob creates a page blob of specific size and sets custom
// metadata. The parameter client is the Azure pageblob client
// representing a blob in a container, size is the size of the new
// page blob in bytes and parameter vhdMetadata is the custom metadata
// to be associated with the page blob.
func createBlob(ctx context.Context, client *pageblob.Client, size int64, vhdMetadata *metadata.Metadata) error {
	m, err := vhdMetadata.ToMap()
	if err != nil {
		return err
	}
	opts := pageblob.CreateOptions{
		Metadata: m,
	}
	_, err = client.Create(ctx, size, &opts)
	return err
}

// setBlobMD5Hash sets MD5 hash of the blob in its properties
func setBlobMD5Hash(ctx context.Context, client *blob.Client, vhdMetadata *metadata.Metadata) error {
	if vhdMetadata.FileMetadata == nil || len(vhdMetadata.FileMetadata.MD5Hash) == 0 {
		return nil
	}
	buf := make([]byte, base64.StdEncoding.EncodedLen(len(vhdMetadata.FileMetadata.MD5Hash)))
	base64.StdEncoding.Encode(buf, vhdMetadata.FileMetadata.MD5Hash)
	blobHeaders := blob.HTTPHeaders{
		BlobContentMD5: buf,
	}
	_, err := client.SetHTTPHeaders(ctx, blobHeaders, nil)
	return err
}

// getAlreadyUploadedBlobRanges returns the range slice containing
// ranges of a page blob those are already uploaded. The parameter
// client is the Azure pageblob client representing a blob in a
// container.
func getAlreadyUploadedBlobRanges(ctx context.Context, client *pageblob.Client) ([]*common.IndexRange, error) {
	var (
		marker       *string
		rangesToSkip []*common.IndexRange
	)
	for {
		opts := pageblob.GetPageRangesOptions{
			Marker: marker,
		}
		pager := client.NewGetPageRangesPager(&opts)
		for pager.More() {
			response, err := pager.NextPage(ctx)
			if err != nil {
				return nil, err
			}
			tmpRanges := make([]*common.IndexRange, len(response.PageRange))
			for i, page := range response.PageRange {
				tmpRanges[i] = common.NewIndexRange(*page.Start, *page.End)
			}
			rangesToSkip = append(rangesToSkip, tmpRanges...)
			marker = response.NextMarker
		}
		if marker == nil || *marker == "" {
			break
		}
	}
	return rangesToSkip, nil
}
