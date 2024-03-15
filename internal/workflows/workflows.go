package workflows

import (
	"context"
	"fmt"

	"github.com/krateoplatformops/installer/apis/workflows/v1alpha1"
	"github.com/krateoplatformops/installer/internal/cache"
	"github.com/krateoplatformops/installer/internal/dynamic"
	"github.com/krateoplatformops/installer/internal/workflows/steps"
	"k8s.io/client-go/rest"
)

func New(rc *rest.Config, ns string, verbose bool) (*Workflow, error) {
	dyn, err := dynamic.NewGetter(rc)
	if err != nil {
		return nil, err
	}

	app, err := dynamic.NewApplier(rc)
	if err != nil {
		return nil, err
	}

	cli, err := newHelmClient(helmClientOptions{
		namespace:  ns,
		restConfig: rc,
		verbose:    verbose,
	})
	if err != nil {
		return nil, err
	}

	env := cache.New[string, string]()

	return &Workflow{
		env: env,
		rc:  rc,
		ns:  ns,
		reg: map[v1alpha1.StepType]steps.Handler{
			v1alpha1.TypeVar:    steps.VarHandler(dyn, env),
			v1alpha1.TypeObject: steps.ObjectHandler(app, env),
			v1alpha1.TypeChart:  steps.ChartHandler(cli, env),
		},
	}, nil
}

type StepResult struct {
	id     string
	digest string
	err    error
}

func (r *StepResult) ID() string {
	return r.id
}

func (r *StepResult) Digest() string {
	return r.digest
}

func (r *StepResult) Err() error {
	return r.err
}

func Err(results []StepResult) (string, error) {
	for _, x := range results {
		if x.Err() != nil {
			return x.ID(), x.Err()
		}
	}

	return "", nil
}

type Workflow struct {
	rc  *rest.Config
	ns  string
	env *cache.Cache[string, string]
	reg map[v1alpha1.StepType]steps.Handler
}

func (wf *Workflow) Run(ctx context.Context, spec *v1alpha1.WorkflowSpec) (results []StepResult) {
	results = make([]StepResult, len(spec.Steps))

	for i, x := range spec.Steps {
		results[i] = StepResult{id: x.ID}

		job, ok := wf.reg[x.Type]
		if !ok {
			results[i].err = fmt.Errorf("handler for step of type %q not found", x.Type)
			return
		}

		job.Namespace(wf.ns)

		err := job.Handle(ctx, x.ID, x.With)
		if err != nil {
			results[i].err = err
			return
		}

		results[i].digest = x.Digest()
	}

	return
}
