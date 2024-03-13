package releases

import (
	"fmt"

	"github.com/krateoplatformops/installer/internal/helmclient"
	"github.com/krateoplatformops/provider-runtime/pkg/logging"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
)

const (
	annotationKeyConnectorVerbose = "krateo.io/connector-verbose"
)

func helmClientForObject(rc *rest.Config, mo metav1.Object, log logging.Logger) (helmclient.Client, error) {
	verbose := mo.GetAnnotations()[annotationKeyConnectorVerbose] == "true"

	opts := &helmclient.Options{
		Namespace:        mo.GetNamespace(),
		RepositoryCache:  "/tmp/.helmcache",
		RepositoryConfig: "/tmp/.helmrepo",
		Debug:            true,
		Linting:          false,
		DebugLog: func(format string, v ...interface{}) {
			if !verbose {
				return
			}

			if len(v) > 0 {
				log.Debug(fmt.Sprintf(format, v))
			} else {
				log.Debug(format)
			}
		},
	}

	return helmclient.NewClientFromRestConf(&helmclient.RestConfClientOptions{
		Options: opts, RestConfig: rc,
	})
}
