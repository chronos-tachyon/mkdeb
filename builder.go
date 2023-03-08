package mkdeb

import (
	"archive/tar"
	"bytes"
	"encoding/hex"
	"fmt"
	"hash"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"time"
)

type Builder struct {
	Root        fs.FS
	ZeroTime    time.Time
	Compression CompressAlgorithm
	Hashes      []HashAlgorithm
}

func (builder *Builder) fillDefaults() {
	if builder.ZeroTime.IsZero() {
		builder.ZeroTime = time.Unix(1577836800, 0)
	}

	if builder.Compression == CompressAuto {
		builder.Compression = CompressGZIP
	}

	if builder.Hashes == nil {
		builder.Hashes = standardHashes[:]
	}
}

func (builder Builder) Build(w io.Writer, manifest *Manifest) error {
	builder.fillDefaults()

	err := manifest.Resolve(builder.Root)
	if err != nil {
		return err
	}

	tempDir, err := os.MkdirTemp("", "mkdeb-*.d")
	if err != nil {
		return fmt.Errorf("failed to create temporary directory: %w", err)
	}

	needRemoveTempDir := true
	defer func() {
		if needRemoveTempDir {
			_ = os.RemoveAll(tempDir)
		}
	}()

	suffix := builder.Compression.Suffix()

	controlPath := filepath.Join(tempDir, "control.tar"+suffix)
	controlFile, err := os.OpenFile(controlPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return fmt.Errorf("failed to create temporary file: %q: %w", controlPath, err)
	}

	needCloseControlFile := true
	defer func() {
		if needCloseControlFile {
			_ = controlFile.Close()
		}
	}()

	dataPath := filepath.Join(tempDir, "data.tar"+suffix)
	dataFile, err := os.OpenFile(dataPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return fmt.Errorf("failed to create temporary file: %q: %w", dataPath, err)
	}

	needCloseDataFile := true
	defer func() {
		if needCloseDataFile {
			_ = dataFile.Close()
		}
	}()

	err = builder.BuildDataTarball(dataFile, manifest)
	if err != nil {
		return fmt.Errorf("failed to build data tarball in temporary file: %w", err)
	}

	err = builder.BuildControlTarball(controlFile, manifest)
	if err != nil {
		return fmt.Errorf("failed to build control tarball in temporary file: %w", err)
	}

	err = builder.writeArFile(w, controlFile, dataFile)
	if err != nil {
		return err
	}

	needCloseDataFile = false
	_ = dataFile.Close()

	needCloseControlFile = false
	_ = controlFile.Close()

	needRemoveTempDir = false
	err = os.RemoveAll(tempDir)
	if err != nil {
		return fmt.Errorf("failed to clean up temporary directory: %q: %w", tempDir, err)
	}

	return nil
}

func (builder Builder) BuildDataTarball(w io.Writer, manifest *Manifest) error {
	if !manifest.isResolved {
		panic(fmt.Errorf("must call manifest.Resolve first"))
	}

	builder.fillDefaults()

	cw, err := builder.Compression.NewWriter(w)
	if err != nil {
		return err
	}

	tw := tar.NewWriter(cw)

	var rc io.ReadCloser
	needClose := false
	defer func() {
		if needClose {
			_ = rc.Close()
		}
	}()

	for index := range manifest.Files {
		file := &manifest.Files[index]
		file.isHashed = false
		file.hashes = nil

		hdr := file.AsTarHeader()
		if hdr.ModTime.IsZero() {
			hdr.ModTime = builder.ZeroTime
		}

		err = tw.WriteHeader(&hdr)
		if err != nil {
			return fmt.Errorf("files[%d]: tar.WriteHeader: %w", index, err)
		}

		if file.Type == TypeREG {
			rc, err = file.Reader(builder.Root)
			if err != nil {
				return fmt.Errorf("files[%d]: Open: %w", index, err)
			}
			needClose = true

			hw := &hashWriter{
				file:    tw,
				hashers: make(map[HashAlgorithm]hash.Hash, len(builder.Hashes)),
			}
			for _, algo := range builder.Hashes {
				hw.hashers[algo] = algo.New()
			}

			_, err = io.Copy(hw, rc)
			if err != nil {
				return fmt.Errorf("files[%d]: Copy: %w", index, err)
			}

			needClose = false
			err = rc.Close()
			if err != nil {
				return fmt.Errorf("files[%d]: Close: %w", index, err)
			}

			file.hashes = make(map[HashAlgorithm][]byte, len(builder.Hashes))
			for _, algo := range builder.Hashes {
				file.hashes[algo] = hw.hashers[algo].Sum(nil)
			}

			file.isHashed = true
		}
	}

	err = tw.Close()
	if err != nil {
		return err
	}

	err = cw.Close()
	if err != nil {
		return err
	}

	manifest.isHashed = true
	return nil
}

