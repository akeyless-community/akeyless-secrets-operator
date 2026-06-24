/*
Copyright Akeyless Community

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    https://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// DynamicSecretType restricts which Akeyless item types may be synced.
// +kubebuilder:validation:Enum=Dynamic;Rotated
type DynamicSecretType string

const (
	DynamicSecretTypeDynamic DynamicSecretType = "Dynamic"
	DynamicSecretTypeRotated DynamicSecretType = "Rotated"
)

// AkeylessDynamicSecretSpec syncs a single dynamic or rotated Akeyless item.
type AkeylessDynamicSecretSpec struct {
	StoreRef SecretStoreRef `json:"storeRef"`
	Target   Target         `json:"target"`

	// Path is the Akeyless item path (dynamic-secret or rotated-secret).
	Path string `json:"path"`

	// Type selects dynamic vs rotated secret handling. Defaults to Dynamic.
	// +kubebuilder:default=Dynamic
	Type DynamicSecretType `json:"type,omitempty"`

	// SecretKey is the key written in the Kubernetes Secret. Defaults to "value".
	// +kubebuilder:default=value
	SecretKey string `json:"secretKey,omitempty"`

	// Property extracts a field from JSON payloads returned by Akeyless.
	// +optional
	Property string `json:"property,omitempty"`

	// +kubebuilder:default=OnRemoteChange
	SyncPolicy SyncPolicy `json:"syncPolicy,omitempty"`

	// +kubebuilder:default="1m0s"
	SyncInterval *metav1.Duration `json:"syncInterval,omitempty"`

	// RolloutRestartTargets restart workloads when synced secret data changes.
	// +optional
	RolloutRestartTargets []RolloutRestartTarget `json:"rolloutRestartTargets,omitempty"`
}

// AkeylessDynamicSecretStatus defines observed sync state.
type AkeylessDynamicSecretStatus struct {
	RefreshTime      metav1.Time                    `json:"refreshTime,omitempty"`
	SyncedGeneration int64                          `json:"syncedGeneration,omitempty"`
	RemoteVersions   map[string]int32               `json:"remoteVersions,omitempty"`
	Conditions       []AkeylessDynamicSecretCondition `json:"conditions,omitempty"`
}

// AkeylessDynamicSecretCondition reports sync readiness.
type AkeylessDynamicSecretCondition = SecretCondition

// AkeylessDynamicSecret syncs dynamic or rotated Akeyless credentials into Kubernetes.
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:storageversion
// +kubebuilder:metadata:labels="secrets.akeyless.io/component=controller"
// +kubebuilder:resource:scope=Namespaced,categories={akeyless-secrets},shortName=ads
// +kubebuilder:printcolumn:name="Path",type=string,JSONPath=`.spec.path`
// +kubebuilder:printcolumn:name="Type",type=string,JSONPath=`.spec.type`
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
// +kubebuilder:printcolumn:name="Last Sync",type=date,JSONPath=`.status.refreshTime`
type AkeylessDynamicSecret struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AkeylessDynamicSecretSpec   `json:"spec,omitempty"`
	Status AkeylessDynamicSecretStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
type AkeylessDynamicSecretList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AkeylessDynamicSecret `json:"items"`
}
