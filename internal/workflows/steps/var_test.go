//go:build integration
// +build integration

package steps

import (
	"context"
	"fmt"
	"testing"

	"github.com/krateoplatformops/installer/internal/cache"
	"github.com/krateoplatformops/installer/internal/dynamic"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestVarExpand(t *testing.T) {
	dat, err := loadSample("var-subst.json")
	if err != nil {
		t.Fatal(err)
	}

	env := cache.New[string, string]()
	env.Set("KUBECONFIG_KUBERNETES_IP", "127.0.0.1")

	vr := VarHandler(nil, env)
	err = vr.Handle(context.TODO(), "test", &runtime.RawExtension{
		Raw: dat,
	})
	if err != nil {
		t.Fatal(err)
	}

	env.ForEach(func(k, v string) bool {
		fmt.Printf("==> %s: %s\n", k, v)
		return true
	})
}

func TestVarResolver(t *testing.T) {
	dat, err := loadSample("var-value-from-secret.json")
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

	vr := &varStepHandler{
		dyn: dyn,
		env: env,
		ns:  "krateo-system",
	}

	err = vr.Handle(context.TODO(), "test", &runtime.RawExtension{
		Raw: dat,
	})
	if err != nil {
		t.Fatal(err)
	}

	env.ForEach(func(k, v string) bool {
		fmt.Printf("==> %s: %s\n", k, v)
		return true
	})
}
