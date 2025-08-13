package steps

import (
	"fmt"
	"path"
	"strings"
)

func Strval(v any) string {
	switch v := v.(type) {
	case string:
		return v
	case []byte:
		return string(v)
	case error:
		return v.Error()
	case fmt.Stringer:
		return v.String()
	default:
		return fmt.Sprintf("%v", v)
	}
}

// write a func to  bring this to krateo-bff https://raw.githubusercontent.com/matteogastaldello/private-charts/main/krateo-bff-0.18.1.tgz
func DeriveReleaseName(repoUrl string) string {
	releaseName := strings.TrimSuffix(path.Base(repoUrl), ".tgz")
	versionIndex := strings.LastIndex(releaseName, "-")
	releaseName = releaseName[:versionIndex]

	return releaseName
}

func DeriveRepoName(repoUrl string) string {

	idx1 := strings.LastIndexByte(repoUrl, '.')
	idx2 := strings.LastIndexByte(repoUrl, '/')
	if idx1 > idx2 {
		idx := strings.LastIndexByte(repoUrl[0:idx1], '.')
		return strings.ReplaceAll(repoUrl[idx+1:], ".", "-")
	}

	return strings.ReplaceAll(repoUrl[idx2+1:], ".", "-")
}

// Ending ellipsis a long string s -> "front..."
func Ellipsis(s string, n int) string {
	if n <= 3 {
		return "..."
	}

	if len(s) <= n {
		return s
	}

	n -= 3

	var sb strings.Builder
	sb.WriteString(cutString(s, n, cutLeftToRight))
	sb.WriteString("...")

	return sb.String()
}

const utf8CharMaxSize = 4

type cutDirection bool

const (
	cutLeftToRight cutDirection = true
	cutRightToLeft cutDirection = false
)

// cutString cuts a string s into a string of n utf-8 runes.
func cutString(s string, n int, leftToRight cutDirection) string {
	if n <= 0 {
		return ""
	}

	if n >= len(s) {
		return s
	}

	maxLen := n * utf8CharMaxSize
	if maxLen >= len(s) {
		maxLen = len(s)
	}

	var runes []rune
	if leftToRight {
		runes = []rune(s[:maxLen])
		if len(runes) > n {
			runes = runes[:n]
		}
	} else {
		runes = []rune(s[len(s)-maxLen:])
		if len(runes) > n {
			runes = runes[len(runes)-n:]
		}
	}

	return string(runes)
}
