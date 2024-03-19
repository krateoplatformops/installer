package steps

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/krateoplatformops/installer/apis/workflows/v1alpha1"
	"github.com/krateoplatformops/installer/internal/cache"
	"github.com/krateoplatformops/installer/internal/dynamic"
	"helm.sh/helm/v3/pkg/strvals"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var _ Handler = (*objStepHandler)(nil)

func ObjectHandler(app *dynamic.Applier, del *dynamic.Deletor, env *cache.Cache[string, string]) Handler {
	return &objStepHandler{
		app: app, del: del, env: env,
	}
}

type objStepHandler struct {
	app *dynamic.Applier
	del *dynamic.Deletor
	env *cache.Cache[string, string]
	ns  string
	op  Op
}

func (r *objStepHandler) Namespace(ns string) {
	r.ns = ns
}

func (r *objStepHandler) Op(op Op) {
	r.op = op
}

func (r *objStepHandler) Handle(ctx context.Context, id string, ext *runtime.RawExtension) error {
	uns, err := r.toUnstructured(ext)
	if err != nil {
		return err
	}

	gv, err := schema.ParseGroupVersion(uns.GetAPIVersion())
	if err != nil {
		return err
	}

	if r.op == Delete {
		err := r.del.Delete(ctx, dynamic.DeleteOptions{
			GVK:       gv.WithKind(uns.GetKind()),
			Namespace: uns.GetNamespace(),
			Name:      uns.GetName(),
		})
		if apierrors.IsNotFound(err) {
			err = nil
		}
		return err
	}

	return r.app.Apply(ctx, uns.Object, dynamic.ApplyOptions{
		GVK:       gv.WithKind(uns.GetKind()),
		Namespace: uns.GetNamespace(),
		Name:      uns.GetName(),
	})
}

func (r *objStepHandler) toUnstructured(ext *runtime.RawExtension) (*unstructured.Unstructured, error) {
	res := v1alpha1.Object{}
	err := json.Unmarshal(ext.Raw, &res)
	if err != nil {
		return nil, err
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
		return nil, err
	}

	return &unstructured.Unstructured{Object: src}, nil
}
