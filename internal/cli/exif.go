package cli

import (
	"fmt"
	"os"
	"pt/internal/fileutil"

	"github.com/dsoprea/go-exif/v3"
	pngstructure "github.com/dsoprea/go-png-image-structure/v2"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func exifCmd(cli *cli) *cobra.Command {
	var flags struct {
		sourceFile string
	}
	var cmd = &cobra.Command{
		Use: "exif",
		PreRun: func(cmd *cobra.Command, args []string) {
			_ = viper.BindPFlag("source-file", cmd.Flags().Lookup("source-file"))
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			fh, err := os.Open(flags.sourceFile)
			if err != nil {
				return err
			}

			contentType, err := fileutil.GetContentType(fh)
			if err != nil {
				return err
			}

			fmt.Printf("DEBUG:%s\n", contentType)

			switch contentType {
			case "image/jpeg", "image/heic":
				rawExif, err := exif.SearchFileAndExtractExif(flags.sourceFile)
				if err != nil {
					return err
				}
				entries, _, err := exif.GetFlatExifData(rawExif, nil)
				if err != nil {
					return err
				}

				for _, i := range entries {
					fmt.Printf("NAME=[%s] VALUE=[%s]\n", i.TagName, i.Formatted)
				}
			case "image/png":
				pmp := pngstructure.NewPngMediaParser()

				intfc, err := pmp.ParseFile(flags.sourceFile)
				if err != nil {
					return fmt.Errorf("DEBUG2:%w", err)
				}

				cs := intfc.(*pngstructure.ChunkSlice)
				e, err := cs.FindExif()
				if err != nil {
					return fmt.Errorf("DEBUG3:%w", err)
				}
				fmt.Printf("%#v\n", e)

			}

			return nil
		},
	}
	cmd.Flags().StringVar(&flags.sourceFile, "source-file", "", "Source file")
	return cmd
}
