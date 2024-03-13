package release

import (
	"strings"
)

func deriveRepoName(repoUrl string) string {
	idx := strings.LastIndexByte(repoUrl, '.')
	if idx > 0 {
		idx = strings.LastIndexByte(repoUrl[0:idx], '.')
	}
	return strings.ReplaceAll(repoUrl[idx+1:], ".", "-")
}

func isKrateoGateway(n string) bool {
	return strings.HasPrefix(n, "gateway-") ||
		strings.HasSuffix(n, "-gateway")
}

func isVcluster(n string) bool {
	return strings.HasPrefix(n, "vcluster-") ||
		strings.HasSuffix(n, "-vcluster")
}
