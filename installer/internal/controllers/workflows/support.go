package workflows

import (
	"fmt"
	"strconv"

	"github.com/krateoplatformops/installer/apis/workflows/v1alpha1"
	"github.com/krateoplatformops/installer/internal/helmclient"
	"github.com/krateoplatformops/provider-runtime/pkg/logging"
	"github.com/twmb/murmur3"
	"k8s.io/client-go/rest"
)

func digestForSteps(cr *v1alpha1.KrateoPlatformOps) string {
	hasher := murmur3.New64()

	for _, x := range cr.Spec.Steps {
		hasher.Write([]byte(x.Digest()))
	}

	return strconv.FormatUint(hasher.Sum64(), 16)
}

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
