package hostedclusters

import (
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func newHostedCluster() *HostedCluster {
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

type HostedCluster struct {
	HostedCluster unstructured.Unstructured
}

func (a *HostedCluster) ClientObject() client.Object {
	return &a.HostedCluster
}

func (a *HostedCluster) GetConditions() (*[]metav1.Condition, error) {
	interfaceConds, ok, err := unstructured.NestedSlice(a.HostedCluster.Object, "status", "conditions")

	if err != nil {
		return nil, fmt.Errorf("getting conditions: %w", err)
	}
	if !ok {
		return nil, fmt.Errorf("couldn't find conditions nested slice")
	}
	conds := make([]metav1.Condition, len(interfaceConds))
	for i, d := range interfaceConds {
		conds[i] = d.(metav1.Condition)
	}
	return &conds, nil
}
