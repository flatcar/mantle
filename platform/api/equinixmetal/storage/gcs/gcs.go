// Copyright The Mantle Authors.
// SPDX-License-Identifier: Apache-2.0
package gcs

import (
	"bytes"
	"context"
	"fmt"

	gs "google.golang.org/api/storage/v1"

	"github.com/flatcar-linux/mantle/platform/api/equinixmetal/storage"
	ms "github.com/flatcar-linux/mantle/storage"
)

func New(bucket *ms.Bucket) storage.Storage {
	return &GCS{
		bucket: bucket,
	}
}

// GCS implements a Storage interface for Google Cloud Storage.
type GCS struct {
	bucket *ms.Bucket
}

// Upload the []byte data to Google Cloud Storage bucket.
func (g *GCS) Upload(name, contentType string, data []byte) (string, string, error) {
	obj := gs.Object{
		Name:        g.bucket.Prefix() + name,
		ContentType: contentType,
	}
	err := g.bucket.Upload(context.TODO(), &obj, bytes.NewReader(data))
	if err != nil {
		return "", "", fmt.Errorf("uploading object: %v", err)
	}

	url := fmt.Sprintf("https://bucket.release.flatcar-linux.net/%v/%v", g.bucket.Name(), obj.Name)
	return obj.Name, url, nil
}

// Delete the uploaded data from Google Cloud Storage bucket.
func (g *GCS) Delete(ctx context.Context, name string) error {
	return g.bucket.Delete(ctx, name)
}

func (g *GCS) Close() error { return nil }
