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
	"strings"

	runnerv1alpha1 "github.com/devjoes/github-runner-autoscaler/operator/api/v1alpha1"
	gen "github.com/devjoes/github-runner-autoscaler/operator/sargenerator"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// SecretReconciler reconciles a Secret object
type SecretReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=secrets/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=core,resources=secrets/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Secret object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.7.0/pkg/reconcile
func (r *SecretReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = r.Log.WithValues("secret", req.NamespacedName)
	secret := &corev1.Secret{}
	err := r.Get(ctx, types.NamespacedName{Name: req.Name, Namespace: req.Namespace}, secret)
	if err != nil {
		return ctrl.Result{}, err
	}
	if secret.Annotations[gen.AnnotationRunnerRef] != "" {
		ref := strings.Split(secret.Annotations[gen.AnnotationRunnerRef], "/")
		modified, err := r.updateRunnerStatus(ctx, ref[0], ref[1], secret)
		if err == nil && modified {
			return ctrl.Result{Requeue: true}, nil
		}
	}
	return ctrl.Result{}, err
}

func (r *SecretReconciler) updateRunnerStatus(ctx context.Context, namespace string, name string, secret *corev1.Secret) (bool, error) {
	config := &runnerv1alpha1.ScaledActionRunner{}
	err := r.Get(ctx, types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}, config)
	if err != nil {
		return false, err
	}
	if config.Status.ReferencedSecrets == nil {
		config.Status.ReferencedSecrets = make(map[string]string)
	}
	if config.Status.ReferencedSecrets[string(secret.UID)] != secret.ResourceVersion {
		config.Status.ReferencedSecrets[string(secret.UID)] = secret.ResourceVersion
		err = r.Update(ctx, config)
		if err != nil {
			return false, err
		}
		return true, nil
	}
	return false, nil
}

func withAnnotation(annotation string) predicate.Predicate {
	return predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			_, exists := e.ObjectNew.GetAnnotations()[annotation]
			return exists
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			_, exists := e.Object.GetAnnotations()[annotation]
			return exists
		},
		CreateFunc: func(e event.CreateEvent) bool {
			_, exists := e.Object.GetAnnotations()[annotation]
			return exists
		},
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *SecretReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Secret{}).
		WithEventFilter(withAnnotation(gen.AnnotationRunnerRef)).
		Complete(r)
}
