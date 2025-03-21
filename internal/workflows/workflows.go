package workflows

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"

	"github.com/krateoplatformops/installer/apis/workflows/v1alpha1"
	"github.com/krateoplatformops/installer/internal/cache"
	"github.com/krateoplatformops/installer/internal/dynamic"
	"github.com/krateoplatformops/installer/internal/workflows/steps"
	"github.com/krateoplatformops/provider-runtime/pkg/logging"
	"k8s.io/client-go/rest"
)

func New(rc *rest.Config, ns string, logr logging.Logger, maxHelmHistory int) (*Workflow, error) {
	dyn, err := dynamic.NewGetter(rc)
	if err != nil {
		return nil, err
	}

	app, err := dynamic.NewApplier(rc)
	if err != nil {
		return nil, err
	}

	del, err := dynamic.NewDeletor(rc)
	if err != nil {
		return nil, err
	}

	cli, err := newHelmClient(helmClientOptions{
		namespace:  ns,
		restConfig: rc,
		logr:       logr,
	})
	if err != nil {
		return nil, err
	}

	env := cache.New[string, string]()

	return &Workflow{
		logr: logr,
		env:  env,
		ns:   ns,
		reg: map[v1alpha1.StepType]steps.Handler{
			v1alpha1.TypeVar:    steps.VarHandler(dyn, env, logr),
			v1alpha1.TypeObject: steps.ObjectHandler(app, del, env, logr),
			v1alpha1.TypeChart: steps.ChartHandler(steps.ChartHandlerOptions{
				HelmClient: cli,
				Env:        env,
				Log:        logr,
				Dyn:        dyn,
			}),
		},
		maxHistory: &maxHelmHistory,
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

func Err(results []StepResult) error {
	for _, x := range results {
		if x.Err() != nil {
			return fmt.Errorf("%s: %w", x.ID(), x.Err())
		}
	}

	return nil
}

type Workflow struct {
	logr       logging.Logger
	ns         string
	env        *cache.Cache[string, string]
	reg        map[v1alpha1.StepType]steps.Handler
	op         steps.Op
	maxHistory *int
}

func (wf *Workflow) Op(op steps.Op) {
	wf.op = op
}

func (wf *Workflow) Run(ctx context.Context, spec *v1alpha1.WorkflowSpec, skip func(*v1alpha1.Step) bool) (results []StepResult) {
	results = make([]StepResult, len(spec.Steps))

	if wf.op == steps.Delete {
		slices.Reverse(spec.Steps)
	}

	for i, x := range spec.Steps {
		if skip(x) {
			wf.logr.Info(fmt.Sprintf("skipping step with id: %s (%v)", x.ID, x.Type))
			continue
		}

		wf.logr.Info(fmt.Sprintf("executing step with id: %s (%v)", x.ID, x.Type))

		results[i] = StepResult{id: x.ID}

		job, ok := wf.reg[x.Type]
		if !ok {
			results[i].err = fmt.Errorf("handler for step of type %q not found", x.Type)
			return
		}

		job.Namespace(wf.ns)
		job.Op(wf.op)

		if x.Type == v1alpha1.TypeChart {
			res := v1alpha1.ChartSpec{}
			err := json.Unmarshal(x.With.Raw, &res)
			if err != nil {
				results[i].err = err
				return
			}
			if res.MaxHistory == nil && wf.maxHistory != nil {
				res.MaxHistory = wf.maxHistory
			}
			x.With.Raw, err = json.Marshal(res)
			if err != nil {
				results[i].err = err
				return
			}
		}

		err := job.Handle(ctx, x.ID, x.With)
		if err != nil {
			results[i].err = err
			return
		}

		results[i].digest = x.Digest()
	}

	return
}
