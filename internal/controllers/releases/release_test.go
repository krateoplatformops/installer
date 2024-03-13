package releases

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/krateoplatformops/installer/internal/helmclient"
	"github.com/krateoplatformops/installer/internal/helmclient/values"
	"helm.sh/helm/v3/pkg/repo"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func TestKrateoInstall(t *testing.T) {
	cli, err := newHelmClient()
	if err != nil {
		t.Fatal(err)
	}

	err = cli.AddOrUpdateChartRepo(repo.Entry{
		Name: "krateo",
		URL:  "https://charts.krateo.io",
	})
	if err != nil {
		t.Fatal(err)
	}

	err = cli.AddOrUpdateChartRepo(repo.Entry{
		Name: "loft-sh",
		URL:  "https://charts.loft.sh",
	})
	if err != nil {
		t.Fatal(err)
	}

	err = installVcluster(cli)
	if err != nil {
		t.Fatal(err)
	}
}

func installVcluster(cli helmclient.Client) error {
	chartSpec := helmclient.ChartSpec{
		ReleaseName:     "krateo-vcluster",
		ChartName:       "loft-sh/vcluster-k8s",
		Namespace:       "krateo-system",
		Version:         "0.19.4",
		CreateNamespace: true,
		UpgradeCRDs:     true,
		Wait:            true,
		ValuesOptions: values.Options{
			Values: []string{"service.type=ClusterIP"},
		},
		Timeout: time.Duration(10 * time.Minute),
	}

	_, err := cli.InstallOrUpgradeChart(context.Background(), &chartSpec, nil)
	return err
}

func newHelmClient() (helmclient.Client, error) {
	rc, err := newRestConfig()
	if err != nil {
		return nil, err
	}

	opt := &helmclient.RestConfClientOptions{
		Options: &helmclient.Options{
			Namespace:        "krateo-system", // Change this to the namespace you wish the client to operate in.
			RepositoryCache:  "/tmp/.helmcache",
			RepositoryConfig: "/tmp/.helmrepo",
			Debug:            true,
			Linting:          true, // Change this to false if you don't want linting.
			DebugLog: func(format string, v ...interface{}) {
				// Change this to your own logger. Default is 'log.Printf(format, v...)'.
				log.Printf(format, v...)
			},
		},
		RestConfig: rc,
	}

	return helmclient.NewClientFromRestConf(opt)
}

func newRestConfig() (*rest.Config, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	return clientcmd.BuildConfigFromFlags("", filepath.Join(home, ".kube", "config"))
}
