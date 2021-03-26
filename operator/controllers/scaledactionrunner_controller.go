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
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"reflect"

	runnerv1alpha1 "github.com/devjoes/github-runner-autoscaler/operator/api/v1alpha1"
	"github.com/devjoes/github-runner-autoscaler/operator/generators"
	"github.com/go-logr/logr"
	keda "github.com/kedacore/keda/v2/api/v1alpha1"
	"github.com/pingcap/errors"
	"github.com/r3labs/diff"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ScaledActionRunnerReconciler reconciles a ScaledActionRunner object
type ScaledActionRunnerReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=runner.devjoes.com,resources=scaledactionrunners,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=runner.devjoes.com,resources=scaledactionrunners/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=runner.devjoes.com,resources=scaledactionrunners/finalizers,verbs=update
// +kubebuilder:rbac:groups=apps,resources=statefulsets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the ScaledActionRunner object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.7.0/pkg/reconcile
func (r *ScaledActionRunnerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("scaledactionrunner", req.NamespacedName)

	runner, err := r.GetScaledActionRunner(ctx, log, req)

	if err != nil {
		return ctrl.Result{}, err
	}
	if runner == nil {
		// Deleted
		e1 := r.deleteDependant(ctx, log, req, &keda.ScaledObject{})
		e2 := r.deleteDependant(ctx, log, req, &appsv1.StatefulSet{})
		if e1 != nil {
			return ctrl.Result{}, e1
		}
		if e2 != nil {
			return ctrl.Result{}, e2
		}
		return ctrl.Result{}, nil
	}

	setModified, err := r.syncStatefulSet(ctx, log, runner)
	if err != nil {
		return ctrl.Result{}, err
	}
	scaledObjectModified, err := r.syncScaledObject(ctx, log, runner)
	if err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{Requeue: setModified || scaledObjectModified}, nil
}

func resourceLog(log logr.Logger, msg string, res client.Object) {
	kind := res.GetObjectKind().GroupVersionKind().Kind
	log.Info(fmt.Sprintf(msg, kind), kind+".Namespace", res.GetNamespace(), kind+".Name", res.GetName())
}

func (r *ScaledActionRunnerReconciler) deleteDependant(ctx context.Context, log logr.Logger, req ctrl.Request, obj client.Object) error {
	(resourceLog(log, "Deleting %s", obj))
	err := r.Get(ctx, types.NamespacedName{Name: req.Name, Namespace: req.Namespace}, obj)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
	}

	if err == nil {
		err = r.Client.Delete(ctx, obj)
	}
	if err != nil {
		(resourceLog(log, "Failed to delete %s", obj))
	}
	return err
}

func (r *ScaledActionRunnerReconciler) GetScaledActionRunner(ctx context.Context, log logr.Logger, req ctrl.Request) (*runnerv1alpha1.ScaledActionRunner, error) {
	scaledActionRunner := &runnerv1alpha1.ScaledActionRunner{}
	runnerv1alpha1.Setup(scaledActionRunner, req.Namespace)
	j, _ := json.Marshal(scaledActionRunner)
	fmt.Printf("Pre get\n\n%s\n\n", string(j))
	err := r.Get(ctx, types.NamespacedName{Name: req.Name, Namespace: req.Namespace}, scaledActionRunner)
	j, _ = json.Marshal(scaledActionRunner)
	fmt.Printf("Post get\n\n%s\n\n", string(j))

	if err != nil {
		if errors.IsNotFound(err) {
			log.Info("ScaledActionRunner resource not found. Ignoring since object must be deleted")
			return nil, nil
		}
		log.Error(err, "Failed to get ScaledActionRunner")
		return nil, err
	}

	if err = runnerv1alpha1.Validate(ctx, scaledActionRunner, r.Client); err != nil {
		return nil, err
	}
	return scaledActionRunner, nil
}
func (r *ScaledActionRunnerReconciler) syncScaledObject(ctx context.Context, log logr.Logger, config *runnerv1alpha1.ScaledActionRunner) (bool, error) {
	//TODO: make these configurable
	metricsEndpoint := "external-metrics-apiserver.runners.svc.cluster.local"
	metricsUrl := fmt.Sprintf("https://%s/apis/external.metrics.k8s.io/v1beta1/namespaces/%s/%s", metricsEndpoint, config.Spec.Namespace, config.Spec.Name)

	var so keda.ScaledObject
	err := r.Get(ctx, types.NamespacedName{Name: config.Spec.Name, Namespace: config.Spec.Namespace}, &so)
	if err != nil {
		if errors.IsNotFound(err) {
			so = *generators.GenerateScaledObject(config, metricsUrl)
			(resourceLog(log, "Creating a new %s", &so))
			ctrl.SetControllerReference(config, &so, r.Scheme)
			err = r.Create(ctx, &so)
			if err != nil {
				log.Error(err, "Failed to create new ScaledObject", "ScaledObject.Namespace", so.Namespace, "ScaledObject.Name", so.Name)
				return false, err
			}
			return true, nil
		} else {
			return false, err
		}
	}
	updatedSo := so.DeepCopy()
	modified := assignScaledObjectPropsFromRunner(updatedSo, config, metricsUrl)

	if modified {
		(resourceLog(log, "Updating %s", &so))
		changes, err := diff.Diff(so, *updatedSo)
		if err != nil {
			log.Error(err, "errored whilst diffing objects")
		}
		log.Info("differences", "changes", changes)
		err = r.Update(ctx, updatedSo)
		if err != nil {
			log.Error(err, "Failed to update ScaledObject "+so.Name)
			return false, err
		}
	}
	return modified, nil
}

