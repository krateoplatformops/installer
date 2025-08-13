package workflows

import (
	"context"
	"fmt"
	"slices"

	"github.com/krateoplatformops/installer/apis/workflows/v1alpha1"
	"github.com/krateoplatformops/installer/internal/cache"
	"github.com/krateoplatformops/installer/internal/dynamic/applier"
	"github.com/krateoplatformops/installer/internal/dynamic/deletor"
	"github.com/krateoplatformops/installer/internal/dynamic/getter"
	"github.com/krateoplatformops/installer/internal/helmclient"
	"github.com/krateoplatformops/installer/internal/workflows/steps"
	charthandler "github.com/krateoplatformops/installer/internal/workflows/steps/chart"
	objecthandler "github.com/krateoplatformops/installer/internal/workflows/steps/object"
	varhandler "github.com/krateoplatformops/installer/internal/workflows/steps/var"

	"github.com/krateoplatformops/plumbing/ptr"
	"github.com/krateoplatformops/provider-runtime/pkg/logging"
)

type Opts struct {
	Getter         *getter.Getter
	Applier        *applier.Applier
	Deletor        *deletor.Deletor
	Log            logging.Logger
	HelmClient     helmclient.Client
	MaxHelmHistory int
	Namespace      string
}

func New(opts Opts) (*Workflow, error) {
	if opts.Getter == nil || opts.Applier == nil || opts.Deletor == nil {
		return nil, fmt.Errorf("dynamic getter, applier, or deletor cannot be nil")
	}

	if opts.HelmClient == nil {
		return nil, fmt.Errorf("helm client cannot be nil")
	}

	if opts.Log == nil {
		opts.Log = logging.NewNopLogger()
	}

	wf := &Workflow{
		logr:       opts.Log.WithValues("namespace", opts.Namespace),
		ns:         opts.Namespace,
		env:        cache.New[string, string](),
		maxHistory: ptr.To(opts.MaxHelmHistory),
	}

	wf.varHandler = varhandler.VarHandler(opts.Getter, wf.env, opts.Log)
	wf.objectHandler = objecthandler.ObjectHandler(opts.Applier, opts.Deletor, wf.env, opts.Log)
	wf.chartHandler = charthandler.ChartHandler(charthandler.ChartHandlerOptions{
		HelmClient: opts.HelmClient,
		Env:        wf.env,
		Log:        opts.Log,
		Dyn:        opts.Getter,
	})

	return wf, nil
}

type StepResult[T any] struct {
	id     string
	digest string
	err    error
	res    T
}

func (r *StepResult[T]) ID() string {
	return r.id
}

func (r *StepResult[T]) Digest() string {
	return r.digest
}

func (r *StepResult[T]) Err() error {
	return r.err
}

// Aggiungi questi metodi al StepResult

func (r *StepResult[T]) Result() T {
	return r.res
}

func Err[T any](results []StepResult[T]) error {
	for _, x := range results {
		if x.Err() != nil {
			return fmt.Errorf("%s: %w", x.ID(), x.Err())
		}
	}

	return nil
}

type Workflow struct {
	logr          logging.Logger
	ns            string
	env           *cache.Cache[string, string]
	varHandler    steps.Handler[*steps.VarResult]
	objectHandler steps.Handler[*steps.ObjectResult]
	chartHandler  steps.Handler[*steps.ChartResult]
	maxHistory    *int
	op            steps.Op
}

func (wf *Workflow) Op(op steps.Op) {
	wf.op = op
}

func (wf *Workflow) Run(ctx context.Context, spec *v1alpha1.WorkflowSpec, skip func(*v1alpha1.Step) bool) (results []StepResult[any]) {
	results = make([]StepResult[any], len(spec.Steps))

	if wf.op == steps.Delete {
		slices.Reverse(spec.Steps)
	}

	for i, x := range spec.Steps {
		if skip(x) {
			wf.logr.Debug(fmt.Sprintf("skipping step with id: %s (%v)", x.ID, x.Type))
			continue
		}

		wf.logr.Debug(fmt.Sprintf("executing step with id: %s (%v)", x.ID, x.Type))

		results[i] = StepResult[any]{id: x.ID}

		switch x.Type {
		case v1alpha1.TypeVar:
			wf.varHandler.Namespace(wf.ns)
			wf.varHandler.Op(wf.op)
			result, err := wf.varHandler.Handle(ctx, x.ID, x.With)
			results[i].res = result
			results[i].err = err

		case v1alpha1.TypeObject:
			wf.objectHandler.Namespace(wf.ns)
			wf.objectHandler.Op(wf.op)
			result, err := wf.objectHandler.Handle(ctx, x.ID, x.With)
			results[i].res = result
			results[i].err = err

		case v1alpha1.TypeChart:
			wf.chartHandler.Namespace(wf.ns)
			wf.chartHandler.Op(wf.op)
			result, err := wf.chartHandler.Handle(ctx, x.ID, x.With)
			results[i].res = result
			results[i].err = err

		default:
			results[i].err = fmt.Errorf("handler for step of type %q not found", x.Type)
		}

		if results[i].err != nil {
			return
		}
	}

	return
}
