package workflows

import (
	"fmt"
	"log"

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
			if opts.verbose {
				log.Printf("DBG: %s", fmt.Sprintf(format, v...))
			}
		},
	}

	return helmclient.NewClientFromRestConf(&helmclient.RestConfClientOptions{
		Options: ho, RestConfig: opts.restConfig,
	})
}
