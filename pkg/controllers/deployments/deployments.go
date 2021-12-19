package deployments

import (
	"context"
	"os"
	"strings"
	"time"

	"github.com/lucasepe/image-cloner/pkg/cloner"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

// kubectl get deployment nginx -n nginx -o=jsonpath='{$.spec.template.spec.containers[:1].image}'

const (
	// KubeNs Namespace to exclude in Reconiler
	KubeNs = "kube-system"

	LocalNs = "local-path-storage"
)

// SetupWithManager sets up the controller with the Manager.
func SetupWithManager(mgr ctrl.Manager) error {
	// Create the reconciler
	rec := newReconciler(mgr)

	// Create a new controller
	con, err := controller.New("image-cloner", mgr, controller.Options{
		Reconciler: rec,
		Log:        ctrllog.Log,
	})
	if err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&appsv1.Deployment{}).
		Owns(&corev1.Pod{}).
		Watches(&source.Kind{Type: &appsv1.Deployment{}}, &handler.EnqueueRequestForObject{}).
		Complete(con)
}

var _ reconcile.Reconciler = &ReconcileDeployment{}

// ReconcileDeployment reconciles a Deployment object
type ReconcileDeployment struct {
	client.Client
	namespacesToSkip []string
	scheme           *runtime.Scheme
}

// Reconcile reconciles Deployment to update the image
func (r *ReconcileDeployment) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	if r.isNamespaceToBeExcluded(req.NamespacedName.Namespace) {
		return ctrl.Result{}, nil
	}

	log := ctrllog.FromContext(ctx) // .WithValues("deployment", req.NamespacedName)

	// Fetch the Deployment instance
	dep := &appsv1.Deployment{}
	err := r.Get(context.TODO(), req.NamespacedName, dep)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request - return and don't requeue:
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request:
		return reconcile.Result{}, err
	}

	if dep.Status.Replicas == 0 || dep.Status.Replicas != dep.Status.ReadyReplicas {
		// Deployment is not ready
		// Ask to requeue after 1 minute in order to give enough time for the
		// pods be created on the cluster side and the operand be able
		// to do the next update step accurately.
		log.V(1).Info("deployment not ready yet")
		return ctrl.Result{RequeueAfter: time.Minute}, nil
	}

	log.V(1).Info("inspecting deployment")

	ic := r.imageCloner()

	containers := dep.Spec.Template.Spec.Containers
	for i, el := range containers {
		log.V(1).Info("checking deployment container", "image", el.Image)
		if strings.HasPrefix(el.Image, ic.GetTargetRegistry()) {
			log.V(1).Info("image already cloned", "image", el.Image)
			continue
		}

		log.V(1).Info("cloning deployment image", "image", el.Image)
		dst, err := ic.CloneEventually(el.Image)
		if err != nil {
			return reconcile.Result{}, err
		}
		//log.Info("image cloned", "image", el.Image)

		log.V(1).Info("updating deployment image", "From", el.Image, "To", dst)
		dep.Spec.Template.Spec.Containers[i].Image = dst
		dep.Spec.Template.Spec.ImagePullSecrets = []corev1.LocalObjectReference{{Name: "cr-token"}}
		err = r.Update(context.TODO(), dep)
		if err != nil {
			return reconcile.Result{}, err
		}
		log.V(1).Info("deployment image reference updated", "image", dst)
	}

	return ctrl.Result{}, nil
}

func (r *ReconcileDeployment) isNamespaceToBeExcluded(ns string) bool {
	if len(r.namespacesToSkip) == 0 {
		return false
	}

	for _, el := range r.namespacesToSkip {
		if el == ns {
			return true
		}
	}

	return false
}

func (r *ReconcileDeployment) imageCloner() cloner.Cloner {
	targetRegistry := os.Getenv("IMAGE_CLONER_REGISTRY")
	if targetRegistry == "" {
		targetRegistry = "localhost:5000"
	}

	targetUser := os.Getenv("IMAGE_CLONER_USER")
	targetPass := os.Getenv("IMAGE_CLONER_PASS")

	return cloner.New(targetRegistry,
		cloner.Credentials{
			Username: targetUser,
			Password: targetPass,
		})
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	namespacesToSkip := []string{
		KubeNs,
		LocalNs,
	}

	if ns := os.Getenv("IMAGE_CLONER_SKIP_NAMESPACES"); len(ns) > 0 {
		names := strings.Split(ns, ",")
		for i := range names {
			names[i] = strings.TrimSpace(names[i])
		}

		namespacesToSkip = append(namespacesToSkip, names...)
	}

	return &ReconcileDeployment{
		Client:           mgr.GetClient(),
		scheme:           mgr.GetScheme(),
		namespacesToSkip: namespacesToSkip,
	}
}
