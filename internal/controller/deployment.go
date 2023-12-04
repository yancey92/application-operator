package controller

import (
	"context"
	"reflect"

	dappsv1 "github.com/yancey92/application-operator/api/v1"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func (r *ApplicationReconciler) reconcileDeployment(ctx context.Context, app *dappsv1.Application) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	var dp = &appsv1.Deployment{}
	err := r.Get(ctx, types.NamespacedName{
		Namespace: app.Namespace,
		Name:      app.Name,
	}, dp)

	if err == nil {
		log.Info("The Deployment has already exist.")

		if !reflect.DeepEqual(dp.Status, app.Status.Workflow) { // 如果状态不相等
			app.Status.Workflow = dp.Status
			if err := r.Status().Update(ctx, app); err != nil {
				log.Error(err, "Failed to update Application Workflow status")
				return ctrl.Result{RequeueAfter: GenericRequeueDuration}, err
			}
			log.Info("The Application Workflow status has been updated.")
		}

		if !reflect.DeepEqual(app.Spec.Deployment.DeploymentSpec, dp.Spec) { // 如果 Spec 不相等
			dp.Spec = app.Spec.Deployment.DeploymentSpec
			// dp.Spec.Template.SetLabels(app.Spec.Deployment.Selector.MatchLabels) // 该字段是不允许修改的，所以如果你修改了CR的 Selector.MatchLabels，那么将作用不到deployment、svc上
			if err := r.Update(ctx, dp); err != nil {
				log.Error(err, "Failed to update Application Workflow spec")
				return ctrl.Result{RequeueAfter: GenericRequeueDuration}, err
			}
			log.Info("The Application Workflow spec has been updated.")
		}

		return ctrl.Result{}, nil
	}

	if !errors.IsNotFound(err) {
		log.Error(err, "Failed to get Deployment, will requeue after a short time.")
		return ctrl.Result{RequeueAfter: GenericRequeueDuration}, err
	}

	// 如果对应的 deployment 不存在，就创建它
	newDp := &appsv1.Deployment{}
	newDp.SetName(app.Name)
	newDp.SetNamespace(app.Namespace)
	newDp.Spec = app.Spec.Deployment.DeploymentSpec
	newDp.Spec.Template.SetLabels(app.Spec.Deployment.Selector.MatchLabels)

	// 给 app 资源设置 owned，这样删除 app 资源时就可以连带删除它的子资源
	if err := ctrl.SetControllerReference(app, newDp, r.Scheme); err != nil {
		log.Error(err, "Failed to SetControllerReference, will requeue after a short time.")
		return ctrl.Result{RequeueAfter: GenericRequeueDuration}, err
	}

	if err := r.Create(ctx, newDp); err != nil {
		log.Error(err, "Failed to create Deployment, will requeue after a short time.")
		return ctrl.Result{RequeueAfter: GenericRequeueDuration}, err
	}

	log.Info("The Deployment has been created.")
	return ctrl.Result{}, nil
}
