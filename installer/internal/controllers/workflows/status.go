package workflows

import (
	workflowsv1alpha1 "github.com/krateoplatformops/installer/apis/workflows/v1alpha1"
	"github.com/krateoplatformops/installer/internal/workflows"
	"github.com/krateoplatformops/installer/internal/workflows/steps"
	rtv1 "github.com/krateoplatformops/provider-runtime/apis/common/v1"
)

// StatusPopulator definisce come un risultato può popolare lo status
type StatusPopulator interface {
	PopulateStatus(cr *workflowsv1alpha1.KrateoPlatformOps)
}

// Wrapper per VarResult
type VarStatusWrapper struct {
	*steps.VarResult
}

func (w VarStatusWrapper) PopulateStatus(cr *workflowsv1alpha1.KrateoPlatformOps) {
	if cr.Status.VarList == nil {
		cr.Status.VarList = make([]workflowsv1alpha1.Var, 0)
	}
	cr.Status.VarList = append(cr.Status.VarList, workflowsv1alpha1.Var{
		Data: workflowsv1alpha1.Data{
			Name:  w.Name,
			Value: w.Value,
		},
	})
}

// Wrapper per ObjectResult
type ObjectStatusWrapper struct {
	*steps.ObjectResult
}

func (w ObjectStatusWrapper) PopulateStatus(cr *workflowsv1alpha1.KrateoPlatformOps) {
	if cr.Status.ObjectList == nil {
		cr.Status.ObjectList = make([]workflowsv1alpha1.Object, 0)
	}
	cr.Status.ObjectList = append(cr.Status.ObjectList, workflowsv1alpha1.Object{
		ObjectMeta: workflowsv1alpha1.ObjectMeta{
			APIVersion: w.APIVersion,
			Kind:       w.Kind,
			Metadata: rtv1.Reference{
				Name:      w.Name,
				Namespace: w.Namespace,
			},
		},
	})
}

// Wrapper per ChartResult
type ChartStatusWrapper struct {
	*steps.ChartResult
}

func (w ChartStatusWrapper) PopulateStatus(cr *workflowsv1alpha1.KrateoPlatformOps) {
	if cr.Status.ReleaseList == nil {
		cr.Status.ReleaseList = make([]workflowsv1alpha1.Release, 0)
	}
	cr.Status.ReleaseList = append(cr.Status.ReleaseList, workflowsv1alpha1.Release{
		ReleaseName:  w.ReleaseName,
		ChartName:    w.ChartName,
		ChartVersion: w.ChartVersion,
		AppVersion:   w.AppVersion,
		Namespace:    w.Namespace,
		Status:       w.Status,
		Revision:     w.Revision,
		Updated:      w.Updated,
	})
}

// Factory function per creare il wrapper appropriato
func wrapResultForStatus(result interface{}) StatusPopulator {
	switch v := result.(type) {
	case *steps.VarResult:
		return VarStatusWrapper{v}
	case *steps.ObjectResult:
		return ObjectStatusWrapper{v}
	case *steps.ChartResult:
		return ChartStatusWrapper{v}
	default:
		return nil
	}
}

// populateStatus popola lo status del CR basandosi sui risultati del workflow
func populateStatus(cr *workflowsv1alpha1.KrateoPlatformOps, results []workflows.StepResult[any]) {
	// Reset delle liste
	cr.Status.ObjectList = make([]workflowsv1alpha1.Object, 0)
	cr.Status.ReleaseList = make([]workflowsv1alpha1.Release, 0)
	cr.Status.VarList = make([]workflowsv1alpha1.Var, 0)

	for _, result := range results {
		if result.Err() != nil {
			continue // Skip risultati con errori
		}

		// Usa il wrapper per popolare lo status
		resultValue := result.Result()
		if resultValue == nil {
			continue
		}

		if wrapper := wrapResultForStatus(resultValue); wrapper != nil {
			wrapper.PopulateStatus(cr)
		}
	}
}
