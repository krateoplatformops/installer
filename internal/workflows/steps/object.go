package steps

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/krateoplatformops/installer/apis/workflows/v1alpha1"
	"github.com/krateoplatformops/installer/internal/cache"
	"github.com/krateoplatformops/installer/internal/dynamic"
	"helm.sh/helm/v3/pkg/strvals"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var _ Handler = (*objStepHandler)(nil)

func ObjectHandler(dyn *dynamic.Applier, env *cache.Cache[string, string]) Handler {
	return &objStepHandler{
		dyn: dyn, env: env,
	}
}

type objStepHandler struct {
	dyn *dynamic.Applier
	env *cache.Cache[string, string]
	ns  string
}

func (r *objStepHandler) Namespace(ns string) {
	r.ns = ns
}

func (r *objStepHandler) Handle(ctx context.Context, id string, ext *runtime.RawExtension) error {
	res := v1alpha1.Object{}
	err := json.Unmarshal(ext.Raw, &res)
	if err != nil {
		return err
	}

	namespace := res.Metadata.Namespace
	if len(namespace) == 0 {
		namespace = r.ns
	}

	src := map[string]any{
		"apiVersion": res.APIVersion,
		"kind":       res.Kind,
		"metadata": map[string]any{
			"name":      res.Metadata.Name,
			"namespace": namespace,
		},
	}

	all := resolveVars(res.Set, r.env)
	err = strvals.ParseInto(strings.Join(all, ","), src)
	if err != nil {
		return err
	}

	uns := unstructured.Unstructured{Object: src}

	gv, err := schema.ParseGroupVersion(uns.GetAPIVersion())
	if err != nil {
		return err
	}

	return r.dyn.Apply(ctx, uns.Object, dynamic.ApplyOptions{
		GVK:       gv.WithKind(uns.GetKind()),
		Namespace: uns.GetNamespace(),
		Name:      uns.GetName(),
	})
}
