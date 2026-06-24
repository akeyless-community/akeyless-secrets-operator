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

// Package adapter converts Akeyless CRDs into types used by the Akeyless provider.
package adapter

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	akeylessv1alpha1 "github.com/external-secrets/external-secrets/apis/akeyless/v1alpha1"
	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
)

// SecretStoreFromNamespaced adapts an AkeylessSecretStore to a legacy SecretStore for the provider client.
func SecretStoreFromNamespaced(store *akeylessv1alpha1.AkeylessSecretStore) *esv1.SecretStore {
	return &esv1.SecretStore{
		TypeMeta: metav1.TypeMeta{
			APIVersion: esv1.SchemeGroupVersion.String(),
			Kind:       esv1.SecretStoreKind,
		},
		ObjectMeta: store.ObjectMeta,
		Spec: esv1.SecretStoreSpec{
			Provider: &esv1.SecretStoreProvider{
				Akeyless: providerFromSpec(store.Spec),
			},
		},
	}
}

// SecretStoreFromCluster adapts a ClusterAkeylessSecretStore to a legacy ClusterSecretStore.
func SecretStoreFromCluster(store *akeylessv1alpha1.ClusterAkeylessSecretStore) *esv1.ClusterSecretStore {
	return &esv1.ClusterSecretStore{
		TypeMeta: metav1.TypeMeta{
			APIVersion: esv1.SchemeGroupVersion.String(),
			Kind:       esv1.ClusterSecretStoreKind,
		},
		ObjectMeta: store.ObjectMeta,
		Spec: esv1.SecretStoreSpec{
			Provider: &esv1.SecretStoreProvider{
				Akeyless: providerFromSpec(store.Spec.AkeylessSecretStoreSpec),
			},
		},
	}
}

func providerFromSpec(spec akeylessv1alpha1.AkeylessSecretStoreSpec) *esv1.AkeylessProvider {
	url := spec.AkeylessGWApiURL
	provider := &esv1.AkeylessProvider{
		AkeylessGWApiURL: &url,
		IgnoreCache:      spec.IgnoreCache,
		CABundle:         spec.CABundle,
		Auth:             authFromSpec(spec.Auth),
	}
	if spec.CAProvider != nil {
		provider.CAProvider = &esv1.CAProvider{
			Type:      esv1.CAProviderType(spec.CAProvider.Type),
			Name:      spec.CAProvider.Name,
			Key:       spec.CAProvider.Key,
			Namespace: spec.CAProvider.Namespace,
		}
	}
	return provider
}

func authFromSpec(auth akeylessv1alpha1.AkeylessAuth) *esv1.AkeylessAuth {
	out := &esv1.AkeylessAuth{
		ServiceAccountRef: auth.ServiceAccountRef,
	}
	if auth.SecretRef != nil {
		out.SecretRef = esv1.AkeylessAuthSecretRef{
			AccessID:        auth.SecretRef.AccessID,
			AccessType:      auth.SecretRef.AccessType,
			AccessTypeParam: auth.SecretRef.AccessTypeParam,
		}
	}
	if auth.KubernetesAuth != nil {
		out.KubernetesAuth = &esv1.AkeylessKubernetesAuth{
			AccessID:          auth.KubernetesAuth.AccessID,
			K8sConfName:       auth.KubernetesAuth.K8sConfName,
			ServiceAccountRef: auth.KubernetesAuth.ServiceAccountRef,
			SecretRef:         auth.KubernetesAuth.SecretRef,
		}
	}
	return out
}

// RemoteRef converts a mapping remote ref to the provider API type.
func RemoteRef(ref akeylessv1alpha1.RemoteRef) esv1.ExternalSecretDataRemoteRef {
	return esv1.ExternalSecretDataRemoteRef{
		Key:      ref.Key,
		Property: ref.Property,
		Version:  ref.Version,
	}
}
