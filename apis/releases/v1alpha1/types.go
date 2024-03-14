package v1alpha1

import (
	"helm.sh/helm/v3/pkg/release"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"

	prv1 "github.com/krateoplatformops/provider-runtime/apis/common/v1"
)

type Data struct {
	Name  string `json:"name"`
	Value string `json:"value,omitempty"`
}

type ObjectMeta struct {
	APIVersion string         `json:"apiVersion"`
	Kind       string         `json:"kind"`
	Metadata   prv1.Reference `json:"metadata"`
}

type ValueFromSource struct {
	ObjectMeta `json:",inline"`
	Selector   string `json:"selector,omitempty"`
}

type Var struct {
	Data      `json:",inline"`
	ValueFrom *ValueFromSource `json:"valueFrom,omitempty"`
}

type HelmChartSpec struct {
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
	Set []*Data `json:"set,omitempty"`
	// SkipCRDs skips installation of CRDs for the release.
	//SkipCRDs bool `json:"skipCRDs,omitempty"`
	// InsecureSkipTLSVerify skips tls certificate checks for the chart download
	InsecureSkipTLSVerify *bool `json:"insecureSkipTLSVerify,omitempty"`
}

type HelmChartObservation struct {
	State    release.Status `json:"state,omitempty"`
	Revision int            `json:"revision,omitempty"`
}

type Object struct {
	ObjectMeta `json:",inline"`
	Set        []*Data `json:"set,omitempty"`
}

// +kubebuilder:validation:Enum=object;chart;var
type StepType string

const (
	TypeObject StepType = "object"
	TypeChart  StepType = "chart"
	TypeVar    StepType = "var"
)

type Step struct {
	ID   *string  `json:"id,omitempty"`
	Type StepType `json:"type"`
	// +kubebuilder:pruning:PreserveUnknownFields
	With *runtime.RawExtension `json:"with"`
}

type StepStatus struct {
	ID  *string `json:"id,omitempty"`
	Err *string `json:"err,omitempty"`
}

type WorkflowSpec struct {
	prv1.ManagedSpec `json:",inline"`
	Steps            []*Step `json:"steps,omitempty"`
}

type WorkflowStatus struct {
	prv1.ManagedStatus `json:",inline"`
	Steps              []*StepStatus `json:"steps,omitempty"`
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

	Spec   WorkflowSpec   `json:"spec"`
	Status WorkflowStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// KrateoPlatformOpsList contains a list of KrateoPlatformOps
type KrateoPlatformOpsList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []KrateoPlatformOps `json:"items"`
}
