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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// SecretStoreRef references an AkeylessSecretStore or ClusterAkeylessSecretStore.
type SecretStoreRef struct {
	Name string `json:"name"`
	// +kubebuilder:validation:Enum=AkeylessSecretStore;ClusterAkeylessSecretStore
	// +kubebuilder:default=AkeylessSecretStore
	Kind string `json:"kind,omitempty"`
}

// CreationPolicy defines how the target Secret is created.
// +kubebuilder:validation:Enum=Owner;Orphan
type CreationPolicy string

const (
	CreationPolicyOwner CreationPolicy = "Owner"
	CreationPolicyOrphan CreationPolicy = "Orphan"
)

// RemoteRef points to a secret path in Akeyless.
type RemoteRef struct {
	Key      string `json:"key"`
	Property string `json:"property,omitempty"`
	Version  string `json:"version,omitempty"`
}

// SecretMapping maps an Akeyless item to a Kubernetes Secret key.
type SecretMapping struct {
	SecretKey string    `json:"secretKey"`
	RemoteRef RemoteRef `json:"remoteRef"`
}

// Target defines the Kubernetes Secret to create or update.
type Target struct {
	Name           string          `json:"name,omitempty"`
	CreationPolicy CreationPolicy  `json:"creationPolicy,omitempty"`
	Template       *SecretTemplate `json:"template,omitempty"`
}

// SecretTemplate defines optional metadata for the managed Secret.
type SecretTemplate struct {
	Type     corev1.SecretType  `json:"type,omitempty"`
	Metadata *TemplateMetadata `json:"metadata,omitempty"`
}

// TemplateMetadata defines labels and annotations on the target Secret.
type TemplateMetadata struct {
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
}

// AkeylessSecretSpec defines desired sync state.
type AkeylessSecretSpec struct {
	StoreRef SecretStoreRef `json:"storeRef"`
	Target   Target         `json:"target"`

	// +kubebuilder:default=OnRemoteChange
	SyncPolicy SyncPolicy `json:"syncPolicy,omitempty"`

	// +kubebuilder:default="5m0s"
	SyncInterval *metav1.Duration `json:"syncInterval,omitempty"`

	Data []SecretMapping `json:"data,omitempty"`

	// RolloutRestartTargets restart workloads when synced secret data changes.
	// +optional
	RolloutRestartTargets []RolloutRestartTarget `json:"rolloutRestartTargets,omitempty"`
}

// AkeylessSecretStatus defines observed sync state.
type AkeylessSecretStatus struct {
	RefreshTime      metav1.Time              `json:"refreshTime,omitempty"`
	SyncedGeneration int64                    `json:"syncedGeneration,omitempty"`
	RemoteVersions   map[string]int32         `json:"remoteVersions,omitempty"`
	Conditions       []AkeylessSecretCondition `json:"conditions,omitempty"`
}

// AkeylessSecretCondition reports sync readiness.
type AkeylessSecretCondition = SecretCondition

// AkeylessSecret syncs Akeyless items into a Kubernetes Secret.
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:storageversion
// +kubebuilder:metadata:labels="secrets.akeyless.io/component=controller"
// +kubebuilder:resource:scope=Namespaced,categories={akeyless-secrets},shortName=as
// +kubebuilder:printcolumn:name="Store",type=string,JSONPath=`.spec.storeRef.name`
// +kubebuilder:printcolumn:name="Sync Policy",type=string,JSONPath=`.spec.syncPolicy`
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
// +kubebuilder:printcolumn:name="Last Sync",type=date,JSONPath=`.status.refreshTime`
type AkeylessSecret struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AkeylessSecretSpec   `json:"spec,omitempty"`
	Status AkeylessSecretStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
type AkeylessSecretList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AkeylessSecret `json:"items"`
}
