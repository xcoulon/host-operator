package activationcode

import (
	"context"
	"fmt"

	toolchainv1alpha1 "github.com/codeready-toolchain/api/api/v1alpha1"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

// SetupWithManager sets up the controller with the Manager.
func (r *ActivationCodeReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&toolchainv1alpha1.ActivationCode{}).
		Watches(&source.Kind{Type: &toolchainv1alpha1.UserSignup{}}, handler.EnqueueRequestsFromMapFunc(MapUserSignupToActivationCode())).
		Complete(r)
}

// ActivationCodeReconciler reconciles a ActivationCode object
type ActivationCodeReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=toolchain.dev.openshift.com,resources=activationcodes,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=toolchain.dev.openshift.com,resources=activationcodes/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=toolchain.dev.openshift.com,resources=activationcodes/finalizers,verbs=update

// Updates the ActivationCode status when a user signs up with an activation code (set as a label on her associated UserSignup resource)
// For each signup, the `ActivationCode.Status.NumberOfUsers` counter is increased.
// Note: It's the responsibility of the Registration Service to make sure that the `ActivationCode.Status.NumberOfUsers <= ActivationCode.Status.MaxNumberOfUsers`
func (r *ActivationCodeReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("Reconciling ActivationCode")

	// fetch the ActivationCode instance
	code := &toolchainv1alpha1.ActivationCode{}
	if err := r.Get(context.TODO(), req.NamespacedName, code); err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		logger.Error(err, "unable to get ActivationCode")
		return reconcile.Result{}, err
	}

	// count UserSignups labeled for this ActivationCode
	userSignups := &toolchainv1alpha1.UserSignupList{}
	if err := r.List(context.TODO(), userSignups, client.InNamespace(code.Namespace), client.MatchingLabels{
		toolchainv1alpha1.UserSignupActivationCodeLabelKey: code.Name,
	}); err != nil {
		logger.Error(err, fmt.Sprintf("unable to list UserSignups for ActivationCode %q", code.Name))
		return reconcile.Result{}, err
	}
	// update the ActivationCode with the number of UserSignups
	code.Status.NumberOfUsers = len(userSignups.Items)
	if err := r.Update(context.TODO(), code); err != nil {
		logger.Error(err, fmt.Sprintf("unable to update ActivationCode %q", code.Name))
		return reconcile.Result{}, err
	}
	return ctrl.Result{}, nil
}
