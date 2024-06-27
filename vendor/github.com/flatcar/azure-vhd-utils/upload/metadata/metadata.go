package metadata

import (
	"bytes"
	"crypto/md5"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/flatcar/azure-vhd-utils/upload/progress"
	"github.com/flatcar/azure-vhd-utils/vhdcore/diskstream"
)

// The key of the page blob metadata collection entry holding VHD metadata as json.
const metadataKey = "diskmetadata"

// Metadata is the type representing metadata associated with an Azure page blob holding the VHD.
// This will be stored as a JSON string in the page blob metadata collection with key 'diskmetadata'.
type Metadata struct {
	FileMetadata *FileMetadata `json:"fileMetaData"`
}

// FileMetadata represents the metadata of a VHD file.
type FileMetadata struct {
	FileName         string    `json:"fileName"`
	FileSize         int64     `json:"fileSize"`
	VHDSize          int64     `json:"vhdSize"`
	LastModifiedTime time.Time `json:"lastModifiedTime"`
	MD5Hash          []byte    `json:"md5Hash"` // Marshal will encodes []byte as a base64-encoded string
}

// ToJSON returns Metadata as a json string.
func (m *Metadata) ToJSON() (string, error) {
	b, err := json.Marshal(m)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// ToMap returns the map representation of the Metadata which can be stored in the page blob metadata collection.
func (m *Metadata) ToMap() (map[string]*string, error) {
	v, err := m.ToJSON()
	if err != nil {
		return nil, err
	}

	return map[string]*string{metadataKey: &v}, nil
}

// NewMetadataFromLocalVHD creates a Metadata instance that should be associated with the page blob
// holding the VHD. The parameter vhdPath is the path to the local VHD.
func NewMetadataFromLocalVHD(vhdPath string) (*Metadata, error) {
	fileStat, err := getFileStat(vhdPath)
	if err != nil {
		return nil, err
	}

	fileMetadata := &FileMetadata{
		FileName:         fileStat.Name(),
		FileSize:         fileStat.Size(),
		LastModifiedTime: fileStat.ModTime(),
	}

	diskStream, err := diskstream.CreateNewDiskStream(vhdPath)
	if err != nil {
		return nil, err
	}
	defer diskStream.Close()
	fileMetadata.VHDSize = diskStream.GetSize()
	fileMetadata.MD5Hash, err = calculateMD5Hash(diskStream)
	if err != nil {
		return nil, err
	}

	return &Metadata{
		FileMetadata: fileMetadata,
	}, nil
}

// NewMetadataFromBlobMetadata returns Metadata instance associated with a Azure page blob, if there is no Metadata
// associated with the blob it returns nil value for Metadata
func NewMetadataFromBlobMetadata(blobmd map[string]*string) (*Metadata, error) {
	m, ok := blobmd[metadataKey]
	if !ok || m == nil {
		return nil, nil
	}
	metadata := new(Metadata)
	if err := json.Unmarshal([]byte(*m), metadata); err != nil {
		return nil, fmt.Errorf("NewMetadataFromBlobMetadata, failed to deserialize blob metadata with key %s: %v", metadataKey, err)
	}
	return metadata, nil
}

// CompareMetadata compares the Metadata associated with the remote page blob and local VHD file. If both metadata
// are same this method returns an empty error slice else a non-empty error slice with each error describing
// the metadata entry that mismatched.
func CompareMetadata(remote, local *Metadata) []error {
	var metadataErrors = make([]error, 0)
	if !bytes.Equal(remote.FileMetadata.MD5Hash, local.FileMetadata.MD5Hash) {
		metadataErrors = append(metadataErrors,
			fmt.Errorf("MD5 hash of VHD file in Azure blob storage (%v) and local VHD file (%v) does not match",
				base64.StdEncoding.EncodeToString(remote.FileMetadata.MD5Hash),
				base64.StdEncoding.EncodeToString(local.FileMetadata.MD5Hash)))
	}

	if remote.FileMetadata.VHDSize != local.FileMetadata.VHDSize {
		metadataErrors = append(metadataErrors,
			fmt.Errorf("Logical size of the VHD file in Azure blob storage (%d) and local VHD file (%d) does not match",
				remote.FileMetadata.VHDSize, local.FileMetadata.VHDSize))
	}

	if remote.FileMetadata.FileSize != local.FileMetadata.FileSize {
		metadataErrors = append(metadataErrors,
			fmt.Errorf("Size of the VHD file in Azure blob storage (%d) and local VHD file (%d) does not match",
				remote.FileMetadata.FileSize, local.FileMetadata.FileSize))
	}

	if remote.FileMetadata.LastModifiedTime != local.FileMetadata.LastModifiedTime {
		metadataErrors = append(metadataErrors,
			fmt.Errorf("Last modified time of the VHD file in Azure blob storage (%v) and local VHD file (%v) does not match",
				remote.FileMetadata.LastModifiedTime, local.FileMetadata.LastModifiedTime))
	}

	if remote.FileMetadata.FileName != local.FileMetadata.FileName {
		metadataErrors = append(metadataErrors,
			fmt.Errorf("Full name of the VHD file in Azure blob storage (%s) and local VHD file (%s) does not match",
				remote.FileMetadata.FileName, local.FileMetadata.FileName))
	}

	return metadataErrors
}

// getFileStat returns os.FileInfo of a file.
func getFileStat(filePath string) (os.FileInfo, error) {
	fd, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("fileMetadata.getFileStat: %v", err)
	}
	defer fd.Close()
	return fd.Stat()
}

// calculateMD5Hash compute the MD5 checksum of a disk stream, it writes the compute progress in stdout
// If there is an error in reading file, then the MD5 compute will stop and it return error.
func calculateMD5Hash(diskStream *diskstream.DiskStream) ([]byte, error) {
	progressStream := progress.NewReaderWithProgress(diskStream, diskStream.GetSize(), 1*time.Second)
	defer progressStream.Close()

	go func() {
		s := time.Time{}
		fmt.Println("Computing MD5 Checksum..")
		for progressRecord := range progressStream.ProgressChan {
			t := s.Add(progressRecord.RemainingDuration)
			fmt.Printf("\r Completed: %3d%% RemainingTime: %02dh:%02dm:%02ds Throughput: %d MB/sec",
				int(progressRecord.PercentComplete),
				t.Hour(), t.Minute(), t.Second(),
				int(progressRecord.AverageThroughputMbPerSecond),
			)
		}
	}()

	h := md5.New()
	buf := make([]byte, 2097152) // 2 MB staging buffer
	_, err := io.CopyBuffer(h, progressStream, buf)
	if err != nil {
		return nil, err
	}
	return h.Sum(nil), nil
}
