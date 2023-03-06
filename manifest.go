package mkdeb

import (
	"bytes"
	"fmt"
	"io/fs"
	"path"
	"strconv"
	"strings"
)

type Manifest struct {
	Package          string   `json:"package"`
	Version          string   `json:"version"`
	Arch             string   `json:"arch"`
	Section          string   `json:"section"`
	Priority         string   `json:"priority"`
	Essential        string   `json:"essential"`
	Depends          string   `json:"depends"`
	PreDepends       string   `json:"preDepends"`
	Recommends       string   `json:"recommends"`
	Suggests         string   `json:"suggests"`
	Enhances         string   `json:"enhances"`
	Breaks           string   `json:"breaks"`
	Conflicts        string   `json:"conflicts"`
	Maintainer       string   `json:"maintainer"`
	HomePage         string   `json:"homePage"`
	BuiltUsing       string   `json:"builtUsing"`
	ShortDescription string   `json:"shortDescription"`
	LongDescription  []string `json:"longDescription"`
	ImplicitDirs     []string `json:"implicitDirs"`
	Files            []File   `json:"files"`
	PreInstall       []string `json:"preInstall"`
	PostInstall      []string `json:"postInstall"`
	PreRemove        []string `json:"preRemove"`
	PostRemove       []string `json:"postRemove"`

	isResolved    bool  `json:"-"`
	installedSize int64 `json:"-"`

	isHashed bool `json:"-"`
}

func (manifest Manifest) Validate() error {
	if err := manifest.validatePre(); err != nil {
		return err
	}

	for index, file := range manifest.Files {
		if err := file.Validate(); err != nil {
			return fmt.Errorf("files[%d]: %w", index, err)
		}
	}

	return manifest.validatePost()
}

func (manifest *Manifest) Resolve(fileSystem fs.FS) error {
	if err := manifest.validatePre(); err != nil {
		return err
	}

	var installedSize int64
	for index := range manifest.Files {
		file := &manifest.Files[index]
		if err := file.Resolve(fileSystem); err != nil {
			return fmt.Errorf("files[%d]: %w", index, err)
		}
		installedSize += padSigned(file.size, 12)
	}

	if err := manifest.validatePost(); err != nil {
		return err
	}

	manifest.installedSize = installedSize
	manifest.isResolved = true
	return nil
}

func (manifest *Manifest) validatePre() error {
	if manifest.Package == "" {
		return fmt.Errorf("package: missing required field")
	}
	if !isValidPackage(manifest.Package) {
		return fmt.Errorf("package: invalid Debian package name %q", manifest.Package)
	}

	if manifest.Version == "" {
		return fmt.Errorf("version: missing required field")
	}
	if !isValidVersion(manifest.Version) {
		return fmt.Errorf("version: invalid Debian package version %q", manifest.Version)
	}

	if manifest.Arch == "" {
		return fmt.Errorf("arch: missing required field")
	}
	if !isValidArch(manifest.Arch) {
		return fmt.Errorf("version: invalid Debian package architecture %q", manifest.Arch)
	}

	if manifest.Section != "" && !isValidSection(manifest.Section) {
		return fmt.Errorf("version: invalid Debian package section %q", manifest.Section)
	}

	if manifest.Priority != "" && !isValidPriority(manifest.Priority) {
		return fmt.Errorf("version: invalid Debian package priority %q", manifest.Priority)
	}

	if manifest.Essential != "" && !isValidDepends(manifest.Essential) {
		return fmt.Errorf("essential: invalid Debian package dependency spec %q", manifest.Essential)
	}
	if manifest.Depends != "" && !isValidDepends(manifest.Depends) {
		return fmt.Errorf("depends: invalid Debian package dependency spec %q", manifest.Depends)
	}
	if manifest.PreDepends != "" && !isValidDepends(manifest.PreDepends) {
		return fmt.Errorf("preDepends: invalid Debian package dependency spec %q", manifest.PreDepends)
	}
	if manifest.Recommends != "" && !isValidDepends(manifest.Recommends) {
		return fmt.Errorf("recommends: invalid Debian package dependency spec %q", manifest.Recommends)
	}
	if manifest.Suggests != "" && !isValidDepends(manifest.Suggests) {
		return fmt.Errorf("suggests: invalid Debian package dependency spec %q", manifest.Suggests)
	}
	if manifest.Enhances != "" && !isValidDepends(manifest.Enhances) {
		return fmt.Errorf("enhances: invalid Debian package dependency spec %q", manifest.Enhances)
	}
	if manifest.Breaks != "" && !isValidDepends(manifest.Breaks) {
		return fmt.Errorf("breaks: invalid Debian package dependency spec %q", manifest.Breaks)
	}
	if manifest.Conflicts != "" && !isValidDepends(manifest.Conflicts) {
		return fmt.Errorf("conflicts: invalid Debian package dependency spec %q", manifest.Conflicts)
	}

	if manifest.Maintainer == "" {
		return fmt.Errorf("maintainer: missing required field")
	}
	if !isValidMaintainer(manifest.Maintainer) {
		return fmt.Errorf("maintainer: invalid Maintainer line %q", manifest.Maintainer)
	}

	if manifest.HomePage != "" && !isValidURL(manifest.HomePage) {
		return fmt.Errorf("homePage: invalid URL %q", manifest.HomePage)
	}

	if manifest.BuiltUsing != "" && !isValidBuiltUsing(manifest.BuiltUsing) {
		return fmt.Errorf("builtUsing: invalid Built-Using line %q", manifest.BuiltUsing)
	}

	if manifest.ShortDescription == "" {
		return fmt.Errorf("shortDescription: missing required field")
	}
	if !isValidDescriptionLine(manifest.ShortDescription) {
		return fmt.Errorf("shortDescription: invalid Description line %q", manifest.ShortDescription)
	}

	for index, line := range manifest.LongDescription {
		if !isValidDescriptionLine(line) {
			return fmt.Errorf("longDescription[%d]: invalid Description continuation line %q", index, line)
		}
	}

	return nil
}

