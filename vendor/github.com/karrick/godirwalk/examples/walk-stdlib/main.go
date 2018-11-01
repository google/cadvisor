package main

import (
	"fmt"
	"os"
	"path/filepath"
)

func main() {
	dirname := "."
	if len(os.Args) > 1 {
		dirname = os.Args[1]
	}
	err := filepath.Walk(dirname, func(osPathname string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		// fmt.Printf("%s %s\n", info.Mode(), osPathname)
		return nil
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}
