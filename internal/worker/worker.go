package worker

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"pt/internal/file"
	"pt/internal/fileutil"
	"pt/internal/logwrap"
	"strings"
)

// Copier accepts a channel of file.File and copies files sent to the channel
// to destinationDir. If checkDuplicatesDB is set, the file will be checked to
// see if it exists in the hash table and if so, the file will be skipped.
func Copier(ctx context.Context, destinationDir string, deviceNames map[string][]string, c chan file.File, checkDuplicates bool) error {
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

		deviceName := file.DeviceName(deviceNames, f.OriginalFilePath)
		destinationFilePath := f.DestinationFilePath(destinationDir, deviceName, f.Timestamp())

		// Find files within destinationDir/deviceName that match the
		// destinationFilePath. If there are duplicates, check the size of the
		// file and if they match, skip copying this file.
		duplicateFound := false
		if checkDuplicates {

			// yearMonth is a slice of the year and month from the
			// destinationFilePath. This will be used by monthDir to build the
			// directory path where filepath.Walk() will scan for duplicate
			// files.
			yearMonth := strings.Split(
				strings.TrimPrefix(destinationFilePath, destinationDir),
				string(os.PathSeparator))[1:3]
			monthDir := filepath.Join(destinationDir, yearMonth[0], yearMonth[1])

			// Walk the monthDir path looking for files that match the
			// destinationFilePath filename and size of the file to copy. If a
			// match is found, set duplicateFound and break out of
			// filepath.Walk().
			err := filepath.Walk(monthDir, func(p string, info os.FileInfo, err error) error {
				if !info.Mode().IsRegular() {
					return nil
				}
				if filepath.Base(p) == filepath.Base(destinationFilePath) {
					if info.Size() == f.FileInfo.Size() {
						duplicateFound = true
						logger.Debug(fmt.Sprintf("duplicate found, not copying: %s, %s", f.OriginalFilePath, p))
						return nil
					}
				}
				return nil
			})
			if err != nil {
				return err
			}
		}

		// Copy the file only if a duplicate is not found (and check for
		// duplicates has been set).
		if !duplicateFound {
			err = fileutil.Copy(f.OriginalFilePath, destinationFilePath, 2048*1024)
			logger.Debug(fmt.Sprintf("copied %s to %s: %v", f.OriginalFilePath, destinationFilePath, err))
			if err != nil && err != fileutil.ErrFileExists {
				return err
			}
		}
	}
	return nil
}
