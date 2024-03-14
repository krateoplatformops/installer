package workflow

import (
	"context"
	"encoding/json"

	"github.com/krateoplatformops/installer/apis/releases/v1alpha1"
	"github.com/krateoplatformops/installer/internal/cache"
	"github.com/krateoplatformops/installer/internal/kubernetes/dynamic"
	"k8s.io/client-go/rest"
)

func NewStepHandler(rc *rest.Config, env *cache.Cache[string, string]) (*StepHandler, error) {
	dyn, err := dynamic.NewGetter(rc)
	if err != nil {
		return nil, err
	}

	app, err := dynamic.NewApplier(rc)
	if err != nil {
		return nil, err
	}

	return &StepHandler{
		forVar: &varHandler{
			dyn: dyn, env: env,
		},
		forObject: &objHandler{
			app: app, env: env,
		},
	}, nil
}

type StepHandler struct {
	ns        string
	forVar    *varHandler
	forObject *objHandler
}

func (h *StepHandler) Namespace(ns string) {
	h.ns = ns
}

func (h *StepHandler) Handle(ctx context.Context, in *v1alpha1.Step) (err error) {
	if in == nil || in.With == nil {
		return nil
	}

	switch in.Type {
	case v1alpha1.TypeVar:
		res := v1alpha1.Var{}
		err = json.Unmarshal(in.With.Raw, &res)
		if err != nil {
			return err
		}

		err := h.forVar.Namespace(h.ns).Do(ctx, &res)
		if err != nil {
			return err
		}

	case v1alpha1.TypeObject:
		res := v1alpha1.Object{}
		err = json.Unmarshal(in.With.Raw, &res)
		if err != nil {
			return err
		}

		err := h.forObject.Namespace(h.ns).Do(ctx, &res)
		if err != nil {
			return err
		}
	}

	return nil
}
