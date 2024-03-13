package v1alpha1

import (
	"helm.sh/helm/v3/pkg/release"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	prv1 "github.com/krateoplatformops/provider-runtime/apis/common/v1"
)

// DataKeySelector defines required spec to access a key of a configmap or secret
type DataKeySelector struct {
	prv1.Reference `json:",inline,omitempty"`
	Key            string `json:"key,omitempty"`
	Optional       bool   `json:"optional,omitempty"`
}

// ValueFromSource represents source of a value
type ValueFromSource struct {
	ConfigMapKeyRef *DataKeySelector `json:"configMapKeyRef,omitempty"`
	SecretKeyRef    *DataKeySelector `json:"secretKeyRef,omitempty"`
}

// SetVal represents a "set" value override in a Release
type SetVal struct {
	Name      string           `json:"name"`
	Value     string           `json:"value,omitempty"`
	ValueFrom *ValueFromSource `json:"valueFrom,omitempty"`
}

type ReleaseParameters struct {
	// Repository: Helm repository URL, required if ChartSpec.URL not set
	Repository string `json:"repository,omitempty"`
	// Name of Helm chart, required if ChartSpec.URL not set
	Name string `json:"name,omitempty"`
	// Version of Helm chart, late initialized with latest version if not set
	Version string `json:"version,omitempty"`
	// URL to chart package (typically .tgz), optional and overrides others fields in the spec
	URL string `json:"url,omitempty"`
	// PullSecretRef is reference to the secret containing credentials to helm repository
	PullSecretRef prv1.SecretKeySelector `json:"pullSecretRef,omitempty"`

	// Namespace to install the release into.
	//Namespace string `json:"namespace"`
	// SkipCreateNamespace won't create the namespace for the release. This requires the namespace to already exist.
	//SkipCreateNamespace bool `json:"skipCreateNamespace,omitempty"`
	// Wait for the release to become ready.
	Wait *bool `json:"wait,omitempty"`
	// WaitTimeout is the duration Helm will wait for the release to become
	// ready. Only applies if wait is also set. Defaults to 5m.
	WaitTimeout *metav1.Duration `json:"waitTimeout,omitempty"`
	// Set defines the Helm values
	Set []SetVal `json:"set,omitempty"`
	// SkipCRDs skips installation of CRDs for the release.
	//SkipCRDs bool `json:"skipCRDs,omitempty"`
	// InsecureSkipTLSVerify skips tls certificate checks for the chart download
	InsecureSkipTLSVerify *bool `json:"insecureSkipTLSVerify,omitempty"`
}

type ReleaseObservation struct {
	State              release.Status `json:"state,omitempty"`
	ReleaseDescription string         `json:"releaseDescription,omitempty"`
	Revision           int            `json:"revision,omitempty"`
}

type KrateoPlatformOpsSpec struct {
	prv1.ManagedSpec `json:",inline"`
	ServiceType      *string              `json:"serviceType,omitempty"`
	Releases         []*ReleaseParameters `json:"install,omitempty"`
}

type KrateoPlatformOpsStatus struct {
	prv1.ManagedStatus `json:",inline"`
	Releases           []*ReleaseObservation `json:"releases,omitempty"`
}

// +kubebuilder:object:root=true

// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:resource:scope=Namespaced,categories={krateo}
type KrateoPlatformOps struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KrateoPlatformOpsSpec   `json:"spec"`
	Status KrateoPlatformOpsStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// KrateoPlatformOpsList contains a list of KrateoPlatformOps
type KrateoPlatformOpsList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []KrateoPlatformOps `json:"items"`
}
