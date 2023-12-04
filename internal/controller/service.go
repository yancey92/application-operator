package controller

import (
	"context"
	"reflect"

	dappsv1 "github.com/yancey92/application-operator/api/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func (r *ApplicationReconciler) reconcileService(ctx context.Context, app *dappsv1.Application) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	var svc = &corev1.Service{}
	err := r.Get(ctx, types.NamespacedName{
		Namespace: app.Namespace,
		Name:      app.Name,
	}, svc)

	if err == nil {
		log.Info("The Service has already exist.")
		if !reflect.DeepEqual(svc.Status, app.Status.Network) {
			app.Status.Network = svc.Status
			if err := r.Status().Update(ctx, app); err != nil {
				log.Error(err, "Failed to update Application Network status")
				return ctrl.Result{RequeueAfter: GenericRequeueDuration}, err
			}
			log.Info("The Application Network status has been updated.")
		}

		if !reflect.DeepEqual(app.Spec.Service.ServiceSpec, svc.Spec) { // 如果 Spec 不相等
			svc.Spec = app.Spec.Service.ServiceSpec
			// svc.Spec.Selector = app.Spec.Deployment.Selector.MatchLabels // 该字段是不允许修改的，所以不建议你修改了CR的 Selector.MatchLabels
			if err := r.Update(ctx, svc); err != nil {
				log.Error(err, "Failed to update Application Network spec")
				return ctrl.Result{RequeueAfter: GenericRequeueDuration}, err
			}
			log.Info("The Application Network spec has been updated.")
		}

		return ctrl.Result{}, nil
	}

	if !errors.IsNotFound(err) {
		log.Error(err, "Failed to get Service, will requeue after a short time.")
		return ctrl.Result{RequeueAfter: GenericRequeueDuration}, err
	}

	newSvc := &corev1.Service{}
	newSvc.SetName(app.Name)
	newSvc.SetNamespace(app.Namespace)
	newSvc.Spec = app.Spec.Service.ServiceSpec
	newSvc.Spec.Selector = app.Spec.Deployment.Selector.MatchLabels

	// 给 app 资源设置 owned，这样删除 app 资源时就可以连带删除它的子资源
	if err := ctrl.SetControllerReference(app, newSvc, r.Scheme); err != nil {
		log.Error(err, "Failed to SetControllerReference, will requeue after a short time.")
		return ctrl.Result{RequeueAfter: GenericRequeueDuration}, err
	}

	if err := r.Create(ctx, newSvc); err != nil {
		log.Error(err, "Failed to create Service, will requeue after a short time.")
		return ctrl.Result{RequeueAfter: GenericRequeueDuration}, err
	}

	log.Info("The Service has been created.")
	return ctrl.Result{}, nil
}
