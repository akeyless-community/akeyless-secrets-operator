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

	// JWTAuth authenticates with an Akeyless OAuth2.0/JWT auth method by submitting
	// access-type=jwt and a bearer JWT. When set, SecretRef and ServiceAccountRef on
	// AkeylessAuth are ignored.
	// +optional
	JWTAuth *AkeylessJWTAuth `json:"jwtAuth,omitempty"`

	// ServiceAccountRef obtains a federated token from a ServiceAccount (e.g. Azure Workload Identity).
	// Ignored for other access types.
	// +optional
	ServiceAccountRef *esmeta.ServiceAccountSelector `json:"serviceAccountRef,omitempty"`
}

// AkeylessJWTAuth authenticates with Akeyless using an OAuth2.0/JWT auth method
// (API access-type=jwt). Provide either a Secret holding the JWT or a ServiceAccount
// for TokenRequest-issued tokens. Using ServiceAccountRef requires create on
// serviceaccounts/token (Helm: rbac.serviceAccountTokenCreate, default true).
// +kubebuilder:validation:XValidation:rule="has(self.secretRef) != has(self.serviceAccountRef)",message="exactly one of secretRef or serviceAccountRef must be set"
// +kubebuilder:validation:XValidation:rule="!has(self.secretRef) || (has(self.secretRef.name) && size(self.secretRef.name) > 0)",message="secretRef.name must be nonempty"
// +kubebuilder:validation:XValidation:rule="!has(self.serviceAccountRef) || (has(self.serviceAccountRef.name) && size(self.serviceAccountRef.name) > 0)",message="serviceAccountRef.name must be nonempty"
// +kubebuilder:validation:XValidation:rule="!has(self.serviceAccountRef) || (has(self.serviceAccountRef.audiences) && size(self.serviceAccountRef.audiences) > 0)",message="serviceAccountRef.audiences must be set to the Akeyless auth method audience"
// +kubebuilder:validation:XValidation:rule="!has(self.expirationSeconds) || has(self.serviceAccountRef)",message="expirationSeconds only applies to serviceAccountRef"
type AkeylessJWTAuth struct {
	// AccessID is the Akeyless OAuth2.0/JWT auth-method access-id.
	// +kubebuilder:validation:MinLength=1
	AccessID string `json:"accessID"`

	// SecretRef refers to a key in a Secret containing the JWT used to authenticate with Akeyless.
	// +optional
	SecretRef *esmeta.SecretKeySelector `json:"secretRef,omitempty"`

	// ServiceAccountRef requests a token via the TokenRequest API. audiences is required and
	// must match the Akeyless OAuth2.0/JWT auth-method audience. Mutually exclusive with SecretRef.
	// +optional
	ServiceAccountRef *esmeta.ServiceAccountSelector `json:"serviceAccountRef,omitempty"`

	// ExpirationSeconds is the TokenRequest token lifetime. Only valid with ServiceAccountRef.
	// Defaults to 600. Kubernetes rejects values below 600.
	// +optional
	// +kubebuilder:validation:Minimum=600
	ExpirationSeconds *int64 `json:"expirationSeconds,omitempty"`
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
	Type               string                 `json:"type"`
	Status             corev1.ConditionStatus `json:"status"`
	Reason             string                 `json:"reason,omitempty"`
	Message            string                 `json:"message,omitempty"`
	LastTransitionTime metav1.Time            `json:"lastTransitionTime,omitempty"`
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
