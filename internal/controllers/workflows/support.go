package workflows

import (
	"github.com/krateoplatformops/installer/apis/workflows/v1alpha1"
	"github.com/krateoplatformops/installer/internal/ptr"
)

func currentDigestMap(cr *v1alpha1.KrateoPlatformOps) map[string]string {
	got := map[string]string{}

	for _, x := range cr.Status.Steps {
		id := ptr.Deref(x.ID, "")
		hash := ptr.Deref(x.Digest, "")
		if len(id) == 0 || len(hash) == 0 {
			continue
		}
		got[id] = hash
	}

	return got
}

func listOfStepIdToUpdate(cr *v1alpha1.KrateoPlatformOps) []string {
	digestMap := currentDigestMap(cr)

	all := []string{}
	for _, x := range cr.Spec.Steps {
		obs := x.Digest()
		old, ok := digestMap[x.ID]
		if ok && (old == obs) {
			continue
		}

		all = append(all, x.ID)
	}

	return all
}
