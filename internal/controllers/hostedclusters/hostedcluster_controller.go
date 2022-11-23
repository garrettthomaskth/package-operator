package hostedclusters

import (
	"context"
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	corev1alpha1 "package-operator.run/apis/core/v1alpha1"
)

type HostedClusterController struct {
	client client.Client
	log    logr.Logger
	scheme *runtime.Scheme
	image  string
}

func NewHostedClusterController(
	c client.Client, log logr.Logger, scheme *runtime.Scheme, image string,
) *HostedClusterController {
	controller := &HostedClusterController{
		client: c,
		log:    log,
		scheme: scheme,
		image:  image,
	}
	return controller
}

func (c *HostedClusterController) Reconcile(
	ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := c.log.WithValues("HostedCluster", req.String())
	defer log.Info("reconciled")
	ctx = logr.NewContext(ctx, log)
	hostedCluster := newHostedCluster()
	if err := c.client.Get(ctx, req.NamespacedName, hostedCluster.ClientObject()); err != nil {
		// Ignore not found errors on delete
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	conds, err := hostedCluster.GetConditions()
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("getting hostedcluster conditions: %w", err)
	}
	ok := isHostedClusterReady(conds)
	if !ok {
		return ctrl.Result{}, nil
	}

	desiredPackage := c.desiredPackage(hostedCluster)
	err = controllerutil.SetControllerReference(hostedCluster.ClientObject(), desiredPackage, c.scheme)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("setting controller reference: %w", err)
	}

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

func isHostedClusterReady(conds *[]metav1.Condition) bool {
	ready := false
	for _, cond := range *conds {
		// TODO: is this the condition we want to check?
		if cond.Type == "Available" {
			if cond.Status == "True" {
				ready = true
			}
			break
		}
	}
	return ready
}

func (c *HostedClusterController) desiredPackage(cluster *HostedCluster) *corev1alpha1.Package {
	pkg := &corev1alpha1.Package{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cluster.ClientObject().GetName() + "_remote_phase_manager",
			Namespace: cluster.ClientObject().GetNamespace(),
		},
		Spec: corev1alpha1.PackageSpec{
			Image: c.image,
		},
	}
	return pkg
}

func (c *HostedClusterController) SetupWithManager(mgr ctrl.Manager) error {
	hostedCluster := newHostedCluster().ClientObject()

	return ctrl.NewControllerManagedBy(mgr).
		For(hostedCluster).
		Owns(&corev1alpha1.Package{}).
		Complete(c)
}
