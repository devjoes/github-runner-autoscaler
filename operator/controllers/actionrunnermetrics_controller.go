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

// ScaledActionRunnerCoreReconciler reconciles a ScaledActionRunnerCore object
type ScaledActionRunnerCoreReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=runner.devjoes.com,resources=scaledactionrunnercore,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=runner.devjoes.com,resources=scaledactionrunnercore/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=runner.devjoes.com,resources=scaledactionrunnercore/finalizers,verbs=update
// +kubebuilder:rbac:groups=apps,resources=statefulsets;deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=secrets;serviceaccounts;services;configmaps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apiregistration.k8s.io,resources=apiservices,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=monitoring.coreos.com,resources=servicemonitors,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the ScaledActionRunnerCore object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.7.0/pkg/reconcile
func (r *ScaledActionRunnerCoreReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("scaledactionrunnercore", req.NamespacedName)
	changed := false
	metrics, err := r.GetScaledActionRunnerCore(ctx, log, req)
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
	objs := []client.Object{}
	objs = append(objs, o...)
	o2 := armgenerator.GenerateMetricsApiServer(metrics)
	objs = append(objs, o2...)
	objs = append(objs, armgenerator.GeneratePrometheusServiceMonitor(metrics)...)
	objs = append(objs, armgenerator.GenerateAuthTrigger(metrics)...)

	deploy := func(toDeploy []client.Object, preReqsOnly bool) (bool, error) {
		c := false
		for _, o := range toDeploy {
			k := o.GetObjectKind().GroupVersionKind().Kind
			isPreReq := k == "Secret" || k == "ServiceAccount"
			alreadyCreated := (k == "" && o.GetResourceVersion() != "")
			if preReqsOnly != isPreReq || alreadyCreated {
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
func (r *ScaledActionRunnerCoreReconciler) CreateUpdateOrReplace(ctx context.Context, log logr.Logger, crd *runnerv1alpha1.ScaledActionRunnerCore, obj client.Object) (bool, error) {
	logMsg := func(msg string, obj client.Object) {
		label := fmt.Sprintf("%s %s/%s", obj.GetObjectKind().GroupVersionKind().Kind, obj.GetNamespace(), obj.GetName())
		log.Info(fmt.Sprintf(msg, label))
	}
	old := unstructured.Unstructured{}
	gvk := obj.GetObjectKind().GroupVersionKind()

	old.SetGroupVersionKind(gvk)
	ctrl.SetControllerReference(crd, obj, r.Scheme)
	key := client.ObjectKeyFromObject(obj)
	err := r.Client.Get(ctx, key, &old)
	if err != nil {
		if !errors.IsNotFound(err) {
			return false, err
		}
	} else {
		oldKey, foundOld := old.GetAnnotations()[armgenerator.CrdKey]
		newKey, foundNew := obj.GetAnnotations()[armgenerator.CrdKey]

		if foundNew && foundOld && oldKey == newKey {
			label := fmt.Sprintf("%s %s/%s", obj.GetObjectKind().GroupVersionKind().Kind, obj.GetNamespace(), obj.GetName())
			log.V(5).Info(fmt.Sprintf("Ignoring unchanged %s", label))
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
func (r *ScaledActionRunnerCoreReconciler) GetScaledActionRunnerCore(ctx context.Context, log logr.Logger, req ctrl.Request) (*runnerv1alpha1.ScaledActionRunnerCore, error) {
	scaledActionRunnerCore := &runnerv1alpha1.ScaledActionRunnerCore{}
	err := r.Get(ctx, types.NamespacedName{Name: req.Name, Namespace: req.Namespace}, scaledActionRunnerCore)

	if err != nil {
		if errors.IsNotFound(err) {
			log.Info("ScaledActionRunnerCore resource not found. Ignoring since object must be deleted")
			return nil, nil
		}
		log.Error(err, "Failed to get ScaledActionRunnerCore")
		return nil, err
	}

	scaledActionRunnerCore.Setup()
	return scaledActionRunnerCore, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ScaledActionRunnerCoreReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&runnerv1alpha1.ScaledActionRunnerCore{}).
		Complete(r)
}
