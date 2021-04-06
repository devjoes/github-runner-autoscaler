package sargenerator

// import (
// 	autoscalingv2beta2 "k8s.io/api/autoscaling/v2beta2"
// 	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
// 	"k8s.io/apimachinery/pkg/runtime/schema"
// )

// // There is a reference incompatability between which causes issues so I've just dropped the type in here
// type ScaledObject struct {
// 	metav1.TypeMeta   `json:",inline"`
// 	metav1.ObjectMeta `json:"metadata,omitempty"`

// 	Spec ScaledObjectSpec `json:"spec"`
// 	// +optional
// 	Status ScaledObjectStatus `json:"status,omitempty"`
// }

// // ScaledObjectSpec is the spec for a ScaledObject resource
// type ScaledObjectSpec struct {
// 	ScaleTargetRef *ScaleTarget `json:"scaleTargetRef"`
// 	// +optional
// 	PollingInterval *int32 `json:"pollingInterval,omitempty"`
// 	// +optional
// 	CooldownPeriod *int32 `json:"cooldownPeriod,omitempty"`
// 	// +optional
// 	MinReplicaCount *int32 `json:"minReplicaCount,omitempty"`
// 	// +optional
// 	MaxReplicaCount *int32 `json:"maxReplicaCount,omitempty"`
// 	// +optional
// 	Advanced *AdvancedConfig `json:"advanced,omitempty"`

// 	Triggers []ScaleTriggers `json:"triggers"`
// }

// // AdvancedConfig specifies advance scaling options
// type AdvancedConfig struct {
// 	// +optional
// 	HorizontalPodAutoscalerConfig *HorizontalPodAutoscalerConfig `json:"horizontalPodAutoscalerConfig,omitempty"`
// 	// +optional
// 	RestoreToOriginalReplicaCount bool `json:"restoreToOriginalReplicaCount,omitempty"`
// }

// // HorizontalPodAutoscalerConfig specifies horizontal scale config
// type HorizontalPodAutoscalerConfig struct {
// 	// +optional
// 	Behavior *autoscalingv2beta2.HorizontalPodAutoscalerBehavior `json:"behavior,omitempty"`
// }

// // ScaleTarget holds the a reference to the scale target Object
// type ScaleTarget struct {
// 	Name string `json:"name"`
// 	// +optional
// 	APIVersion string `json:"apiVersion,omitempty"`
// 	// +optional
// 	Kind string `json:"kind,omitempty"`
// 	// +optional
// 	EnvSourceContainerName string `json:"envSourceContainerName,omitempty"`
// }

// // ScaleTriggers reference the scaler that will be used
// type ScaleTriggers struct {
// 	Type string `json:"type"`
// 	// +optional
// 	Name     string            `json:"name,omitempty"`
// 	Metadata map[string]string `json:"metadata"`
// 	// +optional
// 	AuthenticationRef *ScaledObjectAuthRef `json:"authenticationRef,omitempty"`
// }

// // ScaledObjectStatus is the status for a ScaledObject resource
// // +optional
// type ScaledObjectStatus struct {
// 	// +optional
// 	ScaleTargetKind string `json:"scaleTargetKind,omitempty"`
// 	// +optional
// 	ScaleTargetGVKR *GroupVersionKindResource `json:"scaleTargetGVKR,omitempty"`
// 	// +optional
// 	OriginalReplicaCount *int32 `json:"originalReplicaCount,omitempty"`
// 	// +optional
// 	LastActiveTime *metav1.Time `json:"lastActiveTime,omitempty"`
// 	// +optional
// 	ExternalMetricNames []string `json:"externalMetricNames,omitempty"`
// 	// +optional
// 	ResourceMetricNames []string `json:"resourceMetricNames,omitempty"`
// 	// +optional
// 	Conditions Conditions `json:"conditions,omitempty"`
// }

// // +kubebuilder:object:root=true

// // ScaledObjectList is a list of ScaledObject resources
// type ScaledObjectList struct {
// 	metav1.TypeMeta `json:",inline"`
// 	metav1.ListMeta `json:"metadata"`
// 	Items           []ScaledObject `json:"items"`
// }

// // ScaledObjectAuthRef points to the TriggerAuthentication or ClusterTriggerAuthentication object that
// // is used to authenticate the scaler with the environment
// type ScaledObjectAuthRef struct {
// 	Name string `json:"name"`
// 	// Kind of the resource being referred to. Defaults to TriggerAuthentication.
// 	// +optional
// 	Kind string `json:"kind,omitempty"`
// }

// // GroupVersionKindResource provides unified structure for schema.GroupVersionKind and Resource
// type GroupVersionKindResource struct {
// 	Group    string `json:"group"`
// 	Version  string `json:"version"`
// 	Kind     string `json:"kind"`
// 	Resource string `json:"resource"`
// }

// // GroupVersionKind returns the group, version and kind of GroupVersionKindResource
// func (gvkr GroupVersionKindResource) GroupVersionKind() schema.GroupVersionKind {
// 	return schema.GroupVersionKind{Group: gvkr.Group, Version: gvkr.Version, Kind: gvkr.Kind}
// }

