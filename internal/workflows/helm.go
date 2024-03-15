package workflows

import (
	"fmt"
	"os"

	"github.com/krateoplatformops/installer/internal/helmclient"
	"k8s.io/client-go/rest"
)

type helmClientOptions struct {
	namespace  string
	restConfig *rest.Config
	verbose    bool
}

func newHelmClient(opts helmClientOptions) (helmclient.Client, error) {
	ho := &helmclient.Options{
		Namespace:        opts.namespace,
		RepositoryCache:  "/tmp/.helmcache",
		RepositoryConfig: "/tmp/.helmrepo",
		Debug:            opts.verbose,
		Linting:          false,
		DebugLog: func(format string, v ...interface{}) {
			if !opts.verbose {
				return
			}

			fmt.Fprintf(os.Stderr, format, v...)
			fmt.Fprintln(os.Stderr)
		},
	}

	return helmclient.NewClientFromRestConf(&helmclient.RestConfClientOptions{
		Options: ho, RestConfig: opts.restConfig,
	})
}
