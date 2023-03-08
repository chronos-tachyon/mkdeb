package main

import (
	"os"

	"github.com/chronos-tachyon/mkdeb"
)

func main() {
	os.Exit(mkdeb.Main(os.Stdout, os.Stderr, os.Args))
}
