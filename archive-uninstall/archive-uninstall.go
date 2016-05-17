package main

import (
	"archive/tar"
	"archive/zip"
	"crypto/sha256"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"

	"github.com/alecthomas/kingpin"
	"github.com/calennert/archive"
	"github.com/fatih/color"
)

const (
	fmtFilename string = "File: %s\n"
	fmtExists   string = "      Exists : %s\n"
	fmtRemoved  string = "      Removed: %s%s\n"

	reasonVerificationFailure = " (file failed verification)"
	reasonFileNotFound        = " (file not found in target directory)"

	errTypeDetermination = "Unable to determine the file's archive type. Specify with the -type argument."
	errUnrecognizedType  = "The type specified with the -type argument was not recognized."
	errWalkError         = "An error occurred while walking the archive."
)

var (
	/* CLI variables */
	app             = kingpin.New("archive-uninstall", "A tool to remove archive contents from a target directory.")
	verbose         = app.Flag("verbose", "Enable verbose mode.").Short('v').Bool()
	dryRun          = app.Flag("dry-run", "Enable dry run mode. No files will be removed from target directory.").Bool()
	verify          = app.Flag("verify", "Only remove verified files.").Bool()
	noColor         = app.Flag("no-color", "Disable color output in verbose mode. ").Bool()
	archiveTypeText = app.Flag("type", "The archive type. Determined from archive filename, if not specified.").HintOptions(".tar", ".tar.bz2", ".tar.gz", ".tar.xz", ".zip").Short('t').String()
	archiveFile     = app.Arg("archive filename", "The filename of the archive that will be compared to the target directory.").Required().File()
	targetDir       = app.Arg("target directory", "The target directory from which to remove files.").Required().File()

	/* color output functions */
	green, red, cyan func(a ...interface{}) string

	/* other variables */
	archiveType archive.Type
)

func init() {
	green = color.New(color.FgGreen).SprintFunc()
	red = color.New(color.FgRed).SprintFunc()
	cyan = color.New(color.FgCyan).SprintFunc()

	app.Author("https://github.com/calennert")
	app.Version("1.0")
	app.UsageTemplate(CustomUsageTemplate)
	kingpin.MustParse(app.Parse(os.Args[1:]))

	var err error
	if *archiveTypeText == "" {
		archiveType, err = archive.DetermineType((*archiveFile).Name())
		if err != nil {
			app.Errorf(errTypeDetermination)
			os.Exit(10)
		}
	} else {
		archiveType, err = archive.DetermineType(*archiveTypeText)
		if err != nil {
			app.Errorf(errUnrecognizedType)
			os.Exit(12)
		}
	}
}

func printCaption(value bool) string {
	if *noColor {
		if value {
			return "Yes"
		} else {
			return "No"
		}
	} else {
		if value {
			return green("Yes")
		} else {
			return red("No")
		}
	}
}

type verifyCallback func(file *os.File) (bool, error)

func removeFromTargetDir(filename string, targetPath *os.File, verifyFunc verifyCallback) error {
	path := path.Join(targetPath.Name(), filename)
	file, _ := os.Open(path)

	if *verbose {
		if *noColor {
			fmt.Printf(fmtFilename, filename)
		} else {
			fmt.Printf(fmtFilename, cyan(filename))
		}
		fmt.Printf(fmtExists, printCaption((file != nil)))
	}

	removed := false
	reason := ""

	if file != nil {
		verified, err := verifyFunc(file)
		if err != nil {
			return err
		}

		if verified {
			if !*dryRun {
				err = os.Remove(file.Name())
				if err != nil {
					return err
				}
				removed = true
			}
		} else {
			reason = reasonVerificationFailure
		}
	} else {
		reason = reasonFileNotFound
	}

	if *verbose {
		fmt.Printf(fmtRemoved, printCaption(removed), reason)
	}
	return nil
}

func tarCallback(reader *tar.Reader, header *tar.Header) error {
	if header.FileInfo().IsDir() {
		return nil
	}

	verifyFunc := func(file *os.File) (bool, error) {
		if *verify {
			data, err := ioutil.ReadAll(file)
			if err != nil {
				return false, err
			}
			targetSha256 := sha256.Sum256(data)

			data, err = ioutil.ReadAll(reader)
			if err != nil {
				return false, err
			}
			archiveSha256 := sha256.Sum256(data)

			return (targetSha256 == archiveSha256), nil
		}
		return true, nil
	}

	return removeFromTargetDir(header.Name, *targetDir, verifyFunc)
}

func zipCallback(file *zip.File) error {
	if file.FileInfo().IsDir() {
		return nil
	}

	verifyFunc := func(targetFile *os.File) (bool, error) {
		if *verify {
			data, err := ioutil.ReadAll(targetFile)
			if err != nil {
				return false, err
			}
			targetSha256 := sha256.Sum256(data)

			var reader io.ReadCloser
			reader, err = file.Open()
			if err != nil {
				return false, err
			}
			data, err = ioutil.ReadAll(reader)
			if err != nil {
				return false, err
			}
			archiveSha256 := sha256.Sum256(data)

			return (targetSha256 == archiveSha256), nil
		}
		return true, nil
	}

	return removeFromTargetDir(file.Name, *targetDir, verifyFunc)
}

func main() {
	var err error
	archiveFilename := (*archiveFile).Name()

	switch archiveType {
	case archive.Tar:
		err = archive.WalkTar(archiveFilename, tarCallback)
	case archive.TarBz2:
		err = archive.WalkTarBzip2(archiveFilename, tarCallback)
	case archive.TarGz:
		err = archive.WalkTarGz(archiveFilename, tarCallback)
	case archive.TarXz:
		err = archive.WalkTarXz(archiveFilename, tarCallback)
	case archive.Zip:
		err = archive.WalkZip(archiveFilename, zipCallback)
	}

	if err != nil {
		app.FatalIfError(err, errWalkError)
	}
}
