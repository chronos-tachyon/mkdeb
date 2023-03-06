package mkdeb

import (
	"encoding"
	"fmt"
	"strings"
)

type Type byte

const (
	TypeAUTO Type = iota
	TypeDIR
	TypeREG
	TypeLNK
	TypeFIFO
	TypeCHR
	TypeBLK
)

var fileTypeGoNameArray = [...]string{
	"mkdeb.TypeAUTO",
	"mkdeb.TypeDIR",
	"mkdeb.TypeREG",
	"mkdeb.TypeLNK",
	"mkdeb.TypeFIFO",
	"mkdeb.TypeCHR",
	"mkdeb.TypeBLK",
}

var fileTypeNameArray = [...]string{
	"AUTO",
	"DIR",
	"REG",
	"LNK",
	"FIFO",
	"CHR",
	"BLK",
}

var fileTypeAliasMap = map[string]Type{
	"":              TypeAUTO,
	"auto":          TypeAUTO,
	"directory":     TypeDIR,
	"dir":           TypeDIR,
	"d":             TypeDIR,
	"regular-file":  TypeREG,
	"regular":       TypeREG,
	"reg":           TypeREG,
	"r":             TypeREG,
	"file":          TypeREG,
	"f":             TypeREG,
	"-":             TypeREG,
	"link":          TypeLNK,
	"l":             TypeLNK,
	"symbolic-link": TypeLNK,
	"sym-link":      TypeLNK,
	"symlink":       TypeLNK,
	"fifo":          TypeFIFO,
	"pipe":          TypeFIFO,
	"p":             TypeFIFO,
	"char-device":   TypeCHR,
	"char-dev":      TypeCHR,
	"chardev":       TypeCHR,
	"char":          TypeCHR,
	"chr-dev":       TypeCHR,
	"chrdev":        TypeCHR,
	"chr":           TypeCHR,
	"c":             TypeCHR,
	"block-device":  TypeBLK,
	"block-dev":     TypeBLK,
	"blockdev":      TypeBLK,
	"block":         TypeBLK,
	"blk-dev":       TypeBLK,
	"blkdev":        TypeBLK,
	"blk":           TypeBLK,
	"b":             TypeBLK,
}

func (ft Type) IsValid() bool {
	return ft < Type(len(fileTypeGoNameArray))
}

func (ft Type) GoString() string {
	if ft < Type(len(fileTypeGoNameArray)) {
		return fileTypeGoNameArray[ft]
	}
	return fmt.Sprintf("mkdeb.Type(0x%02x)", byte(ft))
}

func (ft Type) String() string {
	if ft < Type(len(fileTypeNameArray)) {
		return fileTypeNameArray[ft]
	}
	return fmt.Sprintf("fileType#%02x", byte(ft))
}

func (ft Type) MarshalText() ([]byte, error) {
	str := ft.String()
	return []byte(str), nil
}

func (ft *Type) Parse(input string) error {
	if value, found := fileTypeAliasMap[input]; found {
		*ft = value
		return nil
	}
	if value, found := fileTypeAliasMap[strings.ToLower(input)]; found {
		*ft = value
		return nil
	}
	*ft = 0
	return fmt.Errorf("failed to parse %q as mkdeb.Type enum constant", input)
}

func (ft *Type) UnmarshalText(input []byte) error {
	return ft.Parse(string(input))
}

var (
	_ fmt.GoStringer           = Type(0)
	_ fmt.Stringer             = Type(0)
	_ encoding.TextMarshaler   = Type(0)
	_ encoding.TextUnmarshaler = (*Type)(nil)
)
