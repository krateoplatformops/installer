package steps

import (
	"io"
	"log"
	"os"
	"path/filepath"
	"testing"

	"github.com/krateoplatformops/installer/internal/helmclient"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func TestDeriveRepoName(t *testing.T) {
	table := []struct {
		in   string
		want string
	}{
		{"https://charts.loft.sh", "loft-sh"},
		{"https://https://charts.krateo.io", "krateo-io"},
	}

	for i, tc := range table {
		got := deriveRepoName(tc.in)
		if got != tc.want {
			t.Fatalf("[tc: %d] - got: %v, expected: %v", i, got, tc.want)
		}
	}
}

func helmClientForNamespace(rc *rest.Config, ns string) (helmclient.Client, error) {
	opt := &helmclient.RestConfClientOptions{
		Options: &helmclient.Options{
			Namespace:        ns,
			RepositoryCache:  "/tmp/.helmcache",
			RepositoryConfig: "/tmp/.helmrepo",
			Debug:            true,
			Linting:          false, // Change this to false if you don't want linting.
			DebugLog: func(format string, v ...interface{}) {
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

func loadSample(fn string) ([]byte, error) {
	fin, err := os.Open(filepath.Join("..", "..", "..", "testdata", fn))
	if err != nil {
		return nil, err
	}
	defer fin.Close()

	return io.ReadAll(fin)
}
