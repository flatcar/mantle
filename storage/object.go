// Copyright The Mantle Authors
// SPDX-License-Identifier: Apache-2.0

package storage

import (
	"encoding/base64"
	"hash/crc32"
	"io"
	"sort"

	"google.golang.org/api/storage/v1"

	"github.com/flatcar-linux/mantle/lang/natsort"
	"github.com/flatcar-linux/mantle/lang/reader"
)

// SortObjects orders Objects by Name using natural sorting.
func SortObjects(objs []*storage.Object) {
	sort.Slice(objs, func(i, j int) bool {
		return natsort.Less(objs[i].Name, objs[j].Name)
	})
}

// Update CRC32c and Size in the given Object
func crcSum(obj *storage.Object, media io.ReaderAt) error {
	c := crc32.New(crc32.MakeTable(crc32.Castagnoli))
	n, err := io.Copy(c, reader.AtReader(media))
	if err != nil {
		return err
	}
	obj.Size = uint64(n)
	obj.Crc32c = base64.StdEncoding.EncodeToString(c.Sum(nil))
	return nil
}

// Judges whether two Objects are equal based on size and CRC. To guard against
// uninitialized fields, nil objects and empty CRC values are never equal.
func crcEq(a, b *storage.Object) bool {
	if a == nil || b == nil {
		return false
	}
	if a.Crc32c == "" || b.Crc32c == "" {
		return false
	}
	return a.Size == b.Size && a.Crc32c == b.Crc32c
}

// Duplicate basic Object metadata, useful for preparing a copy operation.
func dupObj(src *storage.Object) *storage.Object {
	dst := &storage.Object{
		Bucket:             src.Bucket,
		CacheControl:       src.CacheControl,
		ContentDisposition: src.ContentDisposition,
		ContentEncoding:    src.ContentEncoding,
		ContentLanguage:    src.ContentLanguage,
		ContentType:        src.ContentType,
		Crc32c:             src.Crc32c,
		Md5Hash:            src.Md5Hash,
		Name:               src.Name,
		Size:               src.Size,
	}
	if len(src.Metadata) > 0 {
		dst.Metadata = make(map[string]string)
		for k, v := range src.Metadata {
			dst.Metadata[k] = v
		}
	}
	return dst
}