func assignScaledObjectPropsFromRunner(found *keda.ScaledObject, config *runnerv1alpha1.ScaledActionRunner, metricsUrl string) bool {
	updated := false
	if found.ObjectMeta.Name != config.Spec.Name {
		found.ObjectMeta.Name = config.Spec.Name
		updated = true
	}
	if found.ObjectMeta.Namespace != config.Spec.Namespace {
		found.ObjectMeta.Namespace = config.Spec.Namespace
		updated = true
	}
	spec := found.Spec
	if config.Spec.Scaling != nil {
		// if config.Spec.Scaling.Behavior != nil {
		// 	spec.Advanced = &keda.AdvancedConfig{
		// 		HorizontalPodAutoscalerConfig: &keda.HorizontalPodAutoscalerConfig{
		// 			Behavior: config.Spec.Scaling.Behavior,
		// 		},
		// 	}
		// }
		spec.CooldownPeriod = config.Spec.Scaling.CooldownPeriod
		spec.PollingInterval = config.Spec.Scaling.PollingInterval
	}
	if spec.MinReplicaCount == nil || *spec.MinReplicaCount != config.Spec.MinRunners {
		spec.MinReplicaCount = &config.Spec.MinRunners
	}
	if spec.MaxReplicaCount == nil || *spec.MaxReplicaCount != config.Spec.MaxRunners {
		spec.MaxReplicaCount = &config.Spec.MaxRunners
	}
	if spec.ScaleTargetRef == nil || spec.Triggers == nil || len(spec.Triggers) == 0 {
		so := generators.GenerateScaledObject(config, metricsUrl)
		spec = so.Spec
	}
	if spec.ScaleTargetRef.Name != config.Spec.Name {
		spec.ScaleTargetRef.Name = config.Spec.Name
	}

	if metricsUrl != spec.Triggers[0].Metadata["url"] {
		spec.Triggers[0].Metadata["url"] = metricsUrl
	}

	if !reflect.DeepEqual(spec, found.Spec) {
		found.Spec = spec
		updated = true
	}
	return updated
}

func (r *ScaledActionRunnerReconciler) getSecretsHash(ctx context.Context, c *runnerv1alpha1.ScaledActionRunner, log logr.Logger) (string, error) {
	sha := sha256.New()
	getHash := func(secName string) ([]byte, error) {
		var secret corev1.Secret
		nsName := types.NamespacedName{
			Namespace: c.Spec.Namespace,
			Name:      secName,
		}
		if err := r.Get(ctx, nsName, &secret); err != nil {
			return []byte{}, err
		}
		ref := fmt.Sprintf("%s/%s", c.Namespace, c.Name)
		if secret.Annotations == nil {
			secret.Annotations = make(map[string]string)
		}
		if secret.Annotations[generators.AnnotationRunnerRef] != ref {
			secret.Annotations[generators.AnnotationRunnerRef] = ref
			r.Update(ctx, &secret)
		}

		data, err := json.Marshal(secret.Data)
		if err != nil {
			return []byte{}, err
		}
		return sha.Sum(data), nil
	}

	secretData, err := getHash(c.Spec.GithubTokenSecret)
	if err != nil {
		return "", err
	}
	for i := 0; i < int(c.Spec.MaxRunners); i++ {
		if i >= len(c.Spec.RunnerSecrets) {
			log.Error(fmt.Errorf("could not find %d th secret in %d RunnerSecrets", i, len(c.Spec.RunnerSecrets)), "error getting secret")
			continue
		}

		h, err := getHash(c.Spec.RunnerSecrets[i])
		if err != nil {
			log.Error(err, "Secret invalid", "secret", c.Spec.RunnerSecrets[i])
		} else {
			secretData = append(secretData, h...)
		}
	}
	return base64.StdEncoding.EncodeToString(sha.Sum(secretData)), nil
}

