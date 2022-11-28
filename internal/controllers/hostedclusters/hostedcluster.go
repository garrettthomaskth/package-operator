package hostedclusters

// Take only what we need from https://github.com/openshift/hypershift/blob/main/api/v1alpha1/hostedcluster_types.go
import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/scheme"
)

var (
	// GroupVersion is group version used to register these objects.
	GroupVersion = schema.GroupVersion{Group: "package-operator.run", Version: "v1alpha1"}

	// SchemeBuilder is used to add go types to the GroupVersionKind scheme.
	SchemeBuilder = &scheme.Builder{GroupVersion: GroupVersion}

	// AddToScheme adds the types in this group-version to the given scheme.
	AddToScheme = SchemeBuilder.AddToScheme
)

func init() {
	SchemeBuilder.Register(&HostedCluster{}, &HostedClusterList{})
}

// HostedClusterStatus is the latest observed status of a HostedCluster.
type HostedClusterStatus struct {

	// KubeConfig is a reference to the secret containing the default kubeconfig
	// for the cluster.
	// +optional
	KubeConfig *corev1.LocalObjectReference `json:"kubeconfig,omitempty"`

	// KubeadminPassword is a reference to the secret that contains the initial
	// kubeadmin user password for the guest cluster.
	// +optional
	KubeadminPassword *corev1.LocalObjectReference `json:"kubeadminPassword,omitempty"`

	// Conditions represents the latest available observations of a control
	// plane's current state.
	Conditions []metav1.Condition `json:"conditions"`
}

// +genclient

// HostedCluster is the primary representation of a HyperShift cluster and encapsulates
// the control plane and common data plane configuration. Creating a HostedCluster
// results in a fully functional OpenShift control plane with no attached nodes.
// To support workloads (e.g. pods), a HostedCluster may have one or more associated
// NodePool resources.
//
// +kubebuilder:object:root=true
// +kubebuilder:resource:path=hostedclusters,shortName=hc;hcs,scope=Namespaced
// +kubebuilder:storageversion
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="KubeConfig",type="string",JSONPath=".status.kubeconfig.name",description="KubeConfig Secret"
// +kubebuilder:printcolumn:name="Available",type="string",JSONPath=".status.conditions[?(@.type==\"Available\")].status",description="Available"
// +kubebuilder:printcolumn:name="Progressing",type="string",JSONPath=".status.conditions[?(@.type==\"Progressing\")].status",description="Progressing"
// +kubebuilder:printcolumn:name="Message",type="string",JSONPath=".status.conditions[?(@.type==\"Available\")].message",description="Message"
type HostedCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Status is the latest observed status of the HostedCluster.
	Status HostedClusterStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
// HostedClusterList contains a list of HostedCluster
type HostedClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []HostedCluster `json:"items"`
}
