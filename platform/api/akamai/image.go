// Copyright The Mantle Authors.
// SPDX-License-Identifier: Apache-2.0

package akamai

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/flatcar/mantle/util"
	"github.com/linode/linodego"
)

func (a *API) UploadImage(ctx context.Context, name, file string) (string, error) {
	f, err := os.Open(file)
	if err != nil {
		return "", fmt.Errorf("opening file: %w", err)
	}

	opts := linodego.ImageUploadOptions{
		Region:      a.opts.Region,
		Label:       name,
		Description: "Created by Mantle",
		CloudInit:   true,
		Tags:        &tags,
		Image:       f,
	}
	plog.Infof("Uploading image file: %s", file)
	img, err := a.client.UploadImage(ctx, opts)
	if err != nil {
		return "", fmt.Errorf("uploading image file: %w", err)
	}

	ID := img.ID
	plog.Infof("Image uploaded with ID: %s - waiting to be ready", ID)

	if err := util.WaitUntilReady(5*time.Minute, 5*time.Second, func() (bool, error) {
		i, err := a.client.GetImage(ctx, ID)
		if err != nil {
			return false, fmt.Errorf("getting image: %w", err)
		}

		return i.Status == linodego.ImageStatusAvailable, nil
	}); err != nil {
		return "", fmt.Errorf("getting image ready: %w", err)
	}

	return ID, nil
}
