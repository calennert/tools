/*
Package archive is a convenience package for enumerating the contents of zip files,
tar files, and compressed tar files. Supported archive types are: zip, tar,
gzip-compressed tar, bzip2-compressed tar, and xz-compressed tar.

Usage

To list the contents of a zip file:

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

To extract the contents of a .tar.xz file:

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

To determine the type of archive file:

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
*/
package archive
