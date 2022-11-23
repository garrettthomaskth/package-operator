package hostedclusters

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type hostedCluster interface {
	ClientObject() client.Object
	GetConditions() *[]metav1.Condition
	// GetStatusKubeconfig() string
}

type hostedClusterFactory func() hostedCluster

func newHostedCluster() hostedCluster {
	obj := unstructured.Unstructured{}
	obj.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "hypershift.openshift.io",
		Kind:    "HostedCluster",
		Version: "v1alpha1",
	})

	return &HostedCluster{
		HostedCluster: obj,
	}
}

var (
	_ hostedCluster = (*HostedCluster)(nil)
)

type HostedCluster struct {
	HostedCluster unstructured.Unstructured
}

func (a *HostedCluster) ClientObject() client.Object {
	return &a.HostedCluster
}

func (a *HostedCluster) GetConditions() *[]metav1.Condition {
	interfaceConds, ok, err := unstructured.NestedSlice(a.HostedCluster.Object, "status", "conditions")

	if ok == false || err != nil {
		// TODO: Should we do something here?
	}
	conds := make([]metav1.Condition, len(interfaceConds))
	for i, d := range interfaceConds {
		conds[i] = d.(metav1.Condition)
	}
	return &conds
}

//func (a *HostedCluster) GetStatusKubeconfig() string {
//	// TODO: Is it a problem that Kubeconfig is a pointer?
//	kubeconfig, ok, err := unstructured.NestedString(a.HostedCluster.Object, "status", "kubeconfig", "name")
//	if ok == false || err != nil {
//		// TODO: Should we do something here?
//	}
//	return kubeconfig
//}
