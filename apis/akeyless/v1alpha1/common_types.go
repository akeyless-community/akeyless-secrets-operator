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

// SyncPolicy defines when the operator syncs from Akeyless into Kubernetes.
// +kubebuilder:validation:Enum=CreatedOnce;Periodic;OnSpecChange;OnRemoteChange;OnWebhook
type SyncPolicy string

const (
	SyncPolicyCreatedOnce    SyncPolicy = "CreatedOnce"
	SyncPolicyPeriodic       SyncPolicy = "Periodic"
	SyncPolicyOnSpecChange   SyncPolicy = "OnSpecChange"
	SyncPolicyOnRemoteChange SyncPolicy = "OnRemoteChange"
	// SyncPolicyOnWebhook syncs when an Akeyless event webhook triggers a force-sync,
	// with syncInterval as fallback polling when webhooks are unavailable.
	SyncPolicyOnWebhook SyncPolicy = "OnWebhook"
)

// RolloutRestartTarget triggers a rolling restart when synced secret data changes.
type RolloutRestartTarget struct {
	// +kubebuilder:validation:Enum=Deployment;StatefulSet;DaemonSet
	Kind string `json:"kind"`
	Name string `json:"name"`
}

// SecretTarget defines the Kubernetes Secret destination.
type SecretTarget struct {
	Name           string         `json:"name,omitempty"`
	CreationPolicy CreationPolicy `json:"creationPolicy,omitempty"`
	Template       *SecretTemplate `json:"template,omitempty"`
}

// SyncSpec controls refresh behavior shared by AkeylessSecret and AkeylessDynamicSecret.
type SyncSpec struct {
	SyncPolicy   SyncPolicy       `json:"syncPolicy,omitempty"`
	SyncInterval *metav1.Duration `json:"syncInterval,omitempty"`
}

// SecretStatus is shared status for sync resources.
type SecretStatus struct {
	RefreshTime      metav1.Time       `json:"refreshTime,omitempty"`
	SyncedGeneration int64             `json:"syncedGeneration,omitempty"`
	RemoteVersions   map[string]int32  `json:"remoteVersions,omitempty"`
	Conditions       []SecretCondition `json:"conditions,omitempty"`
}

// SecretCondition reports sync readiness.
type SecretCondition struct {
	Type               string                 `json:"type"`
	Status             corev1.ConditionStatus `json:"status"`
	Reason             string                 `json:"reason,omitempty"`
	Message            string                 `json:"message,omitempty"`
	LastTransitionTime metav1.Time            `json:"lastTransitionTime,omitempty"`
}

const (
	SecretConditionReady = "Ready"

	ReasonSecretSynced      = "SecretSynced"
	ReasonSecretSyncedError = "SecretSyncedError"

	AnnotationDataHash  = "secrets.akeyless.io/data-hash"
	AnnotationForceSync = "secrets.akeyless.io/force-sync"
	AnnotationRestarted = "secrets.akeyless.io/restartedAt"

	LabelManaged      = "secrets.akeyless.io/managed"
	LabelManagedValue = "true"
)
