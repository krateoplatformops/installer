package release

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"testing"

	"github.com/krateoplatformops/installer/apis/releases/v1alpha1"
	"github.com/krateoplatformops/installer/internal/helmclient"
	"k8s.io/apimachinery/pkg/runtime"
	jsonserializer "k8s.io/apimachinery/pkg/runtime/serializer/json"
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

func decodeSample() (*v1alpha1.KrateoPlatformOps, error) {
	fin, err := os.Open("../../testdata/sample.yaml")
	if err != nil {
		return nil, err
	}
	defer fin.Close()

	dat, err := io.ReadAll(fin)
	if err != nil {
		return nil, err
	}

	s := runtime.NewScheme()
	s.AddKnownTypes(v1alpha1.SchemeGroupVersion,
		&v1alpha1.KrateoPlatformOps{},
		&v1alpha1.KrateoPlatformOpsList{},
	)

	serializer := jsonserializer.NewSerializerWithOptions(
		jsonserializer.DefaultMetaFactory, // jsonserializer.MetaFactory
		s,                                 // runtime.Scheme implements runtime.ObjectCreater
		s,                                 // runtime.Scheme implements runtime.ObjectTyper
		jsonserializer.SerializerOptions{
			Yaml:   true,
			Pretty: false,
			Strict: false,
		},
	)

	obj, _, err := serializer.Decode(dat, nil, nil)
	if err != nil {
		return nil, err
	}
	if obj == nil {
		return nil, fmt.Errorf("obj is nil")
	}

	res, ok := obj.(*v1alpha1.KrateoPlatformOps)
	if !ok {
		return nil, fmt.Errorf("unexpected type '%T' for obj", obj)
	}
	return res, err
}
