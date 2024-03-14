//go:build integration
// +build integration

package dynamic

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"k8s.io/apimachinery/pkg/runtime/schema"
)

// kubectl apply -f testdata/ns.yaml
// kubectl apply -f testdata/fireworksapp.crd.yaml
func TestApplier(t *testing.T) {
	fin, err := os.Open("../../../testdata/fireworksapp.fake.json")
	if err != nil {
		t.Fatal(err)
	}
	defer fin.Close()

	var content map[string]any
	err = json.NewDecoder(fin).Decode(&content)
	if err != nil {
		t.Fatal(err)
	}
	//spew.Dump(content)

	rc, err := newRestConfig()
	if err != nil {
		t.Fatal(err)
	}

	dyn, err := NewApplier(rc)
	if err != nil {
		t.Fatal(err)
	}

	err = dyn.Apply(context.TODO(), map[string]any{"spec": content}, ApplyOptions{
		GVK: schema.GroupVersionKind{
			Group:   "apps.krateo.io",
			Version: "v1alpha1",
			Kind:    "Fireworksapp",
		},
		Namespace: "demo-system",
		Name:      "demo-apply",
	})
	if err != nil {
		t.Fatal(err)
	}
}
