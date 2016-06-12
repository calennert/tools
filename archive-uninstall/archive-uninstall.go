package main

import (
	"archive/tar"
	"archive/zip"
	"crypto/sha256"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"

	"github.com/alecthomas/kingpin"
	"github.com/calennert/archive"
	"github.com/fatih/color"
)

type colorFunc func(a ...interface{}) string

type reverseStringSlice []string

func (p reverseStringSlice) Len() int           { return len(p) }
func (p reverseStringSlice) Less(i, j int) bool { return p[j] < p[i] }
func (p reverseStringSlice) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }

type verifyCallback func(file *os.File) (bool, error)

const (
	fmtFilename string = "File: %s\n"
	fmtDirname  string = "Dir : %s\n"
	fmtExists   string = "      Exists : %s\n"
	fmtEmpty    string = "      Empty  : %s\n"
	fmtRemoved  string = "      Removed: %s%s\n"

	reasonVerificationFailure string = " (file failed verification)"
	reasonFileNotFound        string = " (file not found in target directory)"
	reasonDirNotFound         string = " (directory does not found)"

	errTypeDetermination string = "Unable to determine the file's archive type. Specify with the -type argument."
	errUnrecognizedType  string = "The type specified with the -type argument was not recognized."
	errWalkError         string = "An error occurred while walking the archive."
	errDirRemovalError   string = "An error occurred while attempting to remove a directory."
)

var (
	/* CLI variables */
	app             = kingpin.New("archive-uninstall", "A tool to remove archive contents from a target directory.")
	verbose         = app.Flag("verbose", "Enable verbose mode.").Short('v').Bool()
	dryRun          = app.Flag("dry-run", "Enable dry run mode. Nothing will be removed from target directory.").Bool()
	removeDirs      = app.Flag("remove-dirs", "Enables removal of empty directories.").Bool()
	verify          = app.Flag("verify", "Only remove verified files.").Bool()
	noColor         = app.Flag("no-color", "Disable color output in verbose mode. ").Bool()
	archiveTypeText = app.Flag("type", "The archive type. Determined from archive filename, if not specified.").HintOptions(".tar", ".tar.bz2", ".tar.gz", ".tar.xz", ".zip").Short('t').String()
	archiveFile     = app.Arg("archive filename", "The filename of the archive that will be compared to the target directory.").Required().File()
	targetDir       = app.Arg("target directory", "The target directory from which to remove files.").Required().File()

	/* other variables */
	archiveType archive.Type
	directories []string
	colorFuncs  map[string]colorFunc
)

func init() {
	colorFuncs = make(map[string]colorFunc)
	colorFuncs["green"] = color.New(color.FgGreen).SprintFunc()
	colorFuncs["red"] = color.New(color.FgRed).SprintFunc()
	colorFuncs["cyan"] = color.New(color.FgCyan).SprintFunc()

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

func cyan(value string) string {
	if *noColor {
		return value
	}
	return colorFuncs["cyan"](value)
}

func green(value string) string {
	if *noColor {
		return value
	}
	return colorFuncs["green"](value)
}

func red(value string) string {
	if *noColor {
		return value
	}
	return colorFuncs["red"](value)
}

func printCaption(value bool) string {
	if value {
		return green("Yes")
	}

	return red("No")
}

func removeFromTargetDir(filename string, targetPath *os.File, verifyFunc verifyCallback) error {
	path := filepath.Join(targetPath.Name(), filename)
	file, _ := os.Open(path)

	if *verbose {
		fmt.Printf(fmtFilename, cyan(filename))
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
		addDirectory(header.Name)
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
		addDirectory(file.Name)
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

func addDirectory(dirName string) {
	directories = append(directories, dirName)
}

func removeEmptyDirectories() {
	dirs := reverseStringSlice(directories)
	sort.Sort(dirs)

	for _, d := range dirs {
		path := filepath.Join((*targetDir).Name(), d)

		if *verbose {
			fmt.Printf(fmtDirname, cyan(path))
		}

		count := 0
		walkFunc := func(path string, info os.FileInfo, err error) error {
			count++
			return nil
		}
		if err := filepath.Walk(path, walkFunc); err != nil {
			return
		}

		exists := true
		empty := false
		removed := false
		reason := ""
		if count == 1 {
			empty = true
			if !*dryRun {
				if err := os.Remove(path); err != nil {
					if !os.IsNotExist(err) {
						app.FatalIfError(err, errDirRemovalError)
					} else {
						exists = false
						reason = reasonDirNotFound
					}
				} else {
					removed = true
				}
			}
		}

		if *verbose {
			fmt.Printf(fmtExists, printCaption(exists))
			fmt.Printf(fmtEmpty, printCaption(empty))
			fmt.Printf(fmtRemoved, printCaption(removed), reason)
		}
	}
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

	if *removeDirs {
		removeEmptyDirectories()
	}
}
