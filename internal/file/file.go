package file

import (
	"fmt"
	"io/fs"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"pt/internal/fileutil"
	"pt/internal/logwrap"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/dsoprea/go-exif/v3"
	exifcommon "github.com/dsoprea/go-exif/v3/common"

	"github.com/h2non/filetype"
	"github.com/h2non/filetype/types"
)

// File ...
type File struct {
	// OriginalFilePath is the absolute file path of where the file was copied from.
	OriginalFilePath string

	FileInfo fs.FileInfo

	// ExifData is a map of relevant exif data.
	exifData map[string]string

	timestamp      time.Time
	hash           string
	filenameSuffix string
	logger         *logwrap.LogWrap
}

// Option ...
type Option func(f File) File

// WithFilenameSuffix is an Option that will add a suffix to the OriginalFilePath.
func WithFilenameSuffix(suffix string) Option {
	return func(f File) File {
		f.filenameSuffix = suffix
		return f
	}
}


// NewFile ...
func NewFile(originalFilePath string, fileInfo fs.FileInfo) File {
	logger := logwrap.Get("pt")
	return File{OriginalFilePath: originalFilePath, FileInfo: fileInfo, logger: logger}
}

// Timestamp returns the creation date of the timestamp. If it's unable to find
// the creation date from the files metadata, it will fall back to the files
// modification time.
func (f File) Timestamp() time.Time {
	if !f.timestamp.IsZero() {
		return f.timestamp
	}

	creationDate := time.Time{}

	switch {
	case f.isImage():
		exifData, err := f.getExifData()
		if err != nil {
			break
		}

		t1, ok := exifData["DateTimeOriginal"]
		if !ok {
			break
		}

		dateTimeOriginal, err := exifcommon.ParseExifFullTimestamp(t1)
		if err != nil {
			break
		}

		t2, ok := exifData["SubSecTimeOriginal"]
		if !ok {
			// If there is no SubSecTimeOriginal exif data fall back to 000 milliseconds
			t2 = "000"
		}

		ms, err := strconv.Atoi(t2)
		if err != nil {
			break
		}

		creationDate = dateTimeOriginal.Add(time.Duration(ms) * time.Millisecond)
	case f.isVideo():
		fh, err := os.Open(f.OriginalFilePath)
		if err != nil {
			creationDate = f.FileInfo.ModTime()
			break
		}
		defer fh.Close()
		creationDate, err = fileutil.GetVideoCreationTimeMetadata(fh)
		if err != nil {
			creationDate = f.FileInfo.ModTime()
		}
	}

	if creationDate.IsZero() {
		creationDate = f.FileInfo.ModTime()
	}

	f.timestamp = creationDate

	return creationDate
}

func (f File) isVideo() bool {
	buf, _ := ioutil.ReadFile(f.OriginalFilePath)
	return filetype.IsVideo(buf)
}

func (f File) isImage() bool {
	buf, _ := ioutil.ReadFile(f.OriginalFilePath)
	return filetype.IsImage(buf)
}

// GetExifData returns the exif data for the file if it exists.
func (f File) getExifData() (map[string]string, error) {
	if len(f.exifData) >= 1 {
		return f.exifData, nil
	}

	rawExif, err := exif.SearchFileAndExtractExif(f.OriginalFilePath)
	if err != nil {
		return f.exifData, err
	}

	entries, _, err := exif.GetFlatExifData(rawExif, nil)
	if err != nil {
		return f.exifData, err
	}

	m := map[string]string{}

	for _, i := range entries {
		m[i.TagName] = fmt.Sprintf("%s", i.Value)
	}

	f.exifData = m

	return f.exifData, nil
}

// DirName ...
func (f File) DirName() string {
	fileDir := filepath.Dir(f.OriginalFilePath)
	dirs := strings.Split(fileDir, string(os.PathSeparator))
	return dirs[len(dirs)-1]
}

// Hash ...
func (f File) Hash() (string, error) {
	return fileutil.GetFileHash(f.OriginalFilePath)
}

// DestinationFilePath returns the file path of the final destination of the file. The path includes the album except if the album is "recents" which is removed from the path.
func (f File) DestinationFilePath(destinationDir string, deviceName string, creationDate time.Time, opts ...Option) string {

	for _, opt := range opts {
		f = opt(f)
	}

	album := f.DirName()

	// Rewrite album to a value in this map if the dir matches a key. This
	// allows to copy a DCIM directory into the same photosync structure.
	albumDirMapping := map[string]string{
		`^1\d{2}APPLE$`: "Recents",
		`^1\d{2}CANON$`: "Recents",
		`^\d{4}-\d{2}-\d{2}$`: "Recents",
	}

	for k, v := range albumDirMapping {
		r := regexp.MustCompile(k)
		if r.MatchString(album) {
			album = v
			break
		}
	}

	p := path.Join(
		destinationDir,
		fmt.Sprintf("%d", creationDate.Year()),
		fmt.Sprintf("%02d", creationDate.Month()),
		deviceName,
		album,
		strings.Replace(creationDate.Format("20060102-150405.000"), ".", "", 1),
	)

	if f.filenameSuffix != "" {
		p = fmt.Sprintf("%s-%s", p, f.filenameSuffix)
	}

	p = fmt.Sprintf("%s%s", p, path.Ext(f.OriginalFilePath))

	return p
}

// IsSupportedFileType checks if the file is supported.  supportedTypes
// contains the list of supported file types.
func IsSupportedFileType(originalFilePath string) (bool, error) {
	f, err := os.Open(originalFilePath)
	if err != nil {
		return false, err
	}
	defer f.Close()

	supportedTypes := []types.Type{
		types.Get("jpg"),
		types.Get("png"),
		types.Get("heif"),
		types.Get("cr2"),
		types.Get("mov"),
	}

	head := make([]byte, 261)
	f.Read(head)
	for _, i := range supportedTypes {
		if filetype.IsType(head, i) {
			return true, nil
		}
	}

	return false, nil
}

// DirName returns the dirname of f.
func DirName(f string) string {
	fileDir := filepath.Dir(f)
	dirs := strings.Split(fileDir, string(os.PathSeparator))
	return dirs[len(dirs)-1]
}

// DeviceName ...
func DeviceName(deviceNames map[string][]string, originalFilePath string) string {
	// Check if deviceNames paths are absolute paths.
	for k, v := range deviceNames {
		for _, ii := range v {
			if strings.HasPrefix(ii, "/") && strings.HasPrefix(originalFilePath, ii) {
				return k
			}
		}
	}

	a := strings.Split(originalFilePath, "/")
	for _, i := range a {
		for k, v := range deviceNames {
			for _, ii := range v {
				if ii == i {
					return k
				}
			}
		}
	}
	return ""
}
