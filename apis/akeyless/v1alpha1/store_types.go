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
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// AkeylessSecretStoreSpec defines how the operator connects to Akeyless.
type AkeylessSecretStoreSpec struct {
	// AkeylessGWApiURL is the Akeyless Gateway or SaaS API URL.
	// Example Gateway: https://your.akeyless.gw:8080/v2
	// Example SaaS: https://api.akeyless.io
	AkeylessGWApiURL string `json:"akeylessGWApiURL"`

	// IgnoreCache bypasses the Gateway cache for secret reads when true.
	// Only relevant when akeylessGWApiURL points to an Akeyless Gateway.
	// +optional
	IgnoreCache *bool `json:"ignoreCache,omitempty"`

	// Auth configures how the operator authenticates with Akeyless.
	Auth AkeylessAuth `json:"auth"`

	// PEM/base64 encoded CA bundle used to validate the Gateway certificate.
	// +optional
	CABundle []byte `json:"caBundle,omitempty"`

	// CAProvider loads a CA bundle from a Secret or ConfigMap.
	// +optional
	CAProvider *CAProvider `json:"caProvider,omitempty"`
}

// AkeylessAuth configures authentication with Akeyless.
type AkeylessAuth struct {
	// SecretRef references credentials in a Kubernetes Secret (accessId, accessType, accessTypeParam).
	// +optional
	SecretRef *AkeylessAuthSecretRef `json:"secretRef,omitempty"`

	// KubernetesAuth uses a Kubernetes auth method configured on the Gateway.
	// +optional
	KubernetesAuth *AkeylessKubernetesAuth `json:"kubernetesAuth,omitempty"`

	// ServiceAccountRef obtains a federated token from a ServiceAccount (e.g. Azure Workload Identity).
	// +optional
	ServiceAccountRef *esmeta.ServiceAccountSelector `json:"serviceAccountRef,omitempty"`
}

// AkeylessAuthSecretRef references credential keys in a Secret.
type AkeylessAuthSecretRef struct {
	AccessID        esmeta.SecretKeySelector `json:"accessID,omitempty"`
	AccessType      esmeta.SecretKeySelector `json:"accessType,omitempty"`
	AccessTypeParam esmeta.SecretKeySelector `json:"accessTypeParam,omitempty"`
}

// AkeylessKubernetesAuth configures Kubernetes authentication with Akeyless.
type AkeylessKubernetesAuth struct {
	AccessID          string                         `json:"accessID"`
	K8sConfName       string                         `json:"k8sConfName"`
	ServiceAccountRef *esmeta.ServiceAccountSelector `json:"serviceAccountRef,omitempty"`
	SecretRef         *esmeta.SecretKeySelector      `json:"secretRef,omitempty"`
}

// CAProviderType defines where a CA bundle is loaded from.
// +kubebuilder:validation:Enum=Secret;ConfigMap
type CAProviderType string

const (
	CAProviderTypeSecret    CAProviderType = "Secret"
	CAProviderTypeConfigMap CAProviderType = "ConfigMap"
)

// CAProvider references a CA bundle in a Secret or ConfigMap.
type CAProvider struct {
	Type      CAProviderType `json:"type"`
	Name      string         `json:"name"`
	Key       string         `json:"key,omitempty"`
	Namespace *string        `json:"namespace,omitempty"`
}

// AkeylessSecretStoreStatus defines the observed state of AkeylessSecretStore.
type AkeylessSecretStoreStatus struct {
	// +optional
	Conditions []AkeylessSecretStoreCondition `json:"conditions,omitempty"`
}

// AkeylessSecretStoreCondition describes store readiness.
type AkeylessSecretStoreCondition struct {
	Type               string             `json:"type"`
	Status             corev1.ConditionStatus `json:"status"`
	Reason             string             `json:"reason,omitempty"`
	Message            string             `json:"message,omitempty"`
	LastTransitionTime metav1.Time        `json:"lastTransitionTime,omitempty"`
}

const (
	StoreConditionReady = "Ready"
)

// AkeylessSecretStore connects to Akeyless from a namespace.
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:storageversion
// +kubebuilder:metadata:labels="secrets.akeyless.io/component=controller"
// +kubebuilder:resource:scope=Namespaced,categories={akeyless-secrets},shortName=ass
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
type AkeylessSecretStore struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AkeylessSecretStoreSpec   `json:"spec,omitempty"`
	Status AkeylessSecretStoreStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
type AkeylessSecretStoreList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AkeylessSecretStore `json:"items"`
}

// ClusterAkeylessSecretStoreSpec defines a cluster-scoped Akeyless connection.
type ClusterAkeylessSecretStoreSpec struct {
	AkeylessSecretStoreSpec `json:",inline"`

	// Conditions restrict which namespaces may use this cluster store.
	// +optional
	Conditions []ClusterStoreNamespaceCondition `json:"conditions,omitempty"`
}

// ClusterStoreNamespaceCondition selects namespaces allowed to use the cluster store.
type ClusterStoreNamespaceCondition struct {
	// +optional
	NamespaceSelector *metav1.LabelSelector `json:"namespaceSelector,omitempty"`
	// +optional
	Namespaces []string `json:"namespaces,omitempty"`
}

// ClusterAkeylessSecretStoreStatus defines observed cluster store state.
type ClusterAkeylessSecretStoreStatus struct {
	// +optional
	Conditions []AkeylessSecretStoreCondition `json:"conditions,omitempty"`
}

// ClusterAkeylessSecretStore is a cluster-scoped Akeyless connection.
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:storageversion
// +kubebuilder:metadata:labels="secrets.akeyless.io/component=controller"
// +kubebuilder:resource:scope=Cluster,categories={akeyless-secrets},shortName=cass
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
type ClusterAkeylessSecretStore struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ClusterAkeylessSecretStoreSpec   `json:"spec,omitempty"`
	Status ClusterAkeylessSecretStoreStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
type ClusterAkeylessSecretStoreList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ClusterAkeylessSecretStore `json:"items"`
}
