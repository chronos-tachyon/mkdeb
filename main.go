package mkdeb

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	getopt "github.com/pborman/getopt/v2"
)

var Version = "devel"

func Main(stdout io.Writer, stderr io.Writer, argv []string) int {
	var (
		isHelp       bool
		isVersion    bool
		rootPath     string
		manifestPath string
		filePath     string
		compress     CompressAlgorithm
	)

	flagSet := getopt.New()
	flagSet.SetParameters("")
	flagSet.FlagLong(&isHelp, "help", 'h', "show usage")
	flagSet.FlagLong(&isVersion, "version", 'V', "show version")
	flagSet.FlagLong(&rootPath, "root", 'R', "path to root directory for input files")
	flagSet.FlagLong(&manifestPath, "manifest", 'm', "path to input manifest file (JSON)")
	flagSet.FlagLong(&filePath, "output", 'o', "path to output .deb package file")
	flagSet.FlagLong(&compress, "compression", 'c', "compression algorithm: {none|gzip|bzip2|xz|zstd}")
	err := flagSet.Getopt(argv, nil)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		flagSet.PrintUsage(stderr)
		return 1
	}

	if isHelp {
		flagSet.PrintUsage(stdout)
		return 0
	}

	if isVersion {
		fmt.Fprintf(stdout, "%s\n", Version)
		return 0
	}

	if manifestPath == "" {
		fmt.Fprintf(stderr, "error: missing required flag: -m / --manifest\n")
		return 1
	}

	if filePath == "" {
		fmt.Fprintf(stderr, "error: missing required flag: -o / --output\n")
		return 1
	}

	if rootPath == "" {
		rootPath = "."
	}

	rootPathAbs, err := filepath.Abs(rootPath)
	if err != nil {
		fmt.Fprintf(stderr, "error: failed to make path absolute: %q: %v\n", rootPath, err)
		return 1
	}

	if !filepath.IsAbs(manifestPath) {
		manifestPath = filepath.Join(rootPathAbs, manifestPath)
	}

	if !filepath.IsAbs(filePath) {
		filePath = filepath.Join(rootPathAbs, filePath)
	}

	manifestData, err := os.ReadFile(manifestPath)
	if err != nil {
		fmt.Fprintf(stderr, "error: failed to read manifest file: %q: %v\n", manifestPath, err)
		return 1
	}

	var manifest Manifest
	d := json.NewDecoder(bytes.NewReader(manifestData))
	d.DisallowUnknownFields()
	err = d.Decode(&manifest)
	if err != nil {
		fmt.Fprintf(stderr, "error: failed to parse manifest file as JSON: %q: %v\n", manifestPath, err)
		return 1
	}

	var builder Builder
	builder.Root = os.DirFS(rootPathAbs)
	builder.Compression = compress

	file, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o666)
	if err != nil {
		fmt.Fprintf(stderr, "error: failed to create output file: %q: %v\n", filePath, err)
		return 1
	}

	needCloseFile := true
	defer func() {
		if needCloseFile {
			_ = file.Close()
		}
	}()

	err = builder.Build(file, &manifest)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}

	err = file.Sync()
	if err != nil {
		fmt.Fprintf(stderr, "error: failed to sync output file to disk: %q: %v\n", filePath, err)
		return 1
	}

	needCloseFile = false
	err = file.Close()
	if err != nil {
		fmt.Fprintf(stderr, "error: failed to close output file: %q: %v\n", filePath, err)
		return 1
	}

	dirPath := filepath.Dir(filePath)
	dir, err := os.OpenFile(dirPath, os.O_RDONLY, 0)
	if err != nil {
		fmt.Fprintf(stderr, "error: failed to open directory: %q: %v\n", dirPath, err)
		return 1
	}

	needCloseDir := true
	defer func() {
		if needCloseDir {
			_ = dir.Close()
		}
	}()

	err = dir.Sync()
	if err != nil {
		fmt.Fprintf(stderr, "error: failed to sync directory to disk: %q: %v\n", dirPath, err)
		return 1
	}

	needCloseDir = false
	err = dir.Close()
	if err != nil {
		fmt.Fprintf(stderr, "error: failed to close directory: %q: %v\n", dirPath, err)
		return 1
	}

	return 0
}
