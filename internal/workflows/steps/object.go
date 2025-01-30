package steps

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/krateoplatformops/installer/apis/workflows/v1alpha1"
	"github.com/krateoplatformops/installer/internal/cache"
	"github.com/krateoplatformops/installer/internal/dynamic"
	"github.com/krateoplatformops/installer/internal/expand"
	"github.com/krateoplatformops/installer/internal/ptr"
	"github.com/krateoplatformops/provider-runtime/pkg/logging"
	"helm.sh/helm/v3/pkg/strvals"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var _ Handler = (*objStepHandler)(nil)

func ObjectHandler(app *dynamic.Applier, del *dynamic.Deletor, env *cache.Cache[string, string], logr logging.Logger) Handler {
	return &objStepHandler{
		app: app, del: del, env: env,
		subst: func(k string) string {
			if v, ok := env.Get(k); ok {
				return v
			}

			return "$" + k
		},
		logr: logr,
	}
}

type objStepHandler struct {
	app   *dynamic.Applier
	del   *dynamic.Deletor
	env   *cache.Cache[string, string]
	ns    string
	op    Op
	subst func(k string) string
	logr  logging.Logger
}

func (r *objStepHandler) Namespace(ns string) {
	r.ns = ns
}

func (r *objStepHandler) Op(op Op) {
	r.op = op
}

func (r *objStepHandler) Handle(ctx context.Context, id string, ext *runtime.RawExtension) error {
	uns, err := r.toUnstructured(id, ext)
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

func (r *objStepHandler) toUnstructured(id string, ext *runtime.RawExtension) (*unstructured.Unstructured, error) {
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

	err = r.resolveVars(id, res.Set, src)
	if err != nil {
		return nil, err
	}

	r.logr.Debug(fmt.Sprintf("DBG [object:%s]: %v", id, src))

	return &unstructured.Unstructured{Object: src}, nil
}

func (r *objStepHandler) resolveVars(id string, res []*v1alpha1.Data, src map[string]any) error {
	for _, el := range res {
		if len(el.Value) > 0 {
			val := expand.Expand(el.Value, "", r.subst)
			line := fmt.Sprintf("%s=%s", el.Name, val)
			if ptr.Deref(el.AsString, false) {
				err := strvals.ParseIntoString(line, src)
				if err != nil {
					return err
				}
			} else {
				err := strvals.ParseInto(line, src)
				if err != nil {
					return err
				}
			}

			if r.op != Delete {
				r.logr.Debug(fmt.Sprintf(
					"DBG [object:%s]: prop (name: %s, value: %s)",
					id, el.Name, val))
			} else {
				r.logr.Debug(fmt.Sprintf(
					"DBG [object:%s]: prop (name: %s, value: %s), delete",
					id, el.Name, val),
				)
			}
		} else {
			r.logr.Debug(fmt.Sprintf(
				"DBG [object:%s]: prop (name: %s, value: %s)",
				id, el.Name, "no value"))
		}
	}

	return nil
}
