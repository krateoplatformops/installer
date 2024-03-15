package steps

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/krateoplatformops/installer/apis/workflows/v1alpha1"
	"github.com/krateoplatformops/installer/internal/cache"
	"github.com/twmb/murmur3"
)

func computeDigest(dat []byte) string {
	hasher := murmur3.New64()
	hasher.Write(dat)
	hasher.Sum64()
	return strconv.FormatUint(hasher.Sum64(), 16)
}

func resolveVars(res []*v1alpha1.Data, env *cache.Cache[string, string]) []string {
	all := make([]string, len(res))
	for i, el := range res {
		if len(el.Value) > 0 {
			val := el.Value
			if strings.HasPrefix(val, "$") && len(val) > 1 {
				val, _ = env.Get(el.Value[1:])
			}
			all[i] = fmt.Sprintf("%s=%s", el.Name, val)
		}
	}

	return all
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
