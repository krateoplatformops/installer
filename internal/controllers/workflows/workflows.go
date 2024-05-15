package workflows

import (
	"context"
	"time"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	workflowsv1alpha1 "github.com/krateoplatformops/installer/apis/workflows/v1alpha1"
	"github.com/krateoplatformops/installer/internal/ptr"
	"github.com/krateoplatformops/installer/internal/workflows"
	"github.com/krateoplatformops/installer/internal/workflows/steps"
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

	creationGracePeriod = 2 * time.Minute
	reconcileTimeout    = 10 * time.Minute
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
		reconciler.WithCreationGracePeriod(creationGracePeriod),
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

	wf, err := workflows.New(c.rc,
		cr.GetNamespace(),
		c.log.WithValues("workflow", cr.GetName()),
	)
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

	got := ptr.Deref(cr.Status.Digest, "")
	if len(got) == 0 {
		return reconciler.ExternalObservation{
			ResourceExists:   false,
			ResourceUpToDate: true,
		}, nil
	}

	exp := digestForSteps(cr)

	upToDate := (exp == got)
	if upToDate {
		cr.SetConditions(rtv1.Available())
		//err := e.kube.Status().Update(ctx, cr)

		return reconciler.ExternalObservation{
			ResourceExists:   true,
			ResourceUpToDate: true,
		}, nil
	}

	return reconciler.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: false,
	}, nil
}

func (e *external) Create(ctx context.Context, mg resource.Managed) error {
	cr, ok := mg.(*workflowsv1alpha1.KrateoPlatformOps)
	if !ok {
		return errors.New(errNotCR)
	}

	if meta.WasDeleted(cr) {
		return nil
	}

	if !meta.IsActionAllowed(cr, meta.ActionCreate) {
		e.log.Debug("External resource should not be updated by provider, skip creating.")
		return nil
	}

	cr.SetConditions(rtv1.Creating())

	e.wf.Op(steps.Create)
	results := e.wf.Run(ctx, cr.Spec.DeepCopy(), func(s *workflowsv1alpha1.Step) bool {
		return false
	})
	if err := workflows.Err(results); err != nil {
		e.log.Debug("Worflow failure", "error", err.Error())
		return err
	}

	cr.Status.SetConditions(rtv1.Available())
	cr.Status.Digest = ptr.To(digestForSteps(cr))

	return e.kube.Status().Update(ctx, cr)
}

func (e *external) Update(ctx context.Context, mg resource.Managed) error {
	cr, ok := mg.(*workflowsv1alpha1.KrateoPlatformOps)
	if !ok {
		return errors.New(errNotCR)
	}

	if meta.WasDeleted(cr) {
		return nil
	}

	if !meta.IsActionAllowed(cr, meta.ActionUpdate) {
		e.log.Debug("External resource should not be updated by provider, skip updating.")
		return nil
	}

	e.wf.Op(steps.Update)
	results := e.wf.Run(ctx, cr.Spec.DeepCopy(), func(s *workflowsv1alpha1.Step) bool {
		return false
	})
	if err := workflows.Err(results); err != nil {
		e.log.Debug("Worflow failure", "error", err.Error())
		return err
	}

	cr.Status.SetConditions(rtv1.Available())
	cr.Status.Digest = ptr.To(digestForSteps(cr))

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

	if meta.WasDeleted(cr) {
		return nil
	}

	e.wf.Op(steps.Delete)
	results := e.wf.Run(ctx, cr.Spec.DeepCopy(), func(s *workflowsv1alpha1.Step) bool {
		return s.Type == workflowsv1alpha1.TypeVar
	})

	err := workflows.Err(results)
	if err != nil {
		e.log.Debug("Worflow failure", "op", "delete", "error", err.Error())
	}

	return err
}
