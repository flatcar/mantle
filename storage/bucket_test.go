// Copyright The Mantle Authors
// SPDX-License-Identifier: Apache-2.0

package storage

import (
	"fmt"
	"net/http"
	"testing"

	"google.golang.org/api/storage/v1"
)

type fakeTransport struct{}

func (f fakeTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return nil, fmt.Errorf("FAKE! %s %s", req.Method, req.URL)
}

func FakeBucket(bucketURL string) (*Bucket, error) {
	return NewBucket(&http.Client{Transport: fakeTransport{}}, bucketURL)
}

func (b *Bucket) AddObject(obj *storage.Object) {
	b.addObject(obj)
}

func TestBucketURL(t *testing.T) {
	if _, err := FakeBucket("sftp://bucket/"); err != UnknownScheme {
		t.Errorf("Unexpected error: %v", err)
	}

	if _, err := FakeBucket("gs:///"); err != UnknownBucket {
		t.Errorf("Unexpected error: %v", err)
	}

	for _, test := range []struct {
		url    string
		name   string
		prefix string
	}{
		{"gs://bucket", "bucket", ""},
		{"gs://bucket/", "bucket", ""},
		{"gs://bucket/prefix", "bucket", "prefix/"},
		{"gs://bucket/prefix/", "bucket", "prefix/"},
		{"gs://bucket/prefix/foo", "bucket", "prefix/foo/"},
		{"gs://bucket/prefix/foo/", "bucket", "prefix/foo/"},
	} {

		bkt, err := FakeBucket(test.url)
		if err != nil {
			t.Errorf("Unexpected error for url %q: %v", test.url, err)
			continue
		}

		if bkt.Name() != test.name {
			t.Errorf("Unexpected name for url %q: %q", test.url, bkt.Name())
		}
		if bkt.Prefix() != test.prefix {
			t.Errorf("Unexpected name for url %q: %q", test.url, bkt.Prefix())
		}
	}

}

func ExampleNextPrefix() {
	fmt.Println(NextPrefix("foo/bar/baz"))
	fmt.Println(NextPrefix("foo/bar/"))
	fmt.Println(NextPrefix("foo/bar"))
	fmt.Println(NextPrefix("foo/"))
	fmt.Println(NextPrefix("foo"))
	fmt.Println(NextPrefix(""))
	// Output:
	// foo/bar/
	// foo/
	// foo/
	//
	//
}
