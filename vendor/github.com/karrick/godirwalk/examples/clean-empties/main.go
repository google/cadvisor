package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/karrick/godirwalk"
	"github.com/pkg/errors"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "usage: %s dir1 [dir2 [dir3...]]\n", filepath.Base(os.Args[0]))
		os.Exit(2)
	}

	scratchBuffer := make([]byte, 64*1024) // allocate once and re-use each time
	var count, total int
	var err error

	for _, arg := range os.Args[1:] {
		count, err = pruneEmptyDirectories(arg, scratchBuffer)
		total += count
		if err != nil {
			break
		}
	}

	fmt.Fprintf(os.Stderr, "Removed %d empty directories\n", total)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %s\n", err)
		os.Exit(1)
	}
}

func pruneEmptyDirectories(osDirname string, scratchBuffer []byte) (int, error) {
	var count int

	err := godirwalk.Walk(osDirname, &godirwalk.Options{
		Unsorted:      true,
		ScratchBuffer: scratchBuffer,
		Callback: func(_ string, _ *godirwalk.Dirent) error {
			// no-op while diving in; all the fun happens in PostChildrenCallback
			return nil
		},
		PostChildrenCallback: func(osPathname string, _ *godirwalk.Dirent) error {
			deChildren, err := godirwalk.ReadDirents(osPathname, scratchBuffer)
			if err != nil {
				return errors.Wrap(err, "cannot ReadDirents")
			}
			// NOTE: ReadDirents skips "." and ".."
			if len(deChildren) > 0 {
				return nil // this directory has children; no additional work here
			}
			if osPathname == osDirname {
				return nil // do not remove provided root directory
			}
			err = os.Remove(osPathname)
			if err == nil {
				count++
			}
			return err
		},
	})

	return count, err
}
