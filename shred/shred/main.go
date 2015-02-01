package main

import (
	"fmt"
	"github.com/andres-erbsen/chatterbox/shred"
	"os"
)

func main() {
	if len(os.Args) == 1 {
		fmt.Fprintf(os.Stderr, `%s: missing operand\n
Usage: %s [-rf] FILE...
		`, os.Args[0], os.Args[0])
		os.Exit(1)
	}
	exitCode := 0
	if os.Args[1] == "-rf" {
		for _, path := range os.Args[2:] {
			if err := shred.RemoveAll(path); err != nil {
				fmt.Fprintf(os.Stderr, "%s: %s\n", path, err)
				exitCode = 1
			}
		}
	} else {
		for i, path := range os.Args[1:] {
			if i == 1 && path == "--" {
				continue
			}
			if err := shred.Remove(path); err != nil {
				fmt.Fprintf(os.Stderr, "%s: %s\n", path, err)
				exitCode = 1
			}
		}
	}
	os.Exit(exitCode)
}
