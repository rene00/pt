package cli

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path"
	"path/filepath"
	"pt/internal/file"
	"pt/internal/fileutil"
	"pt/internal/model"
	"strings"

	"github.com/mattn/go-sqlite3"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/volatiletech/sqlboiler/v4/boil"
	"golang.org/x/sync/errgroup"
)

type scanFile struct {
	FilePath string
}

func hasher(ctx context.Context, db *sql.DB, destinationDir string, c <-chan scanFile) error {
	for f := range c {
		fileSupported, err := file.IsSupportedFileType(f.FilePath)
		if err != nil && err != fileutil.ErrUnknownFileType {
			return err
		}
		if !fileSupported {
			continue
		}

		hash := model.Hash{Filepath: strings.TrimLeft(f.FilePath, destinationDir)}
		hash.Hash, err = fileutil.GetFileHash(f.FilePath)
		if err != nil {
			return err
		}

		if err = hash.Insert(ctx, db, boil.Infer()); err != nil {
			var sqliteErr sqlite3.Error
			if errors.As(err, &sqliteErr) {
				if sqliteErr.Code != sqlite3.ErrConstraint {
					return err
				}
			} else {
				return err
			}
		}
	}

	return nil
}

func scanCmd(cli *cli) *cobra.Command {
	var flags struct {
		destinationDir string
	}
	var cmd = &cobra.Command{
		Use: "scan",
		PreRun: func(cmd *cobra.Command, args []string) {
			_ = viper.BindPFlag("destination-dir", cmd.Flags().Lookup("destination-dir"))
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			db, err := sql.Open("sqlite3", cli.config.DBFile)
			if err != nil {
				return err
			}
			defer db.Close()

			destinationDir := cli.config.DestinationDir
			if flags.destinationDir != "" {
				destinationDir = flags.destinationDir
			}

			g, ctx := errgroup.WithContext(cmd.Context())
			c := make(chan scanFile)

			g.Go(func() error {
				defer close(c)
				return filepath.Walk(destinationDir, func(p string, info os.FileInfo, err error) error {
					if !info.Mode().IsRegular() {
						return nil
					}
					if strings.HasPrefix(path.Base(p), ".") {
						return nil
					}

					select {
					case c <- scanFile{p}:
					case <-ctx.Done():
						return ctx.Err()
					}
					return nil
				})
			})

			const numHashers = 4
			for i := 0; i < numHashers; i++ {
				g.Go(func() error {
					return hasher(ctx, db, destinationDir, c)
				})
			}

			// End of pipeline.
			if err := g.Wait(); err != nil {
				return err
			}

			return nil
		},
	}
	cmd.Flags().StringVar(&flags.destinationDir, "destination-dir", "", "Destination directory")
	return cmd
}
