# tools
A collection of tools &amp; utilities written in Go (golang).

## archive-uninstall

This tool permits the removal of previously-extracted contents of an archive file
from a target directory where it may tedious to do so manually.

### Install

```bash
go get github.com/calennert/tools/archive-uninstall
```

### Usage

```
$ archive-uninstall --help
usage: archive-uninstall [<flags>] <archive filename> <target directory>

A tool to remove archive contents from a target directory.

Flags:
      --help       Show context-sensitive help (also try --help-long and --help-man).
  -v, --verbose    Enable verbose mode.
      --dry-run    Enable dry run mode. No files will be removed from target directory.
      --verify     Only remove verified files.
      --no-color   Disable color output in verbose mode.
  -t, --type=TYPE  The archive type. Determined from archive filename, if not specified.
      --version    Show application version.

Args:
  <archive filename>  The filename of the archive that will be compared to the target directory.
  <target directory>  The target directory from which to remove files.

Project home: https://github.com/calennert/tools/archive-uninstall
```