func (manifest *Manifest) validatePost() error {
	knownDirectories := make(map[string]struct{}, 64)
	knownDirectories["."] = struct{}{}
	for index, dir := range manifest.ImplicitDirs {
		if !isValidUnixPath(dir) {
			return fmt.Errorf("implicitDirs[%d]: invalid Unix path %q", index, dir)
		}
		dir = strings.TrimRight(dir, "/")
		knownDirectories[dir] = struct{}{}
	}

	seen := make(map[string]int, 64)
	for index, file := range manifest.Files {
		name := file.Name
		if oldIndex, exists := seen[name]; exists {
			return fmt.Errorf("files[%d]: duplicate file %q has the same name as files[%d]", index, name, oldIndex)
		}
		seen[name] = index

		name = strings.TrimRight(name, "/")
		dir := path.Dir(name)
		if _, exists := knownDirectories[dir]; !exists {
			return fmt.Errorf("files[%d]: directory %q might not exist yet", index, dir)
		}
		if file.Type == TypeDIR {
			knownDirectories[name] = struct{}{}
		}
	}

	return nil
}

func (manifest Manifest) ControlFile() []byte {
	if !manifest.isResolved {
		panic(fmt.Errorf("must call Resolve first"))
	}

	var buf bytes.Buffer
	buf.WriteString("Package: ")
	buf.WriteString(manifest.Package)
	buf.WriteString("\n")
	buf.WriteString("Version: ")
	buf.WriteString(manifest.Version)
	buf.WriteString("\n")
	if manifest.Section != "" {
		buf.WriteString("Section: ")
		buf.WriteString(manifest.Section)
		buf.WriteString("\n")
	}
	if manifest.Priority != "" {
		buf.WriteString("Priority: ")
		buf.WriteString(manifest.Priority)
		buf.WriteString("\n")
	}
	if manifest.Arch != "" {
		buf.WriteString("Architecture: ")
		buf.WriteString(manifest.Arch)
		buf.WriteString("\n")
	}
	if manifest.Essential != "" {
		buf.WriteString("Essential: ")
		buf.WriteString(manifest.Essential)
		buf.WriteString("\n")
	}
	if manifest.Depends != "" {
		buf.WriteString("Depends: ")
		buf.WriteString(manifest.Depends)
		buf.WriteString("\n")
	}
	if manifest.PreDepends != "" {
		buf.WriteString("Pre-Depends: ")
		buf.WriteString(manifest.PreDepends)
		buf.WriteString("\n")
	}
	if manifest.Recommends != "" {
		buf.WriteString("Recommends: ")
		buf.WriteString(manifest.Recommends)
		buf.WriteString("\n")
	}
	if manifest.Suggests != "" {
		buf.WriteString("Suggests: ")
		buf.WriteString(manifest.Suggests)
		buf.WriteString("\n")
	}
	if manifest.Enhances != "" {
		buf.WriteString("Enhances: ")
		buf.WriteString(manifest.Enhances)
		buf.WriteString("\n")
	}
	if manifest.Breaks != "" {
		buf.WriteString("Breaks: ")
		buf.WriteString(manifest.Breaks)
		buf.WriteString("\n")
	}
	if manifest.Conflicts != "" {
		buf.WriteString("Conflicts: ")
		buf.WriteString(manifest.Conflicts)
		buf.WriteString("\n")
	}
	buf.WriteString("Installed-Size: ")
	buf.WriteString(strconv.FormatInt(manifest.installedSize, 10))
	buf.WriteString("\n")
	buf.WriteString("Maintainer: ")
	buf.WriteString(manifest.Maintainer)
	buf.WriteString("\n")
	if manifest.HomePage != "" {
		buf.WriteString("Homepage: ")
		buf.WriteString(manifest.HomePage)
		buf.WriteString("\n")
	}
	if manifest.BuiltUsing != "" {
		buf.WriteString("Built-Using: ")
		buf.WriteString(manifest.BuiltUsing)
		buf.WriteString("\n")
	}
	buf.WriteString("Description: ")
	buf.WriteString(manifest.ShortDescription)
	buf.WriteString("\n")
	for _, line := range manifest.LongDescription {
		if line == "" {
			buf.WriteString(" .\n")
		} else {
			buf.WriteString(" ")
			buf.WriteString(line)
			buf.WriteString("\n")
		}
	}
	return buf.Bytes()
}

