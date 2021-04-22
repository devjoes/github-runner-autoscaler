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

package v1alpha1

import (
	"context"
	"fmt"
	"strconv"

	autoscalingv2beta2 "k8s.io/api/autoscaling/v2beta2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// ScaledActionRunnerSpec defines the desired state of ScaledActionRunner
type ScaledActionRunnerSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Foo is an example field of ScaledActionRunner. Edit ScaledActionRunner_types.go to remove/update
	MaxRunners        int32    `json:"maxRunners"`
	MinRunners        int32    `json:"minRunners,omitempty"`
	RunnerSecrets     []string `json:"runnerSecrets"`
	GithubTokenSecret string   `json:"githubTokenSecret"`
	Owner             string   `json:"owner"`
	Repo              string   `json:"repo"`
	Scaling           *Scaling `json:"scaling,omitempty"`
	ScaleFactor       *string  `json:"scaleFactor,omitempty"`
	MetricsSelector   *string  `json:"metricsSelector,omitempty"`
	Runner            *Runner  `json:"runner,omitempty"`
}

type Runner struct {
	Image                   string                                     `json:"image,omitempty"`
	RunnerLabels            string                                     `json:"runnerLabels,omitempty"`
	Annotations             map[string]string                          `json:"annotations,omitempty"`
	NodeSelector            map[string]string                          `json:"nodeSelector,omitempty"`
	WorkVolumeClaimTemplate *corev1.PersistentVolumeClaimSpec          `json:"workVolumeClaimTemplate,omitempty"`
	Limits                  *map[corev1.ResourceName]resource.Quantity `json:"limits,omitempty"`
	Requests                *map[corev1.ResourceName]resource.Quantity `json:"requests,omitempty"`
	Tolerations             []corev1.Toleration                        `json:"tolerations,omitempty"`
}

type Scaling struct {
	Behavior        *autoscalingv2beta2.HorizontalPodAutoscalerBehavior `json:"behavior,omitempty"`
	PollingInterval *int32                                              `json:"pollingInterval,omitempty"`
	CooldownPeriod  *int32                                              `json:"cooldownPeriod,omitempty"`
}

const (
	DefaultWorkVolumeSize = "5Gi"
	DefaultImage          = "myoung34/github-runner:latest"
)

func Validate(ctx context.Context, sr *ScaledActionRunner, c client.Client, apiServerNs string) error {
	s := corev1.Secret{}
	checkSecret := func(ctx context.Context, c client.Client, name string, namespace string) error {
		if err := c.Get(ctx, types.NamespacedName{Namespace: namespace, Name: name}, &s); err != nil {
			return fmt.Errorf("Could not find secret %s in namespace %s. %s", name, namespace, err.Error())
		}
		return nil
	}
	if err := checkSecret(ctx, c, sr.Spec.GithubTokenSecret, sr.ObjectMeta.Namespace); err != nil {
		if err := checkSecret(ctx, c, sr.Spec.GithubTokenSecret, apiServerNs); err != nil {
			return err
		}
	}

	for _, name := range sr.Spec.RunnerSecrets {
		if err := checkSecret(ctx, c, name, sr.ObjectMeta.Namespace); err != nil {
			return err
		}
	}
	_, err := strconv.ParseFloat(*sr.Spec.ScaleFactor, 64)
	if err != nil {
		return fmt.Errorf("Could not parse %s as a float64", *sr.Spec.ScaleFactor)
	}
	return nil
}

func Setup(sr *ScaledActionRunner, crNamespace string) {
	status := &sr.Status
	spec := &sr.Spec

	if status.ReferencedSecrets == nil {
		status.ReferencedSecrets = make(map[string]string)
	}

	if spec.Runner == nil {
		spec.Runner = &Runner{}
	}
	if spec.Runner.Image == "" {
		spec.Runner.Image = DefaultImage
	}
	if spec.Runner.Requests == nil {
		spec.Runner.Requests = &map[corev1.ResourceName]resource.Quantity{
			corev1.ResourceCPU:    resource.MustParse("200m"),
			corev1.ResourceMemory: resource.MustParse("200Mi"),
		}
	}
	if spec.Runner.Limits == nil {
		spec.Runner.Limits = &map[corev1.ResourceName]resource.Quantity{
			corev1.ResourceCPU:    resource.MustParse("2000m"),
			corev1.ResourceMemory: resource.MustParse("2000Mi"),
		}
	}
	if spec.Runner.WorkVolumeClaimTemplate == nil {
		filesystmem := "Filesystem"
		spec.Runner.WorkVolumeClaimTemplate = &corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: resource.MustParse(DefaultWorkVolumeSize),
				},
			},
			VolumeMode: (*corev1.PersistentVolumeMode)(&filesystmem),
		}
	}
	if spec.ScaleFactor == nil {
		sf := "0.8"
		spec.ScaleFactor = &sf
	}

}

// ScaledActionRunnerStatus defines the observed state of ScaledActionRunner
type ScaledActionRunnerStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	ReferencedSecrets map[string]string `json:"referencedSecrets,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// ScaledActionRunner is the Schema for the scaledactionrunners API
type ScaledActionRunner struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ScaledActionRunnerSpec   `json:"spec,omitempty"`
	Status ScaledActionRunnerStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ScaledActionRunnerList contains a list of ScaledActionRunner
type ScaledActionRunnerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ScaledActionRunner `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ScaledActionRunner{}, &ScaledActionRunnerList{})
}
