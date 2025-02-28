package v1alpha1

import (
	"strconv"

	"helm.sh/helm/v3/pkg/release"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"

	rtv1 "github.com/krateoplatformops/provider-runtime/apis/common/v1"
	"github.com/twmb/murmur3"
)

type Data struct {
	Name     string `json:"name"`
	Value    string `json:"value,omitempty"`
	AsString *bool  `json:"asString,omitempty"`
}

type ObjectMeta struct {
	APIVersion string         `json:"apiVersion"`
	Kind       string         `json:"kind"`
	Metadata   rtv1.Reference `json:"metadata"`
}

type ValueFromSource struct {
	ObjectMeta `json:",inline"`
	Selector   string `json:"selector,omitempty"`
}

type Var struct {
	Data      `json:",inline"`
	ValueFrom *ValueFromSource `json:"valueFrom,omitempty"`
}

type Credentials struct {
	Username    string                 `json:"username"`
	PasswordRef rtv1.SecretKeySelector `json:"passwordRef"`
}

type ChartSpec struct {
	// Repository: Helm repository URL, required if ChartSpec.URL not set
	Repository string `json:"repository,omitempty"`
	// Name of Helm chart, required if ChartSpec.URL not set
	Name string `json:"name,omitempty"`
	// Version of Helm chart, late initialized with latest version if not set
	Version string `json:"version,omitempty"`
	// URL to chart package (typically .tgz), optional and overrides others fields in the spec
	URL string `json:"url,omitempty"`
	// // PullSecretRef is reference to the secret containing credentials to helm repository
	// PullSecretRef prv1.SecretKeySelector `json:"pullSecretRef,omitempty"`

	// ReleaseName is the name of the release. If not set, Name will be used or it will be deriverd from the URL
	// +optional
	ReleaseName string `json:"releaseName,omitempty"`

	// MaxHistory is the maximum number of helm releases to keep in history
	MaxHistory *int `json:"maxHistory,omitempty"`

	// Namespace to install the release into.
	//Namespace string `json:"namespace"`
	// SkipCreateNamespace won't create the namespace for the release. This requires the namespace to already exist.
	SkipCreateNamespace bool `json:"skipCreateNamespace,omitempty"`
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

	// Credentials: credentials for private repos
	// +optional
	Credentials *Credentials `json:"credentials,omitempty"`
}

type ChartObservation struct {
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
	// +kubebuilder:validation:Required
	ID string `json:"id"`
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=object;chart;var
	Type StepType `json:"type"`
	// +kubebuilder:pruning:PreserveUnknownFields
	With *runtime.RawExtension `json:"with"`
}

func (s *Step) Digest() string {
	if s.With == nil || len(s.With.Raw) == 0 {
		return ""
	}

	hasher := murmur3.New64()
	hasher.Write(s.With.Raw)

	return strconv.FormatUint(hasher.Sum64(), 16)
}

type WorkflowSpec struct {
	Steps []*Step `json:"steps,omitempty"`
}

type WorkflowStatus struct {
	rtv1.ConditionedStatus `json:",inline"`
	Digest                 *string `json:"digest,omitempty"`
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

func (mg *KrateoPlatformOps) GetCondition(ct rtv1.ConditionType) rtv1.Condition {
	return mg.Status.GetCondition(ct)
}

func (mg *KrateoPlatformOps) SetConditions(c ...rtv1.Condition) {
	mg.Status.SetConditions(c...)
}
