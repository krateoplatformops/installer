//go:build integration
// +build integration

package workflows

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/krateoplatformops/installer/apis/workflows/v1alpha1"
	"github.com/krateoplatformops/provider-runtime/pkg/meta"
	"k8s.io/apimachinery/pkg/runtime"
	jsonserializer "k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func TestWorkflow(t *testing.T) {
	dat, err := loadSample("krateo.yaml")
	if err != nil {
		t.Fatal(err)
	}

	res, err := decodeYAML(dat)
	if err != nil {
		t.Fatal(err)
	}

	rc, err := newRestConfig()
	if err != nil {
		t.Fatal(err)
	}

	wf, err := New(rc, res.GetNamespace(), meta.IsVerbose(res))
	if err != nil {
		t.Fatal(err)
	}

	results := wf.Run(context.TODO(), res.Spec.DeepCopy(), func(s *v1alpha1.Step) bool {
		return false
	})

	err = Err(results)
	if err != nil {
		t.Fatal(err)
	}

	wf.env.ForEach(func(k, v string) bool {

		fmt.Printf("k: %s, v: %s\n", k, v)
		return true
	})
	fmt.Println()
	spew.Dump(res)
}

func decodeYAML(dat []byte) (*v1alpha1.KrateoPlatformOps, error) {
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

func loadSample(fn string) ([]byte, error) {
	fin, err := os.Open(filepath.Join("..", "..", "testdata", fn))
	if err != nil {
		return nil, err
	}
	defer fin.Close()

	return io.ReadAll(fin)
}

func newRestConfig() (*rest.Config, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	return clientcmd.BuildConfigFromFlags("", filepath.Join(home, ".kube", "config"))
}
