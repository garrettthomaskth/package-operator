package hostedclusters

import (
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	corev1alpha1 "package-operator.run/apis/core/v1alpha1"
	"package-operator.run/package-operator/internal/utils"
)

// TODO: I Think i can remove this whole thing

// TODO: remove the functions we don't need
type genericPackage interface {
	ClientObject() client.Object
	UpdatePhase()
	GetConditions() *[]metav1.Condition
	GetImage() string
	GetSpecHash() string
	setStatusPhase(phase corev1alpha1.PackageStatusPhase)
}

type genericPackageFactory func(scheme *runtime.Scheme) genericPackage

var (
	packageGVK = corev1alpha1.GroupVersion.WithKind("Package")
)

func newGenericPackage(scheme *runtime.Scheme) genericPackage {
	obj, err := scheme.New(packageGVK)
	if err != nil {
		panic(err)
	}

	return &GenericPackage{
		Package: *obj.(*corev1alpha1.Package)}
}

var (
	_ genericPackage = (*GenericPackage)(nil)
)

type GenericPackage struct {
	corev1alpha1.Package
}

func (a *GenericPackage) ClientObject() client.Object {
	return &a.Package
}

func (a *GenericPackage) GetConditions() *[]metav1.Condition {
	return &a.Status.Conditions
}

func (a *GenericPackage) UpdatePhase() {
	updatePackagePhase(a)
}

func (a *GenericPackage) GetImage() string {
	return a.Spec.Image
}

func (a *GenericPackage) GetSpecHash() string {
	return utils.ComputeHash(a.Spec, nil)
}

func (a *GenericPackage) setStatusPhase(phase corev1alpha1.PackageStatusPhase) {
	a.Status.Phase = phase
}

func updatePackagePhase(pkg genericPackage) {
	if meta.IsStatusConditionTrue(*pkg.GetConditions(), corev1alpha1.PackageInvalid) {
		pkg.setStatusPhase(corev1alpha1.PackagePhaseInvalid)
		return
	}

	unpackCond := meta.FindStatusCondition(*pkg.GetConditions(), corev1alpha1.PackageUnpacked)
	if unpackCond == nil {
		pkg.setStatusPhase(corev1alpha1.PackagePhaseUnpacking)
		return
	}

	if meta.IsStatusConditionTrue(
		*pkg.GetConditions(),
		corev1alpha1.PackageProgressing,
	) {
		pkg.setStatusPhase(corev1alpha1.PackagePhaseProgressing)
		return
	}

	if meta.IsStatusConditionTrue(
		*pkg.GetConditions(),
		corev1alpha1.PackageAvailable,
	) {
		pkg.setStatusPhase(corev1alpha1.PackagePhaseAvailable)
		return
	}

	pkg.setStatusPhase(corev1alpha1.PackagePhaseNotReady)
}
