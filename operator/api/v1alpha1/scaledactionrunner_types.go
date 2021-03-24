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
	Name              string             `json:"name,omitempty"`
	Namespace         string             `json:"namespace,omitempty"`
	MaxRunners        int32              `json:"maxRunners,omitempty"`
	MinRunners        int32              `json:"minRunners,omitempty"`
	RunnerSecrets     []string           `json:"runnerSecrets,omitempty"`
	GithubTokenSecret string             `json:"githubTokenSecret,omitempty"`
	Owner             string             `json:"owner,omitempty"`
	Repo              string             `json:"repo,omitempty"`
	Image             string             `json:"runnerImage,omitempty"`
	RunnerLabels      string             `json:"runnerLabels,omitempty"`
	WorkVolumeSize    *resource.Quantity `json:"workVolumeSize,omitempty"`
}

const (
	DefaultWorkVolumeSize = "20Gi"
	DefaultImage          = "joeshearn/action-runner-sideloaded-config:7"
)

func Validate(ctx context.Context, sr *ScaledActionRunner, c client.Client) error {
	s := corev1.Secret{}
	checkSecret := func(ctx context.Context, c client.Client, name string, namespace string) error {
		if err := c.Get(ctx, types.NamespacedName{Namespace: sr.Spec.Namespace, Name: sr.Spec.GithubTokenSecret}, &s); err != nil {
			return fmt.Errorf("Could not find secret %s in namespace %s. %s", name, namespace, err.Error())
		}
		return nil
	}
	if err := checkSecret(ctx, c, sr.Spec.Namespace, sr.Spec.GithubTokenSecret); err != nil {
		return err
	}

	for i := int32(0); i < sr.Spec.MaxRunners; i++ {
		name := fmt.Sprintf("%s-%d", sr.Spec.Name, i)
		if err := checkSecret(ctx, c, sr.Spec.Namespace, name); err != nil {
			return err
		}
	}
	return nil
}

func Setup(sr *ScaledActionRunner, crNamespace string) {
	status := &sr.Status
	spec := &sr.Spec

	if status.ReferencedSecrets == nil {
		status.ReferencedSecrets = make(map[string]string)
	}
	if spec.Namespace == "" {
		spec.Namespace = crNamespace
	}
	if spec.Image == "" {
		spec.Image = DefaultImage
	}
	if spec.WorkVolumeSize == nil {
		quantity := resource.MustParse(DefaultWorkVolumeSize)
		spec.WorkVolumeSize = &quantity
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