func (builder Builder) BuildControlTarball(w io.Writer, manifest *Manifest) error {
	if !manifest.isResolved {
		panic(fmt.Errorf("must call manifest.Resolve first"))
	}
	if !manifest.isHashed {
		panic(fmt.Errorf("must call BuildDataTarball first"))
	}

	builder.fillDefaults()

	cw, err := builder.Compression.NewWriter(w)
	if err != nil {
		return err
	}

	tw := tar.NewWriter(cw)

	err = builder.writeControlFile(tw, "control", false, manifest.ControlFile())
	if err != nil {
		return err
	}

	var buf bytes.Buffer
	buf.Grow(4096)
	for _, algo := range builder.Hashes {
		for _, file := range manifest.Files {
			if file.isHashed {
				buf.WriteString(hex.EncodeToString(file.hashes[algo]))
				buf.WriteString("  ")
				buf.WriteString(file.Name)
				buf.WriteString("\n")
			}
		}
		err = builder.writeControlFile(tw, algo.FileName(), false, buf.Bytes())
		if err != nil {
			return err
		}
		buf.Reset()
	}

	err = builder.writeControlFile(tw, "conffiles", false, manifest.ConfFiles())
	if err != nil {
		return err
	}

	err = builder.writeControlFile(tw, "preinst", true, manifest.PreInstallScript())
	if err != nil {
		return err
	}

	err = builder.writeControlFile(tw, "postinst", true, manifest.PostInstallScript())
	if err != nil {
		return err
	}

	err = builder.writeControlFile(tw, "prerm", true, manifest.PreRemoveScript())
	if err != nil {
		return err
	}

	err = builder.writeControlFile(tw, "postrm", true, manifest.PostRemoveScript())
	if err != nil {
		return err
	}

	err = tw.Close()
	if err != nil {
		return err
	}

	err = cw.Close()
	if err != nil {
		return err
	}

	return nil
}

func (builder Builder) writeControlFile(w *tar.Writer, name string, isExec bool, data []byte) error {
	if data == nil {
		return nil
	}

	var mode int64 = unixModeREG | 0o644
	if isExec {
		mode |= 0o111
	}

	hdr := tar.Header{
		Format:   tar.FormatPAX,
		Typeflag: tar.TypeReg,
		Name:     name,
		Mode:     mode,
		Size:     int64(len(data)),
		ModTime:  builder.ZeroTime,
	}

	err := w.WriteHeader(&hdr)
	if err != nil {
		return fmt.Errorf("tar.WriteHeader: %w", err)
	}

	_, err = w.Write(data)
	if err != nil {
		return fmt.Errorf("Write: %w", err)
	}

	return nil
}

func (builder Builder) writeArFile(w io.Writer, controlFile *os.File, dataFile *os.File) error {
	controlSize, err := controlFile.Seek(0, io.SeekEnd)
	if err != nil {
		return fmt.Errorf("Seek: end: %w", err)
	}

	_, err = controlFile.Seek(0, io.SeekStart)
	if err != nil {
		return fmt.Errorf("Seek: start: %w", err)
	}

	dataSize, err := dataFile.Seek(0, io.SeekEnd)
	if err != nil {
		return fmt.Errorf("Seek: end: %w", err)
	}

	_, err = dataFile.Seek(0, io.SeekStart)
	if err != nil {
		return fmt.Errorf("Seek: start: %w", err)
	}

	arMagic := []byte("!<arch>\n")
	_, err = w.Write(arMagic)
	if err != nil {
		return fmt.Errorf("Write: %w", err)
	}

	debianBinary := []byte("2.0\n")
	err = writeArEntry(w, "debian-binary", int64(len(debianBinary)), bytes.NewReader(debianBinary))
	if err != nil {
		return err
	}

	suffix := builder.Compression.Suffix()

	err = writeArEntry(w, "control.tar"+suffix, controlSize, controlFile)
	if err != nil {
		return err
	}

	err = writeArEntry(w, "data.tar"+suffix, dataSize, dataFile)
	if err != nil {
		return err
	}

	return nil
}

func writeArEntry(w io.Writer, name string, size int64, r io.Reader) error {
	if len(name) > 16 {
		panic(fmt.Errorf("name %q exceeds 16 bytes", name))
	}
	if size < 0 {
		panic(fmt.Errorf("size %d is negative", size))
	}
	sizeString := fmt.Sprintf("%-10d", size)
	if len(sizeString) != 10 {
		panic(fmt.Errorf("internal error: formatted size %q should be exactly %d bytes, but got %d bytes", sizeString, 10, len(sizeString)))
	}

	hdr := [60]byte{
		'?', '?', '?', '?', '?', '?', '?', '?',
		'?', '?', '?', '?', '?', '?', '?', '?',
		'1', '5', '7', '7', '8', '3', '6', '8',
		'0', '0', ' ', ' ', '0', ' ', ' ', ' ',
		' ', ' ', '0', ' ', ' ', ' ', ' ', ' ',
		'1', '0', '0', '6', '4', '4', ' ', ' ',
		'?', '?', '?', '?', '?', '?', '?', '?',
		'?', '?', '`', '\n',
	}

	for i := 0; i < 16; i++ {
		ch := byte(' ')
		if i < len(name) {
			ch = name[i]
		}
		hdr[i] = ch
	}

	for i := 0; i < 10; i++ {
		hdr[i+48] = sizeString[i]
	}

	_, err := w.Write(hdr[:])
	if err != nil {
		return fmt.Errorf("Write: %w", err)
	}

	_, err = io.Copy(w, r)
	if err != nil {
		return fmt.Errorf("Copy: %w", err)
	}

	if (size & 1) != 0 {
		pad := []byte{'\n'}
		_, err = w.Write(pad)
		if err != nil {
			return fmt.Errorf("Write: %w", err)
		}
	}

	return nil
}
