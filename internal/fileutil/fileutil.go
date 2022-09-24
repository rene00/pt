package fileutil

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"pt/internal/logwrap"
	"time"

	"github.com/h2non/filetype"
)

var (
	// ErrFileExists file already exists, don't overwrite it.
	ErrFileExists = errors.New("file exists")
)

// Copy ...
func Copy(src, dst string, BUFFERSIZE int64) error {
	logger := logwrap.Get("pt")
	if logger == nil {
		return fmt.Errorf("Failed to get logwrap")
	}

	_, err := os.Stat(dst)
	if err == nil {
		return ErrFileExists
	}

	os.MkdirAll(path.Dir(dst), os.ModePerm)
	sourceFileStat, err := os.Stat(src)
	if err != nil {
		return err
	}

	if !sourceFileStat.Mode().IsRegular() {
		return fmt.Errorf("%s is not a regular file", src)
	}

	source, err := os.Open(src)
	if err != nil {
		return err
	}
	defer source.Close()


	destination, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destination.Close()

	buf := make([]byte, BUFFERSIZE)
	for {
		n, err := source.Read(buf)
		if err != nil && err != io.EOF {
			return err
		}
		if n == 0 {
			break
		}
		if _, err := destination.Write(buf[:n]); err != nil {
			return err
		}
	}

	logger.Info(fmt.Sprintf("Copied file successfully: %s, %s", src, dst))

	return nil

}

// GetContentType ...
func GetContentType(out *os.File) (string, error) {
	buf := make([]byte, 512)
	_, err := out.Read(buf)
	if err != nil {
		return "", err
	}
	if isHEIC(buf) {
		return "image/heic", nil
	}
	contentType := http.DetectContentType(buf)
	return contentType, nil
}

// isHEIC is copied from https://perkeep.org/internal/magic/magic.go.
func isHEIC(prefix []byte) bool {
	if len(prefix) < 12 {
		return false
	}
	if string(prefix[4:12]) != "ftypheic" {
		return false
	}
	return true
}

const appleEpochAdjustment = 2082844800
const (
	movieResourceAtomType = "moov"
	movieHeaderAtomType   = "mvhd"
)

// GetVideoCreationTimeMetadata copied from https://gist.github.com/phelian/81bbb30cd78aceb05c8d467243edb217
func GetVideoCreationTimeMetadata(videoBuffer io.ReadSeeker) (time.Time, error) {
	buf := make([]byte, 8)
	for {
		// bytes 1-4 is atom size, 5-8 is type. Read atom
		if _, err := videoBuffer.Read(buf); err != nil {
			return time.Time{}, err
		}

		if bytes.Equal(buf[4:8], []byte(movieResourceAtomType)) {
			break // found it!
		}

		atomSize := binary.BigEndian.Uint32(buf) // check size of atom
		videoBuffer.Seek(int64(atomSize)-8, 1)   // jump over data and set seeker at beginning	of next atom
	}

	// read next atom
	if _, err := videoBuffer.Read(buf); err != nil {
		return time.Time{}, err
	}

	atomType := string(buf[4:8]) // skip size and read type
	switch atomType {
	case movieHeaderAtomType:
		// read next atom
		if _, err := videoBuffer.Read(buf); err != nil {
			return time.Time{}, err
		}

		// byte 1 is version, byte 2-4 is flags, 5-8 is creation time
		appleEpoch := int64(binary.BigEndian.Uint32(buf[4:])) // read creation time
		return time.Unix(appleEpoch-appleEpochAdjustment, 0).Local(), nil
	default:
		return time.Time{}, fmt.Errorf("did not find movie header atom")
	}
}

var (
	// ErrUnknownFileType is set when filetype.Match() returns an unknown filetype.
	ErrUnknownFileType = errors.New("unknown file type")
)

// GetFileType returns extension and mime value for given file.
func GetFileType(p string) (string, string, error) {
	f, err := os.Open(p)
	if err != nil {
		return "", "", err
	}
	defer f.Close()

	head := make([]byte, 261)
	f.Read(head)
	
	kind, err := filetype.Match(head)
	if kind == filetype.Unknown {
		return "", "", ErrUnknownFileType
	}

	return kind.Extension, kind.MIME.Value, nil
}

// GetFileHash returns a string hash of a file.
func GetFileHash(p string) (string, error) {
	fh, err := os.Open(p)
	if err != nil {
		return "", err
	}
	defer fh.Close()

	h := sha256.New()
	if _, err := io.Copy(h, fh); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

