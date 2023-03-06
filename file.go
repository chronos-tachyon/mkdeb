package mkdeb

import (
	"archive/tar"
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"path"
	"strings"
	"time"
)

type File struct {
	Name   string    `json:"name"`
	Type   Type      `json:"type"`
	IsConf bool      `json:"isConf"`
	Perm   Perm      `json:"perm"`
	User   Owner     `json:"user"`
	Group  Owner     `json:"group"`
	MTime  time.Time `json:"mtime"`
	Major  *int64    `json:"major"`
	Minor  *int64    `json:"minor"`
	Path   *string   `json:"path"`
	Text   *string   `json:"text"`
	Bytes  *[]byte   `json:"bytes"`
	Link   *string   `json:"link"`

	isResolved bool  `json:"-"`
	size       int64 `json:"-"`

	isHashed bool                     `json:"-"`
	hashes   map[HashAlgorithm][]byte `json:"-"`
}

func (file File) Validate() error {
	return file.validateImpl()
}

func (file *File) Resolve(fileSystem fs.FS) error {
	if err := file.validateImpl(); err != nil {
		return err
	}

	var size int64
	if file.Type == TypeREG {
		var statPath string
		var statNeeded bool

		switch {
		case file.Bytes != nil:
			size = int64(len(*file.Bytes))
		case file.Text != nil:
			size = int64(len(*file.Text))
		case file.Path != nil:
			statPath = *file.Path
			statNeeded = true
		default:
			statPath = file.Name
			statNeeded = true
		}

		if statNeeded {
			fi, err := fs.Stat(fileSystem, statPath)
			if err != nil {
				return fmt.Errorf("failed to stat %q: %w", statPath, err)
			}
			size = fi.Size()
		}
	}

	file.size = size
	file.isResolved = true
	return nil
}

func (file *File) validateImpl() error {
	if file.Name == "" {
		return fmt.Errorf("name: missing required field")
	}

	if !isValidUnixPath(file.Name) {
		return fmt.Errorf("name: invalid Unix path %q", file.Name)
	}

	if !file.Type.IsValid() {
		return fmt.Errorf("type: invalid value %#v", file.Type)
	}

	if file.Type == TypeAUTO {
		file.Type = TypeREG
		if strings.HasSuffix(file.Name, "/") {
			file.Type = TypeDIR
		}
	}

	if file.Type == TypeDIR {
		file.Name = strings.TrimRight(file.Name, "/") + "/"
	} else {
		if file.Name == "." || strings.HasSuffix(file.Name, "/") {
			return fmt.Errorf("name: value is only appropriate for a directory: %q", file.Name)
		}
	}

	if file.Type == TypeREG {
		if file.Path != nil {
			name := *file.Path
			if !isValidUnixPath(name) {
				return fmt.Errorf("path: invalid Unix path %q", name)
			}
			if strings.HasSuffix(name, "/") {
				return fmt.Errorf("path: unexpected trailing '/': %q", name)
			}
		}
		if file.Path != nil && file.Text != nil {
			return fmt.Errorf("text: conflict with field \"path\"")
		}
		if file.Path != nil && file.Bytes != nil {
			return fmt.Errorf("bytes: conflict with field \"path\"")
		}
		if file.Text != nil && file.Bytes != nil {
			return fmt.Errorf("bytes: conflict with field \"text\"")
		}
	} else {
		if file.IsConf {
			return fmt.Errorf("isConf: conflict with field \"type\"")
		}
		if file.Path != nil {
			return fmt.Errorf("path: unexpected value for field: %q", *file.Path)
		}
		if file.Text != nil {
			return fmt.Errorf("text: unexpected value for field (%d bytes)", len(*file.Text))
		}
		if file.Bytes != nil {
			return fmt.Errorf("bytes: unexpected value for field (%d bytes)", len(*file.Bytes))
		}
	}

	if file.Type == TypeLNK {
		if file.Link == nil {
			return fmt.Errorf("link: missing required field")
		}
		link := *file.Link
		clean := path.Clean(link)
		if link != clean {
			return fmt.Errorf("link: value is not canonical: expected %q, got %q", clean, link)
		}
	} else {
		if file.Link != nil {
			return fmt.Errorf("link: unexpected value for field: %q", *file.Link)
		}
	}

	if file.Type == TypeCHR || file.Type == TypeBLK {
		if file.Major == nil {
			return fmt.Errorf("major: missing required field")
		}
		if file.Minor == nil {
			return fmt.Errorf("major: missing required field")
		}
	} else {
		if file.Major != nil {
			return fmt.Errorf("major: unexpected value for field: %d", *file.Major)
		}
		if file.Minor != nil {
			return fmt.Errorf("minor: unexpected value for field: %d", *file.Minor)
		}
	}

	return nil
}

func (file File) AsTarHeader() tar.Header {
	if !file.isResolved {
		panic(fmt.Errorf("must call Resolve first"))
	}

	var hdr tar.Header
	hdr.Format = tar.FormatPAX
	hdr.Typeflag = 0
	hdr.Name = file.Name
	hdr.Mode = unixModeFMT
	hdr.ModTime = file.MTime
	hdr.Size = file.size

	switch file.User.Type {
	case OwnerByID:
		hdr.Uid = file.User.ID
	case OwnerByName:
		hdr.Uname = file.User.Name
	}

	switch file.Group.Type {
	case OwnerByID:
		hdr.Gid = file.Group.ID
	case OwnerByName:
		hdr.Gname = file.Group.Name
	}

	var defaultPerm Perm = 0o600
	switch file.Type {
	case TypeDIR:
		defaultPerm = 0o755
		hdr.Typeflag = tar.TypeDir
		hdr.Mode = unixModeDIR

	case TypeREG:
		defaultPerm = 0o644
		hdr.Typeflag = tar.TypeReg
		hdr.Mode = unixModeREG

	case TypeLNK:
		defaultPerm = 0o777
		hdr.Typeflag = tar.TypeSymlink
		hdr.Mode = unixModeLNK
		hdr.Linkname = *file.Link

	case TypeFIFO:
		hdr.Typeflag = tar.TypeFifo
		hdr.Mode = unixModeFIFO

	case TypeCHR:
		hdr.Typeflag = tar.TypeChar
		hdr.Mode = unixModeCHR
		hdr.Devmajor = *file.Major
		hdr.Devminor = *file.Minor

	case TypeBLK:
		hdr.Typeflag = tar.TypeBlock
		hdr.Mode = unixModeBLK
		hdr.Devmajor = *file.Major
		hdr.Devminor = *file.Minor
	}

	perm := file.Perm
	if perm == 0 {
		perm = defaultPerm
	}
	hdr.Mode |= int64(uint64(perm))

	return hdr
}

func (file File) Reader(fileSystem fs.FS) (io.ReadCloser, error) {
	if !file.isResolved {
		panic(fmt.Errorf("must call Resolve first"))
	}

	if file.Type != TypeREG {
		return emptyReadCloser{}, nil
	}

	var name string
	switch {
	case file.Bytes != nil:
		return io.NopCloser(bytes.NewReader(*file.Bytes)), nil

	case file.Text != nil:
		return io.NopCloser(strings.NewReader(*file.Text)), nil

	case file.Path != nil:
		name = *file.Path

	default:
		name = file.Name
	}

	f, err := fileSystem.Open(name)
	if err != nil {
		return nil, fmt.Errorf("failed to open %q: %w", name, err)
	}
	return f, nil
}
