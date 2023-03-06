package main

import (
	"os"

	"github.com/chronos-tachyon/mkdeb"
)

var Version = "devel"

func main() {
	mkdeb.Version = Version
	os.Exit(mkdeb.Main(os.Stdout, os.Stderr, os.Args))
}
