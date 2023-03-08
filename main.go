package mkdeb

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	getopt "github.com/pborman/getopt/v2"
)

var versionDataKeys []string
var versionDataMap map[string]string

func SetVersion(pairs ...string) {
	pairsLen := uint(len(pairs))
	keysLen := pairsLen >> 1
	if (pairsLen & 1) == 1 {
		panic(fmt.Errorf("expected a whole number of key / value pairs, but got %d and a half (%d strings total)", keysLen, pairsLen))
	}

	keys := make([]string, keysLen)
	values := make(map[string]string, keysLen)
	seen := make(map[string]string, keysLen)

	for i := uint(0); i < keysLen; i++ {
		key := pairs[(i<<1)+0]
		value := pairs[(i<<1)+1]
		lc := strings.ToLower(key)

		if oldKey, found := seen[lc]; found {
			oldValue := values[oldKey]
			var err error
			if oldKey == key {
				err = fmt.Errorf("duplicate key %q: conflict between %q and %q", key, oldValue, value)
			} else {
				err = fmt.Errorf("duplicate key %q / %q: conflict between %q and %q", oldKey, key, oldValue, value)
			}
			panic(err)
		}

		keys[i] = key
		values[key] = value
		seen[lc] = key
	}

	versionDataKeys = keys
	versionDataMap = values
}

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
		for _, key := range versionDataKeys {
			value := versionDataMap[key]
			fmt.Fprintf(stdout, "%s=%s\n", key, value)
		}
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
