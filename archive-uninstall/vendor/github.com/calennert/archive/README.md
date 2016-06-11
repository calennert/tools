# archive [![GoDoc](https://godoc.org/github.com/calennert/archive?status.svg)](https://godoc.org/github.com/calennert/archive) [![Build Status](https://travis-ci.org/calennert/archive.svg?branch=master)](https://travis-ci.org/calennert/archive) [![Coverage](http://gocover.io/_badge/github.com/calennert/archive)](http://gocover.io/github.com/calennert/archive)

Package archive is a convenience package for enumerating the contents of zip files, tar files, 
and compressed tar files. Supported archive types include: zip, tar, gzip-compressed tar,
bzip2-compressed tar, and xz-compressed tar.

## Install

```bash
go get xi2.org/x/xz
go get github.com/calennert/archive
```

## Examples

### List the contents of a .zip file

```go
func zipCallback(file *zip.File) error {
    if file.FileInfo().IsDir() {
        fmt.Printf("Dir : %s\n", file.Name)
    } else  {
        fmt.Printf("File: %s\n", file.Name)
    }

    return nil
}

func main() {
    err := archive.WalkZip("test.zip", zipCallback)
    if err != nil {
        log.Fatal(err)
    }
}

```

### Extract the contents of a .tar.xz file

```go
func tarCallback(reader *tar.Reader, header *tar.Header) error {
	if header.FileInfo().IsDir() {
		os.MkdirAll(header.Name, 0700)
		return nil
	}

	fo, err := os.Create(header.Name)
    if err != nil {
        return err
    }
    defer fo.Close()

    _, err := io.Copy(fo, reader)
    if err != nil {
        return err
    }

    return nil
}

func main() {
    err := archive.WalkTarXz("test.tar.xz", tarCallback)
    if err != nil {
        log.Fatal(err)
    }
}

```

### Determine the type of archive file

```go
func main() {
	archiveType, err := archive.DetermineType(archiveFilename)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Unable to determine the file's archive type.")
		os.Exit(1)
	}

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
		log.Fatal(err)
	}
}

```

## Credits

 * XZ compression support via xi2.org: [xz](https://xi2.org/x/xz)

## License

The MIT License (MIT) - see [`LICENSE.md`](https://github.com/calennert/archive/blob/master/LICENSE.md) for more details.