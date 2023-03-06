package mkdeb

import (
	"compress/gzip"
	"encoding"
	"fmt"
	"io"
	"strings"

	"github.com/klauspost/compress/zstd"
	getopt "github.com/pborman/getopt/v2"
	"github.com/ulikunitz/xz"
)

type CompressAlgorithm byte

const (
	CompressAuto CompressAlgorithm = iota
	CompressNone
	CompressGZIP
	CompressBZIP2
	CompressXZ
	CompressZSTD
)

var compressGoNameArray = [...]string{
	"mkdeb.CompressAuto",
	"mkdeb.CompressNone",
	"mkdeb.CompressGZIP",
	"mkdeb.CompressBZIP2",
	"mkdeb.CompressXZ",
	"mkdeb.CompressZSTD",
}

var compressNameArray = [...]string{
	"auto",
	"none",
	"gzip",
	"bzip2",
	"xz",
	"zstd",
}

var compressSuffixArray = [...]string{
	"",
	"",
	".gz",
	".bz2",
	".xz",
	".zstd",
}

var compressMap = map[string]CompressAlgorithm{
	"":      CompressAuto,
	"auto":  CompressAuto,
	"none":  CompressNone,
	"gzip":  CompressGZIP,
	"gz":    CompressGZIP,
	"bzip2": CompressBZIP2,
	"bzip":  CompressBZIP2,
	"bz2":   CompressBZIP2,
	"xz":    CompressXZ,
	"zstd":  CompressZSTD,
}

func (algo CompressAlgorithm) GoString() string {
	if algo < CompressAlgorithm(len(compressGoNameArray)) {
		return compressGoNameArray[algo]
	}
	return fmt.Sprintf("mkdeb.CompressAlgorithm(0x%02x)", byte(algo))
}

func (algo CompressAlgorithm) String() string {
	if algo < CompressAlgorithm(len(compressNameArray)) {
		return compressNameArray[algo]
	}
	return fmt.Sprintf("compression#%02x", byte(algo))
}

func (algo CompressAlgorithm) Suffix() string {
	if algo < CompressAlgorithm(len(compressSuffixArray)) {
		return compressSuffixArray[algo]
	}
	return ""
}

func (algo CompressAlgorithm) MarshalText() ([]byte, error) {
	str := algo.String()
	return []byte(str), nil
}

func (algo CompressAlgorithm) NewWriter(w io.Writer) (io.WriteCloser, error) {
	switch algo {
	case CompressNone:
		return &nopCloseWriter{w}, nil

	case CompressGZIP:
		cw, err := gzip.NewWriterLevel(w, gzip.BestCompression)
		if err != nil {
			return nil, fmt.Errorf("gzip.NewWriterLevel: %d: %w", gzip.BestCompression, err)
		}
		return cw, nil

	case CompressXZ:
		cw, err := xz.NewWriter(w)
		if err != nil {
			return nil, fmt.Errorf("xz.NewWriter: %w", err)
		}
		return cw, nil

	case CompressZSTD:
		cw, err := zstd.NewWriter(w, zstd.WithEncoderLevel(zstd.SpeedBestCompression))
		if err != nil {
			return nil, fmt.Errorf("zstd.NewWriter: %w", err)
		}
		return cw, nil

	default:
		panic(fmt.Errorf("%#v not implemented", algo))
	}
}

func (algo *CompressAlgorithm) Parse(input string) error {
	if value, found := compressMap[input]; found {
		*algo = value
		return nil
	}
	if value, found := compressMap[strings.ToLower(input)]; found {
		*algo = value
		return nil
	}
	*algo = 0
	return fmt.Errorf("failed to parse %q as mkdeb.CompressAlgorithm enum constant", input)
}

func (algo *CompressAlgorithm) UnmarshalText(input []byte) error {
	return algo.Parse(string(input))
}

func (algo *CompressAlgorithm) Set(value string, opt getopt.Option) error {
	return algo.Parse(value)
}

var (
	_ fmt.GoStringer           = CompressAlgorithm(0)
	_ fmt.Stringer             = CompressAlgorithm(0)
	_ encoding.TextMarshaler   = CompressAlgorithm(0)
	_ encoding.TextUnmarshaler = (*CompressAlgorithm)(nil)
	_ getopt.Value             = (*CompressAlgorithm)(nil)
)
