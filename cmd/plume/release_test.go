// Copyright 2026 The Flatcar Maintainers
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"testing"
	"time"

	compute "google.golang.org/api/compute/v1"
)

func TestGceImagesToKeep(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	cutoff := now.AddDate(0, -gceRetentionMonths, 0)

	// build returns images aged the given number of months, newest first.
	build := func(monthsOld ...int) []*compute.Image {
		imgs := make([]*compute.Image, len(monthsOld))
		for i, m := range monthsOld {
			imgs[i] = &compute.Image{
				Name:              "flatcar-stable-image",
				CreationTimestamp: now.AddDate(0, -m, 0).Format(time.RFC3339),
			}
		}
		return imgs
	}

	tests := []struct {
		name   string
		images []*compute.Image
		limit  int
		want   int
	}{
		{
			// All images are older than the retention window and the channel
			// limit is tiny, so the 10-image floor keeps things from being
			// pruned down to almost nothing.
			name:   "small limit still keeps at least 10",
			images: build(10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24),
			limit:  3,
			want:   gceMinKeptImages,
		},
		{
			// 15 images all within the last 9 months must all be kept, even
			// though the limit is much smaller.
			name:   "recent images are all kept",
			images: build(0, 1, 2, 3, 4, 5, 6, 7, 8, 8, 8, 8, 8, 8, 8),
			limit:  3,
			want:   15,
		},
		{
			// A generous limit is honoured when it is the largest floor.
			name:   "explicit limit wins when largest",
			images: build(10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20),
			limit:  20,
			want:   20,
		},
		{
			// Fewer images than the floor: the helper still reports the floor;
			// the caller only deletes when len(images) > keep, so nothing is
			// pruned.
			name:   "fewer than the floor returns the floor",
			images: build(12, 13, 14),
			limit:  0,
			want:   gceMinKeptImages,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := gceImagesToKeep(tt.images, tt.limit, cutoff)
			if got != tt.want {
				t.Errorf("gceImagesToKeep() = %d, want %d", got, tt.want)
			}
		})
	}
}
