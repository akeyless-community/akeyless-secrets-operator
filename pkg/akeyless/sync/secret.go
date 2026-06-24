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

package sync

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	akeylessv1alpha1 "github.com/external-secrets/external-secrets/apis/akeyless/v1alpha1"
	"github.com/external-secrets/external-secrets/runtime/esutils"
)

// ApplyManagedSecret creates or updates a Kubernetes Secret owned by owner.
func ApplyManagedSecret(
	ctx context.Context,
	c client.Client,
	scheme *runtime.Scheme,
	owner client.Object,
	namespace, name string,
	target akeylessv1alpha1.Target,
	data map[string][]byte,
) error {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				akeylessv1alpha1.LabelManaged: akeylessv1alpha1.LabelManagedValue,
			},
			Annotations: map[string]string{
				akeylessv1alpha1.AnnotationDataHash: esutils.ObjectHash(data),
			},
		},
		Type: corev1.SecretTypeOpaque,
		Data: data,
	}

	if target.Template != nil {
		if target.Template.Type != "" {
			secret.Type = target.Template.Type
		}
		if target.Template.Metadata != nil {
			if target.Template.Metadata.Labels != nil {
				for k, v := range target.Template.Metadata.Labels {
					secret.Labels[k] = v
				}
			}
			if target.Template.Metadata.Annotations != nil {
				for k, v := range target.Template.Metadata.Annotations {
					secret.Annotations[k] = v
				}
			}
		}
	}

	existing := &corev1.Secret{}
	err := c.Get(ctx, client.ObjectKeyFromObject(secret), existing)
	if apierrors.IsNotFound(err) {
		if target.CreationPolicy == akeylessv1alpha1.CreationPolicyOwner || target.CreationPolicy == "" {
			if err := controllerutil.SetControllerReference(owner, secret, scheme); err != nil {
				return err
			}
		}
		return c.Create(ctx, secret)
	}
	if err != nil {
		return err
	}

	existing.Data = data
	if existing.Annotations == nil {
		existing.Annotations = map[string]string{}
	}
	existing.Annotations[akeylessv1alpha1.AnnotationDataHash] = esutils.ObjectHash(data)
	if existing.Labels == nil {
		existing.Labels = map[string]string{}
	}
	existing.Labels[akeylessv1alpha1.LabelManaged] = akeylessv1alpha1.LabelManagedValue
	return c.Update(ctx, existing)
}

// IsManagedSecretValid checks the secret is managed and hash-consistent.
func IsManagedSecretValid(secret *corev1.Secret) bool {
	if secret == nil {
		return false
	}
	if secret.Labels[akeylessv1alpha1.LabelManaged] != akeylessv1alpha1.LabelManagedValue {
		return false
	}
	return secret.Annotations[akeylessv1alpha1.AnnotationDataHash] == esutils.ObjectHash(secret.Data)
}
