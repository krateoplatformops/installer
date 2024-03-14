//go:build integration
// +build integration

package workflow

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/krateoplatformops/installer/apis/releases/v1alpha1"
	"github.com/krateoplatformops/installer/internal/cache"
	"github.com/krateoplatformops/installer/internal/kubernetes/dynamic"
	"github.com/krateoplatformops/installer/internal/ptr"
	"k8s.io/apimachinery/pkg/runtime"
	jsonserializer "k8s.io/apimachinery/pkg/runtime/serializer/json"
)

func TestVarResolver(t *testing.T) {
	dat, err := loadSample("var-value-from-secret.json")
	if err != nil {
		t.Fatal(err)
	}

	res := v1alpha1.Var{}
	err = json.Unmarshal(dat, &res)
	if err != nil {
		t.Fatal(err)
	}

	rc, err := newRestConfig()
	if err != nil {
		t.Fatal(err)
	}

	dyn, err := dynamic.NewGetter(rc)
	if err != nil {
		t.Fatal(err)
	}

	env := cache.New[string, string]()

	vr := &varHandler{
		dyn: dyn,
		env: env,
		ns:  "krateo-system",
	}

	err = vr.Do(context.TODO(), &res)
	if err != nil {
		t.Fatal(err)
	}

	env.ForEach(func(k, v string) bool {
		fmt.Printf("==> %s: %s\n", k, v)
		return true
	})
}

func TestDecodeVar(t *testing.T) {
	dat, err := loadSample("task-sample.yaml")
	if err != nil {
		t.Fatal(err)
	}
	obj, err := decodeYAML(dat)
	if err != nil {
		t.Fatal(err)
	}

	for _, step := range obj.Spec.Steps {
		fmt.Printf("step: %s (%s)\n", ptr.Deref(step.ID, ""), step.Type)
		// switch step.Type {
		// case v1alpha1.TypeVar:
		// 	err := handleVar(step.With)
		// 	if err != nil {
		// 		t.Fatal(err)
		// 	}
		// }
	}
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