func (r *ScaledActionRunnerReconciler) syncStatefulSet(ctx context.Context, log logr.Logger, config *runnerv1alpha1.ScaledActionRunner) (bool, error) {
	secretsHash, err := r.getSecretsHash(ctx, config, log)
	if err != nil && errors.IsNotFound(err) {
		return false, fmt.Errorf("Failed to get secrets %s", err.Error())
	}

	newSs := generators.GenerateStatefulSet(config, secretsHash)
	existingSs := &appsv1.StatefulSet{}
	err = r.Get(ctx, types.NamespacedName{Name: config.Spec.Name, Namespace: config.Spec.Namespace}, existingSs)

	if err != nil && errors.IsNotFound(err) {
		(resourceLog(log, "Creating a new StatefulSet %s", newSs))
		ctrl.SetControllerReference(config, newSs, r.Scheme)
		err = r.Create(ctx, newSs)
		if err != nil {
			log.Error(err, "Failed to create new StatefulSet "+newSs.Name)
			return false, err
		}
		// StatefulSet created successfully - return and requeue
		return true, nil
	} else if err != nil {
		log.Error(err, "Failed to get StatefulSet "+config.Spec.Name)
		return false, err
	}

	updatedSs := getScaledSetUpdates(existingSs, config, secretsHash)
	if updatedSs != nil {
		resourceLog(log, "Deleting StatefulSet to be recreated on next reconcile %s", updatedSs)
		changes, err := diff.Diff(existingSs, updatedSs)
		if err != nil {
			log.Error(err, "errored whilst diffing objects")
		}
		log.Info("differences", "changes", changes)

		err = r.Delete(ctx, existingSs)
		if err != nil {
			log.Error(err, "Failed to delete StatefulSet "+updatedSs.Name)
		}
	}
	return updatedSs != nil, nil
}

func getScaledSetUpdates(oldSs *appsv1.StatefulSet, config *runnerv1alpha1.ScaledActionRunner, secretsHash string) *appsv1.StatefulSet {
	updatedSs := oldSs.DeepCopyObject().(*appsv1.StatefulSet)
	updated := false

	if updatedSs.ObjectMeta.Name != config.Spec.Name {
		updatedSs.ObjectMeta.Name = config.Spec.Name
		updated = true
	}
	if updatedSs.ObjectMeta.Namespace != config.Spec.Namespace {
		updatedSs.ObjectMeta.Namespace = config.Spec.Namespace
		updated = true
	}
	if updatedSs.ObjectMeta.Annotations == nil {
		updatedSs.ObjectMeta.Annotations = make(map[string]string)
	}
	if updatedSs.ObjectMeta.Annotations[generators.AnnotationSecretsHash] != secretsHash {
		updatedSs.ObjectMeta.Annotations[generators.AnnotationSecretsHash] = secretsHash
		updated = true
	}
	if updatedSs.Spec.Template.Spec.Containers[0].Image != config.Spec.Runner.Image {
		updatedSs.Spec.Template.Spec.Containers[0].Image = config.Spec.Runner.Image
		updated = true
	}

	if len(updatedSs.Spec.VolumeClaimTemplates) == 0 || !reflect.DeepEqual(updatedSs.Spec.VolumeClaimTemplates[0].Spec, *config.Spec.Runner.WorkVolumeClaimTemplate) {
		updatedSs.Spec.VolumeClaimTemplates = []corev1.PersistentVolumeClaim{
			{
				Spec:   *config.Spec.Runner.WorkVolumeClaimTemplate,
				Status: corev1.PersistentVolumeClaimStatus{},
			},
		}
		updated = true
	}

	updated = generators.SetEnvVars(config, updatedSs) || updated
	volumes, volumeMounts := generators.GetVolumes(config)
	if !reflect.DeepEqual(volumes, updatedSs.Spec.Template.Spec.Volumes) {
		updatedSs.Spec.Template.Spec.Volumes = volumes
		updated = true
	}
	if !reflect.DeepEqual(volumeMounts, updatedSs.Spec.Template.Spec.Containers[0].VolumeMounts) {
		updatedSs.Spec.Template.Spec.Containers[0].VolumeMounts = volumeMounts
		updated = true
	}
	if updated {
		return updatedSs
	}
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ScaledActionRunnerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&runnerv1alpha1.ScaledActionRunner{}).
		Owns(&appsv1.StatefulSet{}). //TODO: https://sdk.operatorframework.io/docs/building-operators/golang/references/event-filtering/
		Complete(r)

}
