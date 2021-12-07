// Copyright The Mantle Authors
// SPDX-License-Identifier: Apache-2.0

package generator

import (
	"bytes"
	"io/ioutil"
	"os"
	"testing"

	"github.com/golang/protobuf/proto"

	"github.com/flatcar-linux/mantle/update"
	"github.com/flatcar-linux/mantle/update/metadata"
)

type testGenerator struct {
	Generator
	t *testing.T
}

// Report errors to testing framework instead of return value.
func (g *testGenerator) Destroy() {
	g.Generator.Destroy()
}

func TestGenerateWithoutPartition(t *testing.T) {
	g := testGenerator{t: t}
	defer g.Destroy()

	f, err := ioutil.TempFile("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	defer os.Remove(f.Name())

	if err := g.Write(f.Name()); err != nil {
		t.Fatal(err)
	}

	if _, err := f.Seek(0, os.SEEK_SET); err != nil {
		t.Fatal(err)
	}

	payload, err := update.NewPayloadFrom(f)
	if err != nil {
		t.Fatal(err)
	}

	if err := payload.Verify(); err != nil {
		t.Fatal(err)
	}
}

func TestGenerateOneBlockPartition(t *testing.T) {
	g := testGenerator{t: t}
	defer g.Destroy()

	proc := Procedure{
		InstallProcedure: metadata.InstallProcedure{
			NewInfo: &metadata.InstallInfo{
				Hash: testOnesHash,
				Size: proto.Uint64(BlockSize),
			},
			Operations: []*metadata.InstallOperation{
				&metadata.InstallOperation{
					Type: metadata.InstallOperation_REPLACE.Enum(),
					DstExtents: []*metadata.Extent{&metadata.Extent{
						StartBlock: proto.Uint64(0),
						NumBlocks:  proto.Uint64(1),
					}},
					DataLength:     proto.Uint32(BlockSize),
					DataSha256Hash: testOnesHash,
				},
			},
		},
		ReadCloser: ioutil.NopCloser(bytes.NewReader(testOnes)),
	}
	if err := g.Partition(&proc); err != nil {
		t.Fatal(err)
	}

	f, err := ioutil.TempFile("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	defer os.Remove(f.Name())

	if err := g.Write(f.Name()); err != nil {
		t.Fatal(err)
	}

	if _, err := f.Seek(0, os.SEEK_SET); err != nil {
		t.Fatal(err)
	}

	out, err := ioutil.TempFile("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer out.Close()
	defer os.Remove(out.Name())

	updater := update.Updater{
		DstPartition: out.Name(),
	}

	if err := updater.UsePayload(f); err != nil {
		t.Fatal(err)
	}

	if err := updater.Update(); err != nil {
		t.Fatal(err)
	}

	written, err := ioutil.ReadAll(out)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(written, testOnes) {
		t.Errorf("Updater did not replicate source block")
	}
}
