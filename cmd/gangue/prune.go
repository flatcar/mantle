// Copyright 2020 Kinvolk GmbH
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
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"cloud.google.com/go/storage"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"

	"github.com/spf13/cobra"
)

var (
	prune = &cobra.Command{
		Use:   "prune bucket-name path-to-analyze",
		Short: "Delete old files from Google Cloud Storage",
		Long:  "Recursively delete files in GCS bucket under path-to-analyze, if they are older than --days days.",
		Run:   runPrune,
	}

	days      int
	whitelist string
)

func init() {
	prune.PersistentFlags().IntVar(&days, "days", 30, "Minimum age in days for files to get deleted")
	prune.PersistentFlags().StringVar(&whitelist, "whitelist", "developer/", "Whitelist for path prefixes to consider")
	root.AddCommand(prune)
}

func runPrune(cmd *cobra.Command, args []string) {

	if len(args) != 2 {
		fmt.Fprintf(os.Stderr, "bucket-name and path-to-analyze are required.\n")
		os.Exit(1)
	}

	if jsonKeyFile == "" {
		fmt.Fprintf(os.Stderr, "The --json-key filename is required.\n")
		os.Exit(1)
	}

	bucketName := args[0]
	pathPrefix := args[1]

	if !strings.HasPrefix(pathPrefix, whitelist) {
		fmt.Fprintf(os.Stderr, "The provided prefix (%s) isn't whitelisted (%s).\n", pathPrefix, whitelist)
		os.Exit(1)
	}

	// Connect to GCS.
	ctx := context.Background()
	client, err := storage.NewClient(ctx, option.WithCredentialsFile(jsonKeyFile))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Couldn't connect to GCS: %v\n", err)
		os.Exit(1)
	}

	// Open the bucket
	bkt := client.Bucket(bucketName)

	// Iterate over the contents of the prefix in the bucket.
	cutOffDay := time.Now().AddDate(0, 0, -1*days)
	query := &storage.Query{Prefix: pathPrefix}
	it := bkt.Objects(ctx, query)
	for {
		attrs, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error iterating bucket %q: %v\n", bucketName, err)
			os.Exit(1)
		}
		if strings.HasSuffix(attrs.Name, "/") {
			fmt.Printf("Not checking %s\n", attrs.Name)
		}
		// Check the date of the object, delete it if it's older than the selected days
		if attrs.Updated.Before(cutOffDay) {
			fmt.Printf("%s is obsolete (%v) - Deleting\n", attrs.Name, attrs.Updated)
			if err := bkt.Object(attrs.Name).Delete(ctx); err != nil {
				fmt.Fprintf(os.Stderr, "Error deleting object %q: %v\n", attrs.Name, err)
				os.Exit(1)
			}
		}
	}
}
