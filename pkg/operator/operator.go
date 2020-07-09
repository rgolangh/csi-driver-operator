package operator

import (
	"context"
	"os"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	ovirtv1alpha1 "github.com/ovirt/csi-driver-operator/pkg/apis/operator/v1alpha1"
	"github.com/ovirt/csi-driver-operator/pkg/config"
)

const (
	driverImageEnvName              = "RELATED_IMAGE_DRIVER"
	provisionerImageEnvName         = "RELATED_IMAGE_PROVISIONER"
	attacherImageEnvName            = "RELATED_IMAGE_ATTACHER"
	resizerImageEnvName             = "RELATED_IMAGE_RESIZER"
	snapshotterImageEnvName         = "RELATED_IMAGE_SNAPSHOTTER"
	nodeDriverRegistrarImageEnvName = "RELATED_IMAGE_NODE_DRIVER_REGISTRAR"
	livenessProbeImageEnvName       = "RELATED_IMAGE_LIVENESS_PROBE"
)

var log = logf.Log.WithName("controller_ovirtcsioperator")

// Add creates a new OvirtCSIOperator Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileOvirtCSIOperator{
		client:   mgr.GetClient(),
		scheme:   mgr.GetScheme(),
		recorder: mgr.GetEventRecorderFor("ovirt-csi-driver-operator"),
		config: config.Config{
			Images: config.CSIDeploymentContainerImages{
				CSIDriver:            os.Getenv(driverImageEnvName),
				AttacherImage:        os.Getenv(attacherImageEnvName),
				ProvisionerImage:     os.Getenv(provisionerImageEnvName),
				DriverRegistrarImage: os.Getenv(nodeDriverRegistrarImageEnvName),
				LivenessProbeImage:   os.Getenv(livenessProbeImageEnvName),
				ResizerImage:         os.Getenv(resizerImageEnvName),
				SnapshoterImage:      os.Getenv(snapshotterImageEnvName),
			},
		},
	}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("ovirtcsioperator-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource OvirtCSIOperator
	err = c.Watch(&source.Kind{Type: &ovirtv1alpha1.OvirtCSIOperator{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}
	err = c.Watch(&source.Kind{Type: &appsv1.StatefulSet{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &ovirtv1alpha1.OvirtCSIOperator{},
	})
	if err != nil {
		return err
	}

	err = c.Watch(&source.Kind{Type: &appsv1.DaemonSet{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &ovirtv1alpha1.OvirtCSIOperator{},
	})
	if err != nil {
		return err
	}

	err = c.Watch(&source.Kind{Type: &corev1.ServiceAccount{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &ovirtv1alpha1.OvirtCSIOperator{},
	})
	if err != nil {
		return err
	}

	err = c.Watch(&source.Kind{Type: &rbacv1.RoleBinding{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &ovirtv1alpha1.OvirtCSIOperator{},
	})
	if err != nil {
		return err
	}

	err = c.Watch(&source.Kind{Type: &rbacv1.ClusterRoleBinding{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &ovirtv1alpha1.OvirtCSIOperator{},
	})
	if err != nil {
		return err
	}

	err = c.Watch(&source.Kind{Type: &storagev1.StorageClass{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &ovirtv1alpha1.OvirtCSIOperator{},
	})
	if err != nil {
		return err
	}

	return nil
}

// blank assignment to verify that ReconcileOvirtCSIOperator implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileOvirtCSIOperator{}

// ReconcileOvirtCSIOperator reconciles a OvirtCSIOperator object
type ReconcileOvirtCSIOperator struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client   client.Client
	scheme   *runtime.Scheme
	recorder record.EventRecorder
	config   config.Config
}

// Reconcile reads that state of the cluster for a OvirtCSIOperator object and makes changes based on the state read
// and what is in the OvirtCSIOperator.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileOvirtCSIOperator) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling OvirtCSIOperator")

	// Fetch the OvirtCSIOperator instance
	instance := &ovirtv1alpha1.OvirtCSIOperator{}
	err := r.client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		reqLogger.Error(err, "Failed fetching operator instance")
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	// TODO check reconcile errors and report
	return reconcile.Result{}, r.handleCSIDriverDeployment(instance)
}
