package steps

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/twmb/murmur3"
)

func computeDigest(dat []byte) string {
	hasher := murmur3.New64()
	hasher.Write(dat)
	hasher.Sum64()
	return strconv.FormatUint(hasher.Sum64(), 16)
}

func strval(v any) string {
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

func deriveRepoName(repoUrl string) string {
	idx := strings.LastIndexByte(repoUrl, '.')
	if idx > 0 {
		idx = strings.LastIndexByte(repoUrl[0:idx], '.')
	}
	return strings.ReplaceAll(repoUrl[idx+1:], ".", "-")
}

// Ending ellipsis a long string s -> "front..."
func ellipsis(s string, n int) string {
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
