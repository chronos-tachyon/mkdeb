package mkdeb

import (
	"regexp"
	"unicode"
)

func jsonIsNull(input []byte) bool {
	if len(input) <= 0 {
		return true
	}
	if len(input) == 4 && input[0] == 'n' && input[1] == 'u' && input[2] == 'l' && input[3] == 'l' {
		return true
	}
	return false
}

const nameComponent = `(?:[.-]|_+)?[0-9A-Za-z]+(?:(?:[.-]|_+)[0-9A-Za-z]+)*`

var (
	nameRx     = regexp.MustCompile(`^(?:[.]|(?:` + nameComponent + `/)*` + nameComponent + `)/?$`)
	packageRx  = regexp.MustCompile(`^[0-9a-z][0-9a-z]+(?:[.+-][0-9a-z]+)*$`)
	versionRx  = regexp.MustCompile(`^(?:[1-9][0-9]*[:])?[0-9][0-9A-Za-z]*(?:[.~+-][0-9A-Za-z]+)*$`)
	archRx     = regexp.MustCompile(`^[0-9A-Za-z]+(?:[-][0-9A-Za-z]+)*$`)
	sectionRx  = regexp.MustCompile(`^[0-9a-z]+(?:[/-][0-9a-z]+)*$`)
	priorityRx = regexp.MustCompile(`^(?:required|important|standard|optional|extra)$`)
)

func isValidUnixPath(str string) bool {
	return nameRx.MatchString(str)
}

func isValidPackage(str string) bool {
	return packageRx.MatchString(str)
}

func isValidVersion(str string) bool {
	return versionRx.MatchString(str)
}

func isValidArch(str string) bool {
	return archRx.MatchString(str)
}

func isValidSection(str string) bool {
	return sectionRx.MatchString(str)
}

func isValidPriority(str string) bool {
	return priorityRx.MatchString(str)
}

func isValidDepends(str string) bool {
	return true
}

func isValidMaintainer(str string) bool {
	return true
}

func isValidURL(str string) bool {
	return true
}

func isValidBuiltUsing(str string) bool {
	return true
}

func isValidDescriptionLine(str string) bool {
	spaceCount := 0
	for _, ch := range str {
		if unicode.Is(unicode.Cc, ch) {
			return false
		}
		if unicode.Is(unicode.Z, ch) {
			spaceCount++
		} else {
			spaceCount = 0
		}
	}
	return (spaceCount <= 0)
}
