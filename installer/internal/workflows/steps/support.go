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

// const utf8CharMaxSize = 4

// type cutDirection bool

// const (
// 	cutLeftToRight cutDirection = true
// 	cutRightToLeft cutDirection = false
// )