// // GroupVersion returns the group and version of GroupVersionKindResource
// func (gvkr GroupVersionKindResource) GroupVersion() schema.GroupVersion {
// 	return schema.GroupVersion{Group: gvkr.Group, Version: gvkr.Version}
// }

// // GroupResource returns the group and resource of GroupVersionKindResource
// func (gvkr GroupVersionKindResource) GroupResource() schema.GroupResource {
// 	return schema.GroupResource{Group: gvkr.Group, Resource: gvkr.Resource}
// }

// // GVKString returns the group, version and kind in string format
// func (gvkr GroupVersionKindResource) GVKString() string {
// 	return gvkr.Group + "/" + gvkr.Version + "." + gvkr.Kind
// }

// // ConditionType specifies the available conditions for the resource
// type ConditionType string

// const (
// 	// ConditionReady specifies that the resource is ready.
// 	// For long-running resources.
// 	ConditionReady ConditionType = "Ready"
// 	// ConditionActive specifies that the resource has finished.
// 	// For resource which run to completion.
// 	ConditionActive ConditionType = "Active"
// )

// // Condition to store the condition state
// type Condition struct {
// 	// Type of condition
// 	// +required
// 	Type ConditionType `json:"type" description:"type of status condition"`

// 	// Status of the condition, one of True, False, Unknown.
// 	// +required
// 	Status metav1.ConditionStatus `json:"status" description:"status of the condition, one of True, False, Unknown"`

// 	// The reason for the condition's last transition.
// 	// +optional
// 	Reason string `json:"reason,omitempty" description:"one-word CamelCase reason for the condition's last transition"`

// 	// A human readable message indicating details about the transition.
// 	// +optional
// 	Message string `json:"message,omitempty" description:"human-readable message indicating details about last transition"`
// }

// // Conditions an array representation to store multiple Conditions
// type Conditions []Condition

// // AreInitialized performs check all Conditions are initialized
// // return true if Conditions are initialized
// // return false if Conditions are not initialized
// func (c *Conditions) AreInitialized() bool {
// 	foundReady := false
// 	foundActive := false
// 	if *c != nil {
// 		for _, condition := range *c {
// 			if condition.Type == ConditionReady {
// 				foundReady = true
// 				break
// 			}
// 		}
// 		for _, condition := range *c {
// 			if condition.Type == ConditionActive {
// 				foundActive = true
// 				break
// 			}
// 		}
// 	}

// 	return foundReady && foundActive
// }

// // GetInitializedConditions returns Conditions initialized to the default -> Status: Unknown
// func GetInitializedConditions() *Conditions {
// 	return &Conditions{{Type: ConditionReady, Status: metav1.ConditionUnknown}, {Type: ConditionActive, Status: metav1.ConditionUnknown}}
// }

// // IsTrue is true if the condition is True
// func (c *Condition) IsTrue() bool {
// 	if c == nil {
// 		return false
// 	}
// 	return c.Status == metav1.ConditionTrue
// }

// // IsFalse is true if the condition is False
// func (c *Condition) IsFalse() bool {
// 	if c == nil {
// 		return false
// 	}
// 	return c.Status == metav1.ConditionFalse
// }

// // IsUnknown is true if the condition is Unknown
// func (c *Condition) IsUnknown() bool {
// 	if c == nil {
// 		return true
// 	}
// 	return c.Status == metav1.ConditionUnknown
// }

// // SetReadyCondition modifies Ready Condition according to input parameters
// func (c *Conditions) SetReadyCondition(status metav1.ConditionStatus, reason string, message string) {
// 	if *c == nil {
// 		c = GetInitializedConditions()
// 	}
// 	c.setCondition(ConditionReady, status, reason, message)
// }

// // SetActiveCondition modifies Active Condition according to input parameters
// func (c *Conditions) SetActiveCondition(status metav1.ConditionStatus, reason string, message string) {
// 	if *c == nil {
// 		c = GetInitializedConditions()
// 	}
// 	c.setCondition(ConditionActive, status, reason, message)
// }

// // GetActiveCondition returns Condition of type Active
// func (c *Conditions) GetActiveCondition() Condition {
// 	if *c == nil {
// 		c = GetInitializedConditions()
// 	}
// 	return c.getCondition(ConditionActive)
// }

// // GetReadyCondition returns Condition of type Ready
// func (c *Conditions) GetReadyCondition() Condition {
// 	if *c == nil {
// 		c = GetInitializedConditions()
// 	}
// 	return c.getCondition(ConditionReady)
// }

// func (c Conditions) getCondition(conditionType ConditionType) Condition {
// 	for i := range c {
// 		if c[i].Type == conditionType {
// 			return c[i]
// 		}
// 	}
// 	return Condition{}
// }

// func (c Conditions) setCondition(conditionType ConditionType, status metav1.ConditionStatus, reason string, message string) {
// 	for i := range c {
// 		if c[i].Type == conditionType {
// 			c[i].Status = status
// 			c[i].Reason = reason
// 			c[i].Message = message
// 			break
// 		}
// 	}
// }
