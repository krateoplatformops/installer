package workflows

import (
	"strconv"

	"github.com/krateoplatformops/installer/apis/workflows/v1alpha1"
	"github.com/twmb/murmur3"
)

func digestForSteps(cr *v1alpha1.KrateoPlatformOps) string {
	hasher := murmur3.New64()

	for _, x := range cr.Spec.Steps {
		hasher.Write([]byte(x.Digest()))
	}

	return strconv.FormatUint(hasher.Sum64(), 16)
}
