package steps

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/krateoplatformops/installer/apis/workflows/v1alpha1"
	"github.com/krateoplatformops/installer/internal/cache"
	"github.com/krateoplatformops/installer/internal/dynamic"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var _ Handler = (*varStepHandler)(nil)

func VarHandler(dyn *dynamic.Getter, env *cache.Cache[string, string]) Handler {
	return &varStepHandler{
		dyn: dyn, env: env,
	}
}

type varStepHandler struct {
	dyn *dynamic.Getter
	env *cache.Cache[string, string]
	ns  string
}

func (r *varStepHandler) Namespace(ns string) {
	r.ns = ns
}

func (r *varStepHandler) Handle(ctx context.Context, id string, ext *runtime.RawExtension) error {
	res := v1alpha1.Var{}
	err := json.Unmarshal(ext.Raw, &res)
	if err != nil {
		return err
	}

	if len(res.Value) > 0 {
		val := res.Value
		if strings.HasPrefix(val, "$") && len(val) > 1 {
			val, _ = r.env.Get(res.Value[1:])
		}
		r.env.Set(res.Name, val)
	}

	if res.ValueFrom == nil {
		return nil
	}

	gv, err := schema.ParseGroupVersion(res.ValueFrom.APIVersion)
	if err != nil {
		return err
	}

	namespace := res.ValueFrom.Metadata.Namespace
	if len(namespace) == 0 {
		namespace = r.ns
	}

	name := res.ValueFrom.Metadata.Name

	obj, err := r.dyn.Get(ctx, dynamic.GetOptions{
		Name:      name,
		Namespace: namespace,
		GVK:       gv.WithKind(res.ValueFrom.Kind),
	})
	if err != nil {
		return err
	}

	val, err := dynamic.Extract(ctx, obj, res.ValueFrom.Selector)
	if val != nil {
		r.env.Set(res.Name, strval(val))
	}

	return err
}
