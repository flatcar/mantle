package stackit

import (
	"fmt"
	"github.com/spf13/cobra"
)

var (
	cmdCreate = &cobra.Command{
		Use:   "create-image",
		Short: "Create image on STACKIT",
		Long:  "Upload an image to STACKIT",
		RunE:  runCreate,
	}

	name  string
	board string
	file  string
)

func init() {
	STACKIT.AddCommand(cmdCreate)
	cmdCreate.Flags().StringVar(&file, "file", "flatcar_production_stackit_image.img", "path to local Flatcar image (.img)")
	cmdCreate.Flags().StringVar(&name, "name", "flatcar-kola-test", "image name")
	cmdCreate.Flags().StringVar(&board, "board", "amd64-usr", "board of the image")

}

func runCreate(cmd *cobra.Command, args []string) error {
	id, err := API.UploadImage(cmd.Context(), name, file, board)
	if err != nil {
		return fmt.Errorf("creating an image: %w", err)
	}
	fmt.Println(id)
	return nil
}
