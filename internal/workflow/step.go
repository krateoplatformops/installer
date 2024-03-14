package workflow

import (
	"context"

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

	return &StepHandler{
		forVar: &varHandler{
			dyn: dyn, env: env,
		},
	}, nil
}

type StepHandler struct {
	forVar *varHandler
}

func (h *StepHandler) Handle(ctx context.Context, in *v1alpha1.Step) (err error) {
	if in == nil || in.With == nil {
		return nil
	}

	// switch in.Type {
	// case v1alpha1.TypeVar:
	// 	err := h.forVar.Do(ctx, in.With)
	// 	if err != nil {
	// 		return err
	// 	}
	// case v1alpha1.TypeObject:

	// }

	return nil
}
