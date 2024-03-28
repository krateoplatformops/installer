//go:build integration
// +build integration

package steps

import (
	"context"
	"testing"

	"github.com/krateoplatformops/installer/internal/cache"
	"github.com/krateoplatformops/installer/internal/dynamic"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestObjectCreate(t *testing.T) {
	dat, err := loadSample("obj-sample.json")
	if err != nil {
		t.Fatal(err)
	}

	rc, err := newRestConfig()
	if err != nil {
		t.Fatal(err)
	}

	app, err := dynamic.NewApplier(rc)
	if err != nil {
		t.Fatal(err)
	}

	env := cache.New[string, string]()
	env.Set("KUBECONFIG_CAKEY", "XXXXX")

	oh := &objStepHandler{
		app: app,
		env: env,
		ns:  "krateo-system",
		op:  Create,
	}

	err = oh.Handle(context.TODO(), "test", &runtime.RawExtension{
		Raw: dat,
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestObjectDelete(t *testing.T) {
	dat, err := loadSample("obj-sample.json")
	if err != nil {
		t.Fatal(err)
	}

	rc, err := newRestConfig()
	if err != nil {
		t.Fatal(err)
	}

	del, err := dynamic.NewDeletor(rc)
	if err != nil {
		t.Fatal(err)
	}

	env := cache.New[string, string]()
	env.Set("KUBECONFIG_CAKEY", "XXXXX")

	oh := &objStepHandler{
		del:     del,
		env:     env,
		ns:      "krateo-system",
		op:      Delete,
		verbose: true,
	}

	err = oh.Handle(context.TODO(), "test", &runtime.RawExtension{
		Raw: dat,
	})
	if err != nil {
		t.Fatal(err)
	}
}