func (manifest Manifest) ConfFiles() []byte {
	if !manifest.isResolved {
		panic(fmt.Errorf("must call Resolve first"))
	}

	var buf bytes.Buffer
	for _, file := range manifest.Files {
		if file.IsConf {
			buf.WriteString(file.Name)
			buf.WriteString("\n")
		}
	}
	return buf.Bytes()
}

func (manifest Manifest) PreInstallScript() []byte {
	if !manifest.isResolved {
		panic(fmt.Errorf("must call Resolve first"))
	}

	lines := manifest.PreInstall
	if len(lines) <= 0 {
		return nil
	}

	var buf bytes.Buffer
	buf.Grow(4096)
	buf.WriteString("#!/bin/bash\n")
	buf.WriteString("set -euo pipefail\n")
	buf.WriteString("umask 022\n")
	buf.WriteString("cd /\n")
	for _, line := range lines {
		buf.WriteString(line)
		buf.WriteString("\n")
	}
	return buf.Bytes()
}

func (manifest Manifest) PostInstallScript() []byte {
	if !manifest.isResolved {
		panic(fmt.Errorf("must call Resolve first"))
	}

	lines := manifest.PostInstall
	if len(lines) <= 0 {
		return nil
	}

	var buf bytes.Buffer
	buf.Grow(4096)
	buf.WriteString("#!/bin/bash\n")
	buf.WriteString("set -euo pipefail\n")
	buf.WriteString("umask 022\n")
	buf.WriteString("cd /\n")
	for _, line := range lines {
		buf.WriteString(line)
		buf.WriteString("\n")
	}
	return buf.Bytes()
}

func (manifest Manifest) PreRemoveScript() []byte {
	if !manifest.isResolved {
		panic(fmt.Errorf("must call Resolve first"))
	}

	lines := manifest.PreRemove
	if len(lines) <= 0 {
		return nil
	}

	var buf bytes.Buffer
	buf.Grow(4096)
	buf.WriteString("#!/bin/bash\n")
	buf.WriteString("set -euo pipefail\n")
	buf.WriteString("umask 022\n")
	buf.WriteString("cd /\n")
	for _, line := range lines {
		buf.WriteString(line)
		buf.WriteString("\n")
	}
	return buf.Bytes()
}

func (manifest Manifest) PostRemoveScript() []byte {
	if !manifest.isResolved {
		panic(fmt.Errorf("must call Resolve first"))
	}

	lines := manifest.PostRemove
	if len(lines) <= 0 {
		return nil
	}

	var buf bytes.Buffer
	buf.Grow(4096)
	buf.WriteString("#!/bin/bash\n")
	buf.WriteString("set -euo pipefail\n")
	buf.WriteString("umask 022\n")
	buf.WriteString("cd /\n")
	for _, line := range lines {
		buf.WriteString(line)
		buf.WriteString("\n")
	}
	return buf.Bytes()
}
