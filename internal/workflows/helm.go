package workflows

import (
	"fmt"

	"github.com/krateoplatformops/installer/internal/helmclient"
	"github.com/krateoplatformops/provider-runtime/pkg/logging"
	"k8s.io/client-go/rest"
)

type helmClientOptions struct {
	namespace  string
	restConfig *rest.Config
	logr       logging.Logger
	verbose    bool
}

func newHelmClient(opts helmClientOptions) (helmclient.Client, error) {
	l := logging.NewNopLogger()
	if opts.logr != nil {
		l = opts.logr.WithValues("namespace", opts.namespace)
	}
	ho := &helmclient.Options{
		Namespace:        opts.namespace,
		RepositoryCache:  "/tmp/.helmcache",
		RepositoryConfig: "/tmp/.helmrepo",
		Debug:            opts.verbose,
		Linting:          false,
		DebugLog: func(format string, v ...interface{}) {
			if opts.verbose {
				l.Debug(fmt.Sprintf("DBG: %s", fmt.Sprintf(format, v...)))
			}
		},
	}

	return helmclient.NewClientFromRestConf(&helmclient.RestConfClientOptions{
		Options: ho, RestConfig: opts.restConfig,
	})
}
