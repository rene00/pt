package cli

import (
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/sync/errgroup"
)

func deleter(c <- chan string) {
	// {filename: ext}
	type imageFile struct {
		filePath string
		fileExt string
	}
	imageFiles := map[string]imageFile{}
	for file := range c {
		filepathWithoutExt := strings.TrimSuffix(file, filepath.Ext(file))
		if existingImageFile, exists := imageFiles[filepathWithoutExt]; exists {
			existingExt := existingImageFile.fileExt
			if strings.ToLower(existingExt) == ".cr2" {
				fmt.Printf("deleting %s\n", existingImageFile.filePath)
				err := os.Remove(existingImageFile.filePath)
				if err != nil {
					log.Fatal(fmt.Sprintf("Failed deleting %s: %v", existingImageFile.filePath, err))
				}
			} else {
				fmt.Printf("need to delete %s,%s\n", existingImageFile.filePath, file)
			}
			continue
		}
		imageFiles[filepathWithoutExt] = imageFile{file, filepath.Ext(file)}
	}
}

func cr2DupeCmd(cli *cli) *cobra.Command {
	var flags struct {
		imageDir  string
	}
	var cmd = &cobra.Command{
		Use: "cr2dupe",
		PreRun: func(cmd *cobra.Command, args []string) {
			_ = viper.BindPFlag("image-dir", cmd.Flags().Lookup("image-dir"))
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			g, ctx := errgroup.WithContext(cmd.Context())
			c := make(chan string)
			g.Go(func() error {
				defer close(c)
				return filepath.Walk(flags.imageDir, func(p string, info os.FileInfo, err error) error {
					if !info.Mode().IsRegular() {
						return nil
					}

					if strings.HasPrefix(path.Base(p), ".") {
						return nil
					}

					if info.Size() == 0 {
						return nil
					}

					switch strings.ToLower(filepath.Ext(p)) {
					case ".jpg", ".jpeg", ".cr2":
					default:
						return nil
					}

					select {
					case c <- p:
					case <-ctx.Done():
						return ctx.Err()
					}
					return nil
				})
			})

			deleter(c)

			// End of pipeline.
			if err := g.Wait(); err != nil {
				return err
			}

			return nil
		},
	}
	cmd.Flags().StringVar(&flags.imageDir, "image-dir", "", "Image directory")
	cmd.MarkFlagRequired("image-dir")

	return cmd
}
