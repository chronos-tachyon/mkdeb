package mkdeb

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"encoding"
	"fmt"
	"hash"
	"io"
	"strings"

	getopt "github.com/pborman/getopt/v2"
)

type HashAlgorithm byte

const (
	HashMD5 HashAlgorithm = iota
	HashSHA1
	HashSHA256
)

var standardHashes = [...]HashAlgorithm{
	HashMD5,
	HashSHA1,
	HashSHA256,
}

var hashGoNameArray = [...]string{
	"mkdeb.HashMD5",
	"mkdeb.HashSHA1",
	"mkdeb.HashSHA256",
}

var hashNameArray = [...]string{
	"MD5",
	"SHA1",
	"SHA256",
}

var hashFileNameArray = [...]string{
	"md5sum",
	"sha1sum",
	"sha256sum",
}

var hashMap = map[string]HashAlgorithm{
	"md5":     HashMD5,
	"md-5":    HashMD5,
	"sha1":    HashSHA1,
	"sha-1":   HashSHA1,
	"sha256":  HashSHA256,
	"sha-256": HashSHA256,
	"sha2":    HashSHA256,
	"sha-2":   HashSHA256,
}

func (algo HashAlgorithm) GoString() string {
	if algo < HashAlgorithm(len(hashGoNameArray)) {
		return hashGoNameArray[algo]
	}
	return fmt.Sprintf("mkdeb.HashAlgorithm(0x%02x)", byte(algo))
}

func (algo HashAlgorithm) String() string {
	if algo < HashAlgorithm(len(hashNameArray)) {
		return hashNameArray[algo]
	}
	return fmt.Sprintf("algo#%02x", byte(algo))
}

func (algo HashAlgorithm) FileName() string {
	if algo < HashAlgorithm(len(hashFileNameArray)) {
		return hashFileNameArray[algo]
	}
	return fmt.Sprintf("cksum.%02x", byte(algo))
}

func (algo HashAlgorithm) MarshalText() ([]byte, error) {
	str := algo.String()
	return []byte(str), nil
}

func (algo HashAlgorithm) New() hash.Hash {
	switch algo {
	case HashMD5:
		return md5.New()
	case HashSHA1:
		return sha1.New()
	case HashSHA256:
		return sha256.New()
	default:
		panic(fmt.Errorf("%#v not implemented", algo))
	}
}

func (algo *HashAlgorithm) Parse(input string) error {
	if value, found := hashMap[input]; found {
		*algo = value
		return nil
	}
	if value, found := hashMap[strings.ToLower(input)]; found {
		*algo = value
		return nil
	}
	*algo = 0
	return fmt.Errorf("failed to parse %q as HashAlgorithm enum constant", input)
}

func (algo *HashAlgorithm) UnmarshalText(input []byte) error {
	return algo.Parse(string(input))
}

func (algo *HashAlgorithm) Set(value string, opt getopt.Option) error {
	return algo.Parse(value)
}

var (
	_ fmt.GoStringer           = HashAlgorithm(0)
	_ fmt.Stringer             = HashAlgorithm(0)
	_ encoding.TextMarshaler   = HashAlgorithm(0)
	_ encoding.TextUnmarshaler = (*HashAlgorithm)(nil)
)

type hashWriter struct {
	file    io.Writer
	hashers map[HashAlgorithm]hash.Hash
}

func (hw hashWriter) Write(p []byte) (int, error) {
	n, err := hw.file.Write(p)
	if n > 0 {
		for _, hasher := range hw.hashers {
			_, err2 := hasher.Write(p[:n])
			if err2 != nil {
				panic(err2)
			}
		}
	}
	return n, err
}
