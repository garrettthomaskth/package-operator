package hostedclusters

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/utils/pointer"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	corev1alpha1 "package-operator.run/apis/core/v1alpha1"
	"package-operator.run/package-operator/internal/controllers"
)

type HostedClusterController struct {
	client           client.Client
	log              logr.Logger
	scheme           *runtime.Scheme
	newHostedCluster hostedClusterFactory
	image            string
}

func NewHostedClusterController(
	c client.Client, log logr.Logger, scheme *runtime.Scheme, image string,
) *HostedClusterController {
	controller := &HostedClusterController{
		client:           c,
		log:              log,
		scheme:           scheme,
		newHostedCluster: newHostedCluster,
		image:            image,
	}
	return controller
}

func (c *HostedClusterController) Reconcile(
	ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := c.log.WithValues("HostedCluster", req.String())
	defer log.Info("reconciled")
	ctx = logr.NewContext(ctx, log)
	hostedCluster := c.newHostedCluster()
	if err := c.client.Get(ctx, req.NamespacedName, hostedCluster.ClientObject()); err != nil {
		// Ignore not found errors on delete
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	desiredPackage := c.desiredPackage(hostedCluster)
	// Can't use controllerutil.SetControllerReference because the HostedClusterType isn't in Scheme,
	setControllerReference(hostedCluster.ClientObject(), desiredPackage)

	existingPkg := &corev1alpha1.Package{}
	if err := c.client.Get(ctx, client.ObjectKeyFromObject(desiredPackage), existingPkg); err != nil && errors.IsNotFound(err) {
		if err := c.client.Create(ctx, desiredPackage); err != nil {
			return ctrl.Result{}, fmt.Errorf("creating Package: %w", err)
		}
	} else if err != nil {
		return ctrl.Result{}, fmt.Errorf("getting Package: %w", err)
	}
	return ctrl.Result{}, nil
}

func (c *HostedClusterController) desiredPackage(cluster hostedCluster) *corev1alpha1.Package {
	pkg := &corev1alpha1.Package{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cluster.ClientObject().GetName() + "_remote_phase_manager",
			Namespace: cluster.ClientObject().GetNamespace(),
			Labels: map[string]string{
				controllers.DynamicCacheLabel: "True",
			},
		},
		Spec: corev1alpha1.PackageSpec{
			Image: c.image,
		},
	}
	return pkg
}

func setControllerReference(owner, controlled metav1.Object) {
	// Create a new controller ref.
	gvk := schema.GroupVersionKind{
		Group:   "hypershift.openshift.io",
		Kind:    "HostedCluster",
		Version: "v1alpha1",
	}
	ref := metav1.OwnerReference{
		APIVersion:         gvk.GroupVersion().String(),
		Kind:               gvk.Kind,
		Name:               owner.GetName(),
		UID:                owner.GetUID(),
		BlockOwnerDeletion: pointer.Bool(true),
		Controller:         pointer.Bool(true),
	}

	// Update owner references and return.
	ownerRefs := controlled.GetOwnerReferences()
	ownerRefs = append(ownerRefs, ref)
	controlled.SetOwnerReferences(ownerRefs)
}

func (c *HostedClusterController) SetupWithManager(mgr ctrl.Manager) error {
	hostedCluster := c.newHostedCluster().ClientObject()
	// packageObj, err := c.scheme.New(corev1alpha1.GroupVersion.WithKind("Package"))
	// Owns(packageObj.(*corev1alpha1.Package)).

	return ctrl.NewControllerManagedBy(mgr).
		For(hostedCluster).
		Owns(&corev1alpha1.Package{}).
		Complete(c)
}
