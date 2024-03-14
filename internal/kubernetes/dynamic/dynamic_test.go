//go:build integration
// +build integration

package dynamic

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func TestGet(t *testing.T) {
	rc, err := newRestConfig()
	if err != nil {
		t.Fatal(err)
	}

	dyn, err := NewGetter(rc)
	if err != nil {
		t.Fatal(err)
	}

	obj, err := dyn.Get(context.TODO(), GetOptions{
		GVK: schema.GroupVersionKind{
			Group:   "apiextensions.k8s.io",
			Version: "v1",
			Kind:    "CustomResourceDefinition",
		},
		Name: "formtemplates.widgets.ui.krateo.io",
	})
	if err != nil {
		t.Fatal(err)
	}

	filter := `.spec.versions[] | select(.name="v1alpha1") | .schema.openAPIV3Schema`
	xxx, err := Extract(context.TODO(), obj, filter)
	if err != nil {
		t.Fatal(err)
	}

	spew.Dump(xxx)
}

func newRestConfig() (*rest.Config, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	return clientcmd.BuildConfigFromFlags("", filepath.Join(home, ".kube", "config"))
}
