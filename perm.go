package mkdeb

import (
	"encoding"
	"encoding/json"
	"fmt"
	"strconv"
)

type Perm uint16

func (perm Perm) GoString() string {
	return fmt.Sprintf("mkdeb.Perm(0o%04o)", uint16(perm))
}

func (perm Perm) AppendTo(out []byte) []byte {
	return appendOctal(out, uint16(perm))
}

func (perm Perm) String() string {
	var buffer [16]byte
	return string(perm.AppendTo(buffer[:0]))
}

func (perm Perm) MarshalText() ([]byte, error) {
	return perm.AppendTo(make([]byte, 0, 16)), nil
}

func (perm *Perm) Parse(input string) error {
	*perm = 0

	if input == "" {
		return nil
	}

	num, err := strconv.ParseUint(input, 8, 16)
	if err != nil {
		return fmt.Errorf("failed to parse %q as octal uint16: %w", input, err)
	}
	*perm = Perm(num & 0o7777)
	return nil
}

func (perm *Perm) UnmarshalText(input []byte) error {
	return perm.Parse(string(input))
}

func (perm *Perm) UnmarshalJSON(input []byte) error {
	*perm = 0
	if jsonIsNull(input) {
		return nil
	}
	if len(input) == 2 && string(input) == `""` {
		return nil
	}
	if input[0] == '"' {
		var str string
		if err := json.Unmarshal(input, &str); err != nil {
			return fmt.Errorf("failed to parse JSON value %q as string: %w", input, err)
		}
		return perm.Parse(str)
	}
	var num uint16
	if err := json.Unmarshal(input, &num); err != nil {
		return fmt.Errorf("failed to parse JSON value %q as uint16: %w", input, err)
	}
	*perm = Perm(num & 0o7777)
	return nil
}

var (
	_ fmt.GoStringer           = Perm(0)
	_ fmt.Stringer             = Perm(0)
	_ encoding.TextMarshaler   = Perm(0)
	_ encoding.TextUnmarshaler = (*Perm)(nil)
	_ json.Unmarshaler         = (*Perm)(nil)
)

func appendOctal(out []byte, num uint16) []byte {
	const octal = "01234567"
	ch0 := octal[(num>>9)&0o7]
	ch1 := octal[(num>>6)&0o7]
	ch2 := octal[(num>>3)&0o7]
	ch3 := octal[(num>>0)&0o7]
	return append(out, ch0, ch1, ch2, ch3)
}
