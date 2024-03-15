package v1alpha1

import (
	"reflect"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/scheme"
)

const (
	Group   = "krateo.io"
	Version = "v1alpha1"
)

var (
	SchemeGroupVersion = schema.GroupVersion{Group: Group, Version: Version}
	SchemeBuilder      = &scheme.Builder{GroupVersion: SchemeGroupVersion}
)

var (
	KrateoPlatformOpsKind             = reflect.TypeOf(KrateoPlatformOps{}).Name()
	KrateoPlatformOpsGroupKind        = schema.GroupKind{Group: Group, Kind: KrateoPlatformOpsKind}.String()
	KrateoPlatformOpsKindAPIVersion   = KrateoPlatformOpsKind + "." + SchemeGroupVersion.String()
	KrateoPlatformOpsGroupVersionKind = SchemeGroupVersion.WithKind(KrateoPlatformOpsKind)
)

func init() {
	SchemeBuilder.Register(&KrateoPlatformOps{}, &KrateoPlatformOpsList{})
}
