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
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// ActionRunnerMetricsSpec defines the desired state of ActionRunnerMetrics
type ActionRunnerMetricsSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Foo is an example field of ActionRunnerMetrics. Edit ActionRunnerMetrics_types.go to remove/update
	Image                       string        `json:"image"`
	Replicas                    int32         `json:"replicas"`
	CreateApiServer             bool          `json:"createApiServer"`
	CreateMemcached             bool          `json:"createMemcached"`
	MemcachedReplicas           int32         `json:"memcachedReplicas"`
	CreateAuthentication        bool          `json:"createAuthentication"`
	ExistingSslCertSecret       string        `json:"existingSslCertSecret"`
	ExistingMemcacheCredsSecret string        `json:"existingMemcacheCredsSecret"`
	ExistingMemcacheUser        string        `json:"existingMemcacheUser"`
	ExistingMemcacheServers     string        `json:"existingMemcacheServers"`
	CacheWindow                 time.Duration `json:"cacheWindow"`
	CacheWindowWhenEmpty        time.Duration `json:"cacheWindowWhenEmpty"`
	ResyncInterval              time.Duration `json:"resyncInterval"`
	Namespaces                  []string      `json:"namespaces"`
}

// ActionRunnerMetricsStatus defines the observed state of ActionRunnerMetrics
type ActionRunnerMetricsStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// ActionRunnerMetrics is the Schema for the actionrunnermetrics API
type ActionRunnerMetrics struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ActionRunnerMetricsSpec   `json:"spec,omitempty"`
	Status ActionRunnerMetricsStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ActionRunnerMetricsList contains a list of ActionRunnerMetrics
type ActionRunnerMetricsList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ActionRunnerMetrics `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ActionRunnerMetrics{}, &ActionRunnerMetricsList{})
}
