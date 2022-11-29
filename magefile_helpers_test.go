package main

import (
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

func TestReplaceImageAndEnvVar(t *testing.T) {
	d := &appsv1.Deployment{}
	d.Spec.Template.Spec.Containers = []corev1.Container{
		{Name: "manager"},
	}

}
