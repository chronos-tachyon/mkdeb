package mkdeb

import (
	"encoding"
	"encoding/json"
	"fmt"
	"strconv"
)

type OwnerType byte

const (
	OwnerUnspecified OwnerType = iota
	OwnerByID
	OwnerByName
)

type Owner struct {
	Type OwnerType
	ID   int
	Name string
}

func ID(id int) Owner {
	return Owner{Type: OwnerByID, ID: id}
}

func Name(name string) Owner {
	return Owner{Type: OwnerByName, Name: name}
}

func (owner Owner) IsZero() bool {
	return owner.Type != OwnerByID && owner.Type != OwnerByName
}

func (owner Owner) Equal(other Owner) bool {
	switch owner.Type {
	case OwnerByID:
		return other.Type == OwnerByID && owner.ID == other.ID
	case OwnerByName:
		return other.Type == OwnerByName && owner.Name == other.Name
	default:
		return other.IsZero()
	}
}

func (owner Owner) GoString() string {
	switch owner.Type {
	case OwnerByID:
		return fmt.Sprintf("mkdeb.ID(%d)", owner.ID)
	case OwnerByName:
		return fmt.Sprintf("mkdeb.Name(%q)", owner.Name)
	default:
		return "mkdeb.Owner{}"
	}
}

func (owner Owner) String() string {
	switch owner.Type {
	case OwnerByID:
		return fmt.Sprintf("#%d", owner.ID)
	case OwnerByName:
		return owner.Name
	default:
		return ""
	}
}

func (owner Owner) MarshalText() ([]byte, error) {
	str := owner.String()
	return []byte(str), nil
}

func (owner Owner) MarshalJSON() ([]byte, error) {
	switch owner.Type {
	case OwnerByID:
		return json.Marshal(owner.ID)
	case OwnerByName:
		return json.Marshal(owner.Name)
	default:
		return []byte("null"), nil
	}
}

func (owner *Owner) Parse(input string) error {
	*owner = Owner{}
	if input == "" {
		return nil
	}
	if input[0] == '#' {
		if i64, err := strconv.ParseInt(input[1:], 10, 0); err == nil {
			*owner = ID(int(i64))
			return nil
		}
	}
	*owner = Name(input)
	return nil
}

func (owner *Owner) UnmarshalText(input []byte) error {
	return owner.Parse(string(input))
}

func (owner *Owner) UnmarshalJSON(input []byte) error {
	*owner = Owner{}
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
		return owner.Parse(str)
	}
	var id int
	if err := json.Unmarshal(input, &id); err != nil {
		return fmt.Errorf("failed to parse JSON value %q as int: %w", input, err)
	}
	*owner = ID(id)
	return nil
}

var (
	_ fmt.GoStringer           = Owner{}
	_ fmt.Stringer             = Owner{}
	_ encoding.TextMarshaler   = Owner{}
	_ json.Marshaler           = Owner{}
	_ encoding.TextUnmarshaler = (*Owner)(nil)
	_ json.Unmarshaler         = (*Owner)(nil)
)
