package archive

import (
	"archive/tar"
	"archive/zip"
	"compress/bzip2"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"

	"xi2.org/x/xz"
)

// Type defines the archive types that can be processed
// in this package.
type Type uint

// Valid archive types.
const (
	Tar Type = iota + 1
	TarBz2
	TarGz
	TarXz
	Zip
)

// String returns a string representation of the archive type.
func (t Type) String() (result string) {
	switch t {
	case Tar:
		result = "Tar"
	case TarBz2:
		result = "TarBz2"
	case TarGz:
		result = "TarGz"
	case TarXz:
		result = "TarXz"
	case Zip:
		result = "Zip"
	}
	return
}

// Format strings for various errors
const (
	fmtErrArchiveOpen string = "archive: failed to open archive: %v"
	fmtErrNewGzReader string = "archive: failed to gz reader: %v"
	fmtErrNewXzReader string = "archive: failed to xz reader: %v"
)

// ErrUnknownType is returned by DetermineType if the provided filename
// fails to match an archive type supported by this package.
var ErrUnknownType = errors.New("archive: unable to determine type")

func init() {
	typeInfoMap = make(map[Type]typeInfo)

	typeInfoMap[Tar] = typeInfo{extensions: []string{".tar"}}
	typeInfoMap[TarBz2] = typeInfo{extensions: []string{".tar.bz2", ".tar.bzip2", ".tbz", ".tbz2"}}
	typeInfoMap[TarGz] = typeInfo{extensions: []string{".tar.gz", ".tgz"}}
	typeInfoMap[TarXz] = typeInfo{extensions: []string{".tar.xz", ".txz"}}
	typeInfoMap[Zip] = typeInfo{extensions: []string{".zip"}}
}

// DetermineType identifies the archive file type based on the extensions present in the
// filename. Files with the ".tar" extension will be identified as Tar. Files
// with ".tar.bz2", ".tar.bzip2", ".tbz", or ".tbz2" extensions will be identified
// as TarBz2. Files with ".tar.gz" or ".tgz" extensions will be identified
// as TarGz. Files with ".tar.xz" or ".txz" extensions will be identified
// as TarXz. Files with the ".zip" extension will be identified as Zip. Anything
// else returns 0 and a non-nil error.
func DetermineType(filename string) (Type, error) {
	f := strings.ToLower(filename)

	for archType, info := range typeInfoMap {
		if info.regex == "" {
			info.regex = makeRegex(info.extensions)
		}

		m := regexp.MustCompile(info.regex).FindStringSubmatch(f)
		if len(m) != 0 {
			return archType, nil
		}
	}

	return 0, ErrUnknownType
}

// TarCallback is the type of function called for each file or directory entry
// visited by the WalkTar functions.
type TarCallback func(*tar.Reader, *tar.Header) error

// ZipCallback is the type of function called for each file or directory entry
// visited by WalkZip.
type ZipCallback func(*zip.File) error

// WalkZip walks the contents of a zip file and invokes the callback
// function for each entry.
func WalkZip(archivePath string, callback ZipCallback) error {
	r, err := zip.OpenReader(archivePath)
	if err != nil {
		return fmt.Errorf(fmtErrArchiveOpen, err)
	}
	defer r.Close()

	for _, f := range r.File {
		if callback != nil {
			err := callback(f)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// WalkTar walks the contents of a tar file and invokes the callback
// function for each entry.
func WalkTar(archivePath string, callback TarCallback) error {
	file, err := os.Open(archivePath)
	if err != nil {
		return fmt.Errorf(fmtErrArchiveOpen, err)
	}
	defer file.Close()

	return readTar(tar.NewReader(file), callback)
}

// WalkTarBzip2 walks the contents of a bzip2-compressed tar file and invokes the
// callback function for each entry.
func WalkTarBzip2(archivePath string, callback TarCallback) error {
	file, err := os.Open(archivePath)
	if err != nil {
		return fmt.Errorf(fmtErrArchiveOpen, err)
	}
	defer file.Close()

	reader := bzip2.NewReader(file)

	return readTar(tar.NewReader(reader), callback)
}

// WalkTarGz walks the contents of a gzip-compressed tar file and invokes the
// callback function for each entry.
func WalkTarGz(archivePath string, callback TarCallback) error {
	file, err := os.Open(archivePath)
	if err != nil {
		return fmt.Errorf(fmtErrArchiveOpen, err)
	}
	defer file.Close()

	reader, err := gzip.NewReader(file)
	if err != nil {
		return fmt.Errorf(fmtErrNewGzReader, err)
	}
	defer reader.Close()

	return readTar(tar.NewReader(reader), callback)
}

// WalkTarXz walks the contents of a lzma2-compressed (xz) tar file and invokes the
// callback function for each entry.
func WalkTarXz(archivePath string, callback TarCallback) error {
	file, err := os.Open(archivePath)
	if err != nil {
		return fmt.Errorf(fmtErrArchiveOpen, err)
	}
	defer file.Close()

	reader, err := xz.NewReader(file, 0)
	if err != nil {
		return fmt.Errorf(fmtErrNewXzReader, err)
	}

	return readTar(tar.NewReader(reader), callback)
}

// Reads the tar file contents.
func readTar(reader *tar.Reader, callback TarCallback) error {
	for {
		header, err := reader.Next()
		if err == io.EOF {
			break
		} else if err != nil {
			return err
		}

		if callback != nil {
			err := callback(reader, header)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// Struct typeInfo contains information relating to a given
// archive type.
type typeInfo struct {
	extensions []string
	regex      string
}

var typeInfoMap map[Type]typeInfo

// Constructs a regular expression string from the specified extensions.
func makeRegex(extensions []string) (regex string) {
	regex = strings.Join(extensions, "|")
	regex = strings.Replace(regex, ".", `\.`, -1)
	regex = fmt.Sprintf("(%s)$", regex)
	return
}
