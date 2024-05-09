//go:build integration
// +build integration

package steps

import (
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-logr/logr"
	"github.com/go-logr/logr/funcr"
	"github.com/krateoplatformops/installer/internal/helmclient"
	"github.com/krateoplatformops/provider-runtime/pkg/logging"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
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

func stdoutLogger() logging.Logger {
	return logging.NewLogrLogger(newStdoutLogger())
}

func newStdoutLogger() logr.Logger {
	return funcr.New(func(prefix, args string) {
		if prefix != "" {
			fmt.Printf("%s: %s\n", prefix, args)
		} else {
			fmt.Println(args)
		}
	}, funcr.Options{})
}

func loadChart(f fs.FS) (*chart.Chart, error) {
	files := []*loader.BufferedFile{}

	err := fs.WalkDir(f, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		data, err := fs.ReadFile(f, path)
		if err != nil {
			return fmt.Errorf("could not read manifest %s: %w", path, err)
		}

		files = append(files, &loader.BufferedFile{
			Name: path,
			Data: data,
		})

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("could not walk chart directory: %w", err)
	}

	return loader.LoadFiles(files)
}
