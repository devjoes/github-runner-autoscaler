/*
Copyright 2021.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	"github.com/pingcap/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	runnerv1alpha1 "github.com/devjoes/github-runner-autoscaler/operator/api/v1alpha1"
	"github.com/devjoes/github-runner-autoscaler/operator/armgenerator"
)

// ActionRunnerMetricsReconciler reconciles a ActionRunnerMetrics object
type ActionRunnerMetricsReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=runner.devjoes.com,resources=actionrunnermetrics,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=runner.devjoes.com,resources=actionrunnermetrics/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=runner.devjoes.com,resources=actionrunnermetrics/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the ActionRunnerMetrics object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.7.0/pkg/reconcile
func (r *ActionRunnerMetricsReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("actionrunnermetrics", req.NamespacedName)
	changed := false
	metrics, err := r.GetActionRunnerMetrics(ctx, log, req)
	if err != nil {
		return ctrl.Result{}, err
	}
	if metrics == nil {
		log.Info("CRD Deleted")
		return ctrl.Result{}, nil
	}
	o, err := armgenerator.GenerateMemcachedResources(metrics)
	if err != nil {
		return ctrl.Result{}, err
	}
	objs := append(armgenerator.GenerateMetricsApiServer(metrics), o...)

	objs = append(objs, armgenerator.GenerateAuthTrigger(metrics)...)
	deploy := func(toDeploy []client.Object, preReqsOnly bool) (bool, error) {
		c := false
		for _, o := range toDeploy {
			k := o.GetObjectKind().GroupVersionKind().Kind
			isPreReq := k == "Secret" || k == "ServiceAccount"
			if preReqsOnly != isPreReq {
				continue
			}
			objChanged, e := r.CreateUpdateOrReplace(ctx, log, metrics, o)
			if e != nil {
				return false, e
			}
			c = c || objChanged
		}
		return c, nil
	}
	c, err := deploy(objs, true)
	changed = c || changed
	if err != nil {
		return ctrl.Result{}, err
	}
	c, err = deploy(objs, false)
	if err != nil {
		return ctrl.Result{}, err
	}
	changed = c || changed

	return ctrl.Result{Requeue: changed}, nil
}
func (r *ActionRunnerMetricsReconciler) CreateUpdateOrReplace(ctx context.Context, log logr.Logger, crd *runnerv1alpha1.ActionRunnerMetrics, obj client.Object) (bool, error) {
	logMsg := func(msg string, obj client.Object) {
		label := fmt.Sprintf("%s %s/%s", obj.GetObjectKind().GroupVersionKind().Kind, obj.GetNamespace(), obj.GetName())
		log.Info(fmt.Sprintf(msg, label))
	}
	old := unstructured.Unstructured{}
	ctrl.SetControllerReference(crd, obj, r.Scheme)
	old.SetGroupVersionKind(obj.GetObjectKind().GroupVersionKind())
	err := r.Client.Get(ctx, client.ObjectKeyFromObject(obj), &old)
	if err != nil {
		if !errors.IsNotFound(err) {
			return false, err
		}
	} else {
		oldKey, foundOld := old.GetAnnotations()[armgenerator.CrdKey]
		newKey, foundNew := obj.GetAnnotations()[armgenerator.CrdKey]

		if foundNew && foundOld && oldKey == newKey {
			logMsg("Ignoring unchanged %s", obj)
			return false, nil
		}
		logMsg("Trying to update %s", obj)
		obj.SetResourceVersion(old.GetResourceVersion())
		err = r.Update(ctx, obj, &client.UpdateOptions{})
		if err == nil {
			return true, nil
		}
		obj.SetResourceVersion("")
		exists := true
		iterations := 0
		logMsg("Deleting %s: "+err.Error(), obj)
		r.Delete(ctx, obj)
		for exists && iterations < 60*5 {
			time.Sleep(time.Millisecond * 200)
			exists = r.Get(ctx, client.ObjectKeyFromObject(obj), &old) == nil
			iterations++
		}
	}
	logMsg("Creating %s", obj)
	ctrl.SetControllerReference(crd, obj, r.Scheme)
	err = r.Create(ctx, obj, &client.CreateOptions{})
	//TODO: handle deletions - e.g. if CreateApiServer was true but is now false
	if err != nil {
		return false, err
	}
	return true, nil
}
func (r *ActionRunnerMetricsReconciler) GetActionRunnerMetrics(ctx context.Context, log logr.Logger, req ctrl.Request) (*runnerv1alpha1.ActionRunnerMetrics, error) {
	actionRunnerMetrics := &runnerv1alpha1.ActionRunnerMetrics{}
	err := r.Get(ctx, types.NamespacedName{Name: req.Name, Namespace: req.Namespace}, actionRunnerMetrics)

	if err != nil {
		if errors.IsNotFound(err) {
			log.Info("ActionRunnerMetrics resource not found. Ignoring since object must be deleted")
			return nil, nil
		}
		log.Error(err, "Failed to get ActionRunnerMetrics")
		return nil, err
	}

	actionRunnerMetrics.Setup()
	return actionRunnerMetrics, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ActionRunnerMetricsReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&runnerv1alpha1.ActionRunnerMetrics{}).
		Complete(r)
}
