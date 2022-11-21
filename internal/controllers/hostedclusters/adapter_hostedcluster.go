package hostedclusters

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type hostedCluster interface {
	ClientObject() client.Object
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
