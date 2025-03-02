package objecttemplate

import (
	"context"
	"fmt"

	manifestsv1alpha1 "package-operator.run/apis/manifests/v1alpha1"
	"package-operator.run/package-operator/internal/environment"

	"k8s.io/apimachinery/pkg/runtime/schema"

	"k8s.io/apimachinery/pkg/api/meta"

	"package-operator.run/package-operator/internal/preflight"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"package-operator.run/package-operator/internal/controllers"
	"package-operator.run/package-operator/internal/dynamiccache"
)

type dynamicCache interface {
	client.Reader
	Source() source.Source
	Free(ctx context.Context, obj client.Object) error
	Watch(ctx context.Context, owner client.Object, obj runtime.Object) error
	OwnersForGKV(gvk schema.GroupVersionKind) []dynamiccache.OwnerReference
}

type reconciler interface {
	Reconcile(ctx context.Context, pkg genericObjectTemplate) (ctrl.Result, error)
}

type preflightChecker interface {
	Check(
		ctx context.Context, owner, obj client.Object,
	) (violations []preflight.Violation, err error)
}

var _ environment.Sinker = (*GenericObjectTemplateController)(nil)

type GenericObjectTemplateController struct {
	newObjectTemplate  genericObjectTemplateFactory
	log                logr.Logger
	scheme             *runtime.Scheme
	client             client.Client
	uncachedClient     client.Client
	dynamicCache       dynamicCache
	templateReconciler *templateReconciler
	reconciler         []reconciler
}

func NewObjectTemplateController(
	client, uncachedClient client.Client,
	log logr.Logger,
	dynamicCache dynamicCache,
	scheme *runtime.Scheme,
	restMapper meta.RESTMapper,
) *GenericObjectTemplateController {
	return newGenericObjectTemplateController(
		client, uncachedClient, log, dynamicCache, scheme,
		restMapper, newGenericObjectTemplate)
}

func NewClusterObjectTemplateController(
	client, uncachedClient client.Client,
	log logr.Logger,
	dynamicCache dynamicCache,
	scheme *runtime.Scheme,
	restMapper meta.RESTMapper,
) *GenericObjectTemplateController {
	return newGenericObjectTemplateController(
		client, uncachedClient, log, dynamicCache, scheme,
		restMapper, newGenericClusterObjectTemplate)
}

func newGenericObjectTemplateController(
	client, uncachedClient client.Client,
	log logr.Logger,
	dynamicCache dynamicCache,
	scheme *runtime.Scheme,
	restMapper meta.RESTMapper,
	newObjectTemplate genericObjectTemplateFactory,
) *GenericObjectTemplateController {
	controller := &GenericObjectTemplateController{
		newObjectTemplate: newObjectTemplate,
		log:               log,
		scheme:            scheme,
		client:            client,
		uncachedClient:    uncachedClient,
		dynamicCache:      dynamicCache,
		templateReconciler: newTemplateReconciler(scheme, client, uncachedClient, dynamicCache, preflight.List{
			preflight.NewAPIExistence(restMapper),
			preflight.NewEmptyNamespaceNoDefault(restMapper),
			preflight.NewNamespaceEscalation(restMapper),
		}),
	}
	controller.reconciler = []reconciler{controller.templateReconciler}
	return controller
}

func (c *GenericObjectTemplateController) Reconcile(
	ctx context.Context, req ctrl.Request,
) (ctrl.Result, error) {
	log := c.log.WithValues("ObjectTemplate", req.String())
	defer log.Info("reconciled")
	ctx = logr.NewContext(ctx, log)

	objectTemplate := c.newObjectTemplate(c.scheme)
	if err := c.client.Get(
		ctx, req.NamespacedName, objectTemplate.ClientObject()); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if !objectTemplate.ClientObject().GetDeletionTimestamp().IsZero() {
		if err := controllers.FreeCacheAndRemoveFinalizer(
			ctx, c.client, objectTemplate.ClientObject(), c.dynamicCache); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	if err := controllers.EnsureCachedFinalizer(ctx, c.client, objectTemplate.ClientObject()); err != nil {
		return ctrl.Result{}, err
	}

	var (
		res ctrl.Result
		err error
	)
	for _, r := range c.reconciler {
		res, err = r.Reconcile(ctx, objectTemplate)
		if err != nil || !res.IsZero() {
			break
		}
	}
	if err != nil {
		return res, err
	}
	return res, c.updateStatus(ctx, objectTemplate)
}

func (c *GenericObjectTemplateController) updateStatus(ctx context.Context, objectTemplate genericObjectTemplate) error {
	objectTemplate.UpdatePhase()
	if err := c.client.Status().Update(ctx, objectTemplate.ClientObject()); err != nil {
		return fmt.Errorf("updating ObjectTemplate status: %w", err)
	}
	return nil
}

func (c *GenericObjectTemplateController) SetEnvironment(env *manifestsv1alpha1.PackageEnvironment) {
	c.templateReconciler.SetEnvironment(env)
}

func (c *GenericObjectTemplateController) SetupWithManager(
	mgr ctrl.Manager,
) error {
	objectTemplate := c.newObjectTemplate(c.scheme).ClientObject()

	return ctrl.NewControllerManagedBy(mgr).
		For(objectTemplate).
		Watches(c.dynamicCache.Source(), &dynamiccache.EnqueueWatchingObjects{
			WatcherRefGetter: c.dynamicCache,
			WatcherType:      objectTemplate,
		}).Complete(c)
}
