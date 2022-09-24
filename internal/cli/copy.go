package cli

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"pt/internal/file"
	"pt/internal/fileutil"
	"pt/internal/model"
	"strings"

	"pt/internal/logwrap"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/volatiletech/sqlboiler/v4/queries/qm"
	"golang.org/x/sync/errgroup"
)

type result struct {
	f                   file.File
	destinationFilePath string
	err                 error
}

func copier(ctx context.Context, db *sql.DB, destinationDir string, deviceNames map[string][]string, c chan file.File, checkDuplicatesDB bool) error {
	logger := logwrap.Get("pt")
	if logger == nil {
		return fmt.Errorf("Unable to get pt logger")
	}
	for f := range c {
		supported, err := file.IsSupportedFileType(f.OriginalFilePath)
		if err != nil && err != fileutil.ErrUnknownFileType {
			return err
		}
		if !supported {
			continue
		}

		if checkDuplicatesDB {
			fileHash, err := fileutil.GetFileHash(f.OriginalFilePath)
			if err != nil {
				return err
			}
			hashes, err := model.Hashes(qm.Where("hash = ?", fileHash)).All(ctx, db)
			if err != nil {
				return err
			}
			if len(hashes) >= 1 {
				logger.Debug(fmt.Sprintf("Hash found in DB: %s, %s, %d", f.OriginalFilePath, fileHash, len(hashes)))
				continue
			}
		}

		deviceName := file.DeviceName(deviceNames, f.OriginalFilePath)
		destinationFilePath := f.DestinationFilePath(destinationDir, deviceName, f.Timestamp())
		err = fileutil.Copy(f.OriginalFilePath, destinationFilePath, 2048*1024)
		if err != nil && err != fileutil.ErrFileExists {
			return err
		}
	}
	return nil
}

func copyCmd(cli *cli) *cobra.Command {
	var flags struct {
		sourceDir         string
		destinationDir    string
		logLevel          string
		checkDuplicatesDB bool
	}
	var cmd = &cobra.Command{
		Use: "copy",
		PreRun: func(cmd *cobra.Command, args []string) {
			_ = viper.BindPFlag("source-file", cmd.Flags().Lookup("source-file"))
			_ = viper.BindPFlag("source-dir", cmd.Flags().Lookup("source-dir"))
			_ = viper.BindPFlag("destination-dir", cmd.Flags().Lookup("destination-dir"))
			_ = viper.BindPFlag("log-level", cmd.Flags().Lookup("log-level"))
			_ = viper.BindPFlag("check-duplicates-db", cmd.Flags().Lookup("check-duplicates-db"))
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

			checkDuplicatesDB := flags.checkDuplicatesDB

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
					case files <- file.NewFile(p, info.ModTime()):
					case <-ctx.Done():
						return ctx.Err()
					}
					return nil
				})
			})

			const numCopiers = 4
			for i := 0; i < numCopiers; i++ {
				g.Go(func() error {
					return copier(ctx, db, destinationDir, cli.config.DeviceNames, files, checkDuplicatesDB)
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
	cmd.Flags().BoolVar(&flags.checkDuplicatesDB, "check-duplicates-db", false, "Check duplicates within the DB and if found, don't clobber")
	return cmd
}
