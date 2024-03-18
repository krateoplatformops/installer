package workflows

import (
	"context"
	"strings"
	"time"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/krateoplatformops/installer/apis/workflows/v1alpha1"
	workflowsv1alpha1 "github.com/krateoplatformops/installer/apis/workflows/v1alpha1"
	"github.com/krateoplatformops/installer/internal/ptr"
	"github.com/krateoplatformops/installer/internal/workflows"
	rtv1 "github.com/krateoplatformops/provider-runtime/apis/common/v1"
	"github.com/krateoplatformops/provider-runtime/pkg/controller"
	"github.com/krateoplatformops/provider-runtime/pkg/event"
	"github.com/krateoplatformops/provider-runtime/pkg/logging"
	"github.com/krateoplatformops/provider-runtime/pkg/meta"
	"github.com/krateoplatformops/provider-runtime/pkg/ratelimiter"
	"github.com/krateoplatformops/provider-runtime/pkg/reconciler"
	"github.com/krateoplatformops/provider-runtime/pkg/resource"
	"github.com/pkg/errors"
)

const (
	errNotCR = "managed resource is not a KrateoPlatformOps custom resource"

	reconcileGracePeriod = 1 * time.Minute
	reconcileTimeout     = 4 * time.Minute
)

func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := reconciler.ControllerName(workflowsv1alpha1.KrateoPlatformOpsKind)
	log := o.Logger.WithValues("controller", name)

	recorder := mgr.GetEventRecorderFor(name)

	r := reconciler.NewReconciler(mgr,
		resource.ManagedKind(workflowsv1alpha1.KrateoPlatformOpsGroupVersionKind),
		reconciler.WithExternalConnecter(&connector{
			kube:     mgr.GetClient(),
			log:      log,
			rc:       mgr.GetConfig(),
			recorder: recorder,
		}),
		reconciler.WithTimeout(reconcileTimeout),
		reconciler.WithCreationGracePeriod(reconcileGracePeriod),
		reconciler.WithPollInterval(o.PollInterval),
		reconciler.WithLogger(log),
		reconciler.WithRecorder(event.NewAPIRecorder(recorder)))

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(o.ForControllerRuntime()).
		For(&workflowsv1alpha1.KrateoPlatformOps{}).
		Complete(ratelimiter.NewReconciler(name, r, o.GlobalRateLimiter))
}

type connector struct {
	kube     client.Client
	rc       *rest.Config
	log      logging.Logger
	recorder record.EventRecorder
}

func (c *connector) Connect(ctx context.Context, mg resource.Managed) (reconciler.ExternalClient, error) {
	cr, ok := mg.(*workflowsv1alpha1.KrateoPlatformOps)
	if !ok {
		return nil, errors.New(errNotCR)
	}

	wf, err := workflows.New(c.rc, cr.GetNamespace(), meta.IsVerbose(cr))
	if err != nil {
		return nil, err
	}

	return &external{
		kube: c.kube,
		log:  c.log,
		wf:   wf,
		rec:  c.recorder,
	}, nil
}

type external struct {
	kube client.Client
	log  logging.Logger
	wf   *workflows.Workflow
	rec  record.EventRecorder
}

func (e *external) Observe(ctx context.Context, mg resource.Managed) (reconciler.ExternalObservation, error) {
	cr, ok := mg.(*workflowsv1alpha1.KrateoPlatformOps)
	if !ok {
		return reconciler.ExternalObservation{}, errors.New(errNotCR)
	}

	verbose := meta.IsVerbose(cr)

	got := currentDigestMap(cr)
	if len(got) == 0 {
		return reconciler.ExternalObservation{
			ResourceExists:   false,
			ResourceUpToDate: true,
		}, nil
	}

	exp := listOfStepIdToUpdate(cr)
	if verbose {
		e.log.Debug("List of step to update", "tot", len(exp), "list", strings.Join(exp, ","))
	}

	return reconciler.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: len(exp) == 0,
	}, nil
}

func (e *external) Create(ctx context.Context, mg resource.Managed) error {
	cr, ok := mg.(*workflowsv1alpha1.KrateoPlatformOps)
	if !ok {
		return errors.New(errNotCR)
	}

	if !meta.IsActionAllowed(cr, meta.ActionCreate) {
		e.log.Debug("External resource should not be updated by provider, skip creating.")
		return nil
	}

	cr.SetConditions(rtv1.Creating())

	results := e.wf.Run(ctx, cr.Spec.DeepCopy(), func(s *workflowsv1alpha1.Step) bool {
		return false
	})
	if id, err := workflows.Err(results); err != nil {
		e.log.Debug("Worflow failure", "step-id", id, "error", err.Error())
		return err
	}

	cr.Status.Steps = make(map[string]v1alpha1.StepStatus)

	for _, x := range results {
		nfo := v1alpha1.StepStatus{
			ID:     ptr.To(x.ID()),
			Digest: ptr.To(x.Digest()),
		}
		if err := x.Err(); err != nil {
			nfo.Err = ptr.To(err.Error())
		}

		cr.Status.Steps[x.ID()] = nfo
	}

	return e.kube.Status().Update(ctx, cr)
}

func (e *external) Update(ctx context.Context, mg resource.Managed) error {
	cr, ok := mg.(*workflowsv1alpha1.KrateoPlatformOps)
	if !ok {
		return errors.New(errNotCR)
	}

	if !meta.IsActionAllowed(cr, meta.ActionUpdate) {
		e.log.Debug("External resource should not be updated by provider, skip updating.")
		return nil
	}

	verbose := meta.IsVerbose(cr)
	_ = verbose

	cr = cr.DeepCopy()

	all := listOfStepIdToUpdate(cr)
	if len(all) == 0 {
		return nil
	}

	if verbose {
		e.log.Debug("Step(s) to update", "ids", strings.Join(all, ","))
	}

	results := e.wf.Run(ctx, cr.Spec.DeepCopy(), func(s *workflowsv1alpha1.Step) bool {
		for _, id := range all {
			if id == s.ID {
				if verbose {
					e.log.Debug("Step must NOT be skipped", "id", s.ID)
				}
				return false
			}
		}
		return true
	})

	if id, err := workflows.Err(results); err != nil {
		e.log.Debug("Worflow failure", "step-id", id, "error", err.Error())
		return err
	}

	cr.Status.Steps = make(map[string]v1alpha1.StepStatus)

	for _, x := range results {
		nfo := v1alpha1.StepStatus{
			ID:     ptr.To(x.ID()),
			Digest: ptr.To(x.Digest()),
		}
		if err := x.Err(); err != nil {
			nfo.Err = ptr.To(err.Error())
		}

		cr.Status.Steps[x.ID()] = nfo
	}

	return e.kube.Status().Update(ctx, cr)
}

func (e *external) Delete(ctx context.Context, mg resource.Managed) error {
	cr, ok := mg.(*workflowsv1alpha1.KrateoPlatformOps)
	if !ok {
		return errors.New(errNotCR)
	}

	if !meta.IsActionAllowed(cr, meta.ActionDelete) {
		e.log.Debug("External resource should not be deleted by provider, skip deleting.")
		return nil
	}

	cr.SetConditions(rtv1.Deleting())

	return nil
}
