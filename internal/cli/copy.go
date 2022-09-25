package cli

import (
	"database/sql"
	"os"
	"path"
	"path/filepath"
	"pt/internal/file"
	"pt/internal/logwrap"
	"pt/internal/worker"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/sync/errgroup"
)

type result struct {
	f                   file.File
	destinationFilePath string
	err                 error
}

func copyCmd(cli *cli) *cobra.Command {
	var flags struct {
		sourceDir       string
		destinationDir  string
		logLevel        string
		checkDuplicates bool
	}
	var cmd = &cobra.Command{
		Use: "copy",
		PreRun: func(cmd *cobra.Command, args []string) {
			_ = viper.BindPFlag("source-file", cmd.Flags().Lookup("source-file"))
			_ = viper.BindPFlag("source-dir", cmd.Flags().Lookup("source-dir"))
			_ = viper.BindPFlag("destination-dir", cmd.Flags().Lookup("destination-dir"))
			_ = viper.BindPFlag("log-level", cmd.Flags().Lookup("log-level"))
			_ = viper.BindPFlag("check-duplicates", cmd.Flags().Lookup("check-duplicates"))
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			db, err := sql.Open("sqlite3", cli.config.DBFile)
			if err != nil {
				return err
			}
			defer db.Close()

			logger := logwrap.New("pt", os.Stdout, false)
			switch flags.logLevel {
			case "info":
				logger.SetLevel(logwrap.INFO)
			case "debug":
				logger.SetLevel(logwrap.DEBUG)
			default:
				logger.SetLevel(logwrap.NONE)
			}

			if cli.debug {
				logger.SetLevel(logwrap.DEBUG)
			}

			sourceDir := cli.config.SourceDir
			if flags.sourceDir != "" {
				sourceDir = flags.sourceDir
			}

			destinationDir := cli.config.DestinationDir
			if flags.destinationDir != "" {
				destinationDir = flags.destinationDir
			}

			g, ctx := errgroup.WithContext(cmd.Context())
			files := make(chan file.File)

			g.Go(func() error {
				defer close(files)
				return filepath.Walk(sourceDir, func(p string, info os.FileInfo, err error) error {
					if !info.Mode().IsRegular() {
						return nil
					}

					if strings.HasPrefix(path.Base(p), ".") {
						return nil
					}

					if strings.HasPrefix(path.Base(p), ".") {
						return nil
					}

					if strings.ToLower(file.DirName(p)) == ".thumbnails" {
						return nil
					}

					select {
					case files <- file.NewFile(p, info):
					case <-ctx.Done():
						return ctx.Err()
					}
					return nil
				})
			})

			const numCopiers = 4
			for i := 0; i < numCopiers; i++ {
				g.Go(func() error {
					return worker.Copier(ctx, destinationDir, cli.config.DeviceNames, files, flags.checkDuplicates)
				})
			}

			if err := g.Wait(); err != nil {
				return err
			}

			return nil
		},
	}
	cmd.Flags().StringVar(&flags.sourceDir, "source-dir", "", "Source directory")
	cmd.Flags().StringVar(&flags.destinationDir, "destination-dir", "", "Destination directory")
	cmd.Flags().StringVar(&flags.logLevel, "log-level", "none", "Log level (none, info, debug)")
	cmd.Flags().BoolVar(&flags.checkDuplicates, "check-duplicates", false, "Check duplicates within the DB and if found, don't clobber")
	return cmd
}
