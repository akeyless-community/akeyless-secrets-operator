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

package adapter

import (
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	akeylessv1alpha1 "github.com/external-secrets/external-secrets/apis/akeyless/v1alpha1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
)

func TestSecretStoreFromNamespacedJWTAuth(t *testing.T) {
	expiration := int64(900)
	saRef := &esmeta.ServiceAccountSelector{Name: "jwt-sa", Audiences: []string{"akeyless.io"}}
	secretRef := &esmeta.SecretKeySelector{Name: "akeyless-jwt", Key: "token"}

	tests := []struct {
		name string
		jwt  *akeylessv1alpha1.AkeylessJWTAuth
	}{
		{
			name: "secretRef",
			jwt:  &akeylessv1alpha1.AkeylessJWTAuth{AccessID: "p-jwt", SecretRef: secretRef},
		},
		{
			name: "serviceAccountRef with expiration",
			jwt: &akeylessv1alpha1.AkeylessJWTAuth{
				AccessID:          "p-jwt",
				ServiceAccountRef: saRef,
				ExpirationSeconds: &expiration,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &akeylessv1alpha1.AkeylessSecretStore{
				ObjectMeta: metav1.ObjectMeta{Name: "s", Namespace: "ns"},
				Spec: akeylessv1alpha1.AkeylessSecretStoreSpec{
					AkeylessGWApiURL: "https://api.akeyless.io",
					Auth:             akeylessv1alpha1.AkeylessAuth{JWTAuth: tt.jwt},
				},
			}

			got := SecretStoreFromNamespaced(store).Spec.Provider.Akeyless.Auth.JWTAuth
			if got == nil {
				t.Fatal("jwtAuth was not mapped to the provider spec")
			}
			if got.AccessID != tt.jwt.AccessID {
				t.Errorf("accessID = %q, want %q", got.AccessID, tt.jwt.AccessID)
			}
			if got.SecretRef != tt.jwt.SecretRef {
				t.Errorf("secretRef = %v, want %v", got.SecretRef, tt.jwt.SecretRef)
			}
			if got.ServiceAccountRef != tt.jwt.ServiceAccountRef {
				t.Errorf("serviceAccountRef = %v, want %v", got.ServiceAccountRef, tt.jwt.ServiceAccountRef)
			}
			if got.ExpirationSeconds != tt.jwt.ExpirationSeconds {
				t.Errorf("expirationSeconds = %v, want %v", got.ExpirationSeconds, tt.jwt.ExpirationSeconds)
			}
		})
	}
}

func TestSecretStoreFromNamespacedWithoutJWTAuth(t *testing.T) {
	store := &akeylessv1alpha1.AkeylessSecretStore{
		Spec: akeylessv1alpha1.AkeylessSecretStoreSpec{
			AkeylessGWApiURL: "https://api.akeyless.io",
			Auth: akeylessv1alpha1.AkeylessAuth{
				KubernetesAuth: &akeylessv1alpha1.AkeylessKubernetesAuth{AccessID: "p-k8s", K8sConfName: "conf"},
			},
		},
	}
	got := SecretStoreFromNamespaced(store).Spec.Provider.Akeyless.Auth
	if got.JWTAuth != nil {
		t.Errorf("jwtAuth should be nil when not configured, got %+v", got.JWTAuth)
	}
	if got.KubernetesAuth == nil || got.KubernetesAuth.AccessID != "p-k8s" {
		t.Errorf("kubernetesAuth mapping regressed: %+v", got.KubernetesAuth)
	}
}
