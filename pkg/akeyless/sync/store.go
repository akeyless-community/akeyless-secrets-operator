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

// Package sync provides helpers shared by Akeyless sync controllers.
package sync

import (
	"context"
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/client"

	akeylessv1alpha1 "github.com/external-secrets/external-secrets/apis/akeyless/v1alpha1"
	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	"github.com/external-secrets/external-secrets/pkg/akeyless/adapter"
	akeylessstore "github.com/external-secrets/external-secrets/pkg/controllers/akeylessstore"
)

// EnsureStoreReady verifies the referenced store is ready for sync.
func EnsureStoreReady(ctx context.Context, c client.Client, namespace string, ref akeylessv1alpha1.SecretStoreRef) error {
	kind := ref.Kind
	if kind == "" {
		kind = akeylessv1alpha1.AkeylessSecretStoreKind
	}
	switch kind {
	case akeylessv1alpha1.AkeylessSecretStoreKind, "AkeylessSecretStore":
		_, err := akeylessstore.IsNamespacedStoreReady(ctx, c, namespace, ref.Name)
		return err
	case akeylessv1alpha1.ClusterAkeylessSecretStoreKind, "ClusterAkeylessSecretStore":
		_, err := akeylessstore.IsClusterStoreReady(ctx, c, ref.Name, namespace)
		return err
	default:
		return fmt.Errorf("unsupported store kind %q", kind)
	}
}

// ResolveStore loads the store CRD and adapts it for the Akeyless provider.
func ResolveStore(ctx context.Context, c client.Client, namespace string, ref akeylessv1alpha1.SecretStoreRef) (esv1.GenericStore, string, error) {
	kind := ref.Kind
	if kind == "" {
		kind = akeylessv1alpha1.AkeylessSecretStoreKind
	}

	switch kind {
	case akeylessv1alpha1.AkeylessSecretStoreKind, "AkeylessSecretStore":
		store := &akeylessv1alpha1.AkeylessSecretStore{}
		if err := c.Get(ctx, client.ObjectKey{Name: ref.Name, Namespace: namespace}, store); err != nil {
			return nil, "", fmt.Errorf("get AkeylessSecretStore %q: %w", ref.Name, err)
		}
		return adapter.SecretStoreFromNamespaced(store), esv1.SecretStoreKind, nil
	case akeylessv1alpha1.ClusterAkeylessSecretStoreKind, "ClusterAkeylessSecretStore":
		store := &akeylessv1alpha1.ClusterAkeylessSecretStore{}
		if err := c.Get(ctx, client.ObjectKey{Name: ref.Name}, store); err != nil {
			return nil, "", fmt.Errorf("get ClusterAkeylessSecretStore %q: %w", ref.Name, err)
		}
		return adapter.SecretStoreFromCluster(store), esv1.ClusterSecretStoreKind, nil
	default:
		return nil, "", fmt.Errorf("unsupported store kind %q", kind)
	}
}
