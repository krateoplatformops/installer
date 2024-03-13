package release

import (
	"fmt"

	"github.com/krateoplatformops/installer/internal/helmclient"
	"github.com/krateoplatformops/provider-runtime/pkg/logging"
	"k8s.io/client-go/rest"
)

type helmClientOptions struct {
	namespace  string
	restConfig *rest.Config
	log        logging.Logger
	verbose    bool
}

func newHelmClient(opts helmClientOptions) (helmclient.Client, error) {
	ho := &helmclient.Options{
		Namespace:        opts.namespace,
		RepositoryCache:  "/tmp/.helmcache",
		RepositoryConfig: "/tmp/.helmrepo",
		Debug:            true,
		Linting:          false,
		DebugLog: func(format string, v ...interface{}) {
			if !opts.verbose {
				return
			}

			if len(v) > 0 {
				opts.log.Debug(fmt.Sprintf(format, v))
			} else {
				opts.log.Debug(format)
			}
		},
	}

	return helmclient.NewClientFromRestConf(&helmclient.RestConfClientOptions{
		Options: ho, RestConfig: opts.restConfig,
	})
}
