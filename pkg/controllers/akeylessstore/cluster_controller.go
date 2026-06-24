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

package akeylessstore

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"

	akeylessv1alpha1 "github.com/external-secrets/external-secrets/apis/akeyless/v1alpha1"
	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	"github.com/external-secrets/external-secrets/pkg/akeyless/adapter"
	akeylessprovider "github.com/external-secrets/external-secrets/providers/v1/akeyless"
)

// ClusterStoreReconciler validates ClusterAkeylessSecretStore resources.
type ClusterStoreReconciler struct {
	client.Client
	Log             logr.Logger
	Scheme          *runtime.Scheme
	Recorder        record.EventRecorder
	Provider        esv1.Provider
	RequeueInterval time.Duration
}

// SetupWithManager registers the cluster store controller.
func (r *ClusterStoreReconciler) SetupWithManager(mgr ctrl.Manager, opts controller.Options) error {
	if r.Provider == nil {
		r.Provider = akeylessprovider.NewProvider()
	}
	if r.Recorder == nil {
		r.Recorder = mgr.GetEventRecorderFor("akeyless-cluster-store-controller")
	}
	if r.RequeueInterval == 0 {
		r.RequeueInterval = 5 * time.Minute
	}
	return ctrl.NewControllerManagedBy(mgr).
		WithOptions(opts).
		For(&akeylessv1alpha1.ClusterAkeylessSecretStore{}).
		Complete(r)
}

// Reconcile validates a cluster-scoped store.
func (r *ClusterStoreReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("clusterakeylesssecretstore", req.Name)

	store := &akeylessv1alpha1.ClusterAkeylessSecretStore{}
	if err := r.Get(ctx, client.ObjectKey{Name: req.Name}, store); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	patch := client.MergeFrom(store.DeepCopy())
	err := validateClusterStore(ctx, r.Client, r.Provider, store)
	clusterSetStoreCondition(store, err)
	if patchErr := r.Status().Patch(ctx, store, patch); patchErr != nil {
		log.Error(patchErr, "failed to patch status")
	}

	if err != nil {
		r.Recorder.Event(store, corev1.EventTypeWarning, reasonStoreInvalid, err.Error())
		return ctrl.Result{RequeueAfter: r.RequeueInterval}, nil
	}

	r.Recorder.Event(store, corev1.EventTypeNormal, reasonStoreValid, msgStoreValid)
	return ctrl.Result{RequeueAfter: r.RequeueInterval}, nil
}

func validateClusterStore(ctx context.Context, kube client.Client, provider esv1.Provider, store *akeylessv1alpha1.ClusterAkeylessSecretStore) error {
	generic := adapter.SecretStoreFromCluster(store)
	if warn, err := provider.ValidateStore(generic); err != nil {
		return fmt.Errorf("invalid store spec: %w", err)
	} else {
		_ = warn
	}

	namespace, err := validationNamespace(ctx, kube, store)
	if err != nil {
		return err
	}

	secretsClient, err := provider.NewClient(ctx, generic, kube, namespace)
	if err != nil {
		return fmt.Errorf("create client: %w", err)
	}
	defer func() { _ = secretsClient.Close(ctx) }()

	result, err := secretsClient.Validate()
	if err != nil && result != esv1.ValidationResultUnknown {
		return fmt.Errorf("validate connectivity: %w", err)
	}
	return nil
}

func validationNamespace(ctx context.Context, kube client.Client, store *akeylessv1alpha1.ClusterAkeylessSecretStore) (string, error) {
	for _, cond := range store.Spec.Conditions {
		if len(cond.Namespaces) > 0 {
			return cond.Namespaces[0], nil
		}
		if cond.NamespaceSelector != nil {
			var nsList corev1.NamespaceList
			if err := kube.List(ctx, &nsList); err != nil {
				return "", err
			}
			selector, err := metav1.LabelSelectorAsSelector(cond.NamespaceSelector)
			if err != nil {
				return "", err
			}
			for _, ns := range nsList.Items {
				if selector.Matches(labels.Set(ns.Labels)) {
					return ns.Name, nil
				}
			}
		}
	}

	var nsList corev1.NamespaceList
	if err := kube.List(ctx, &nsList); err != nil {
		return "", err
	}
	if len(nsList.Items) == 0 {
		return "", fmt.Errorf("no namespaces available for cluster store validation")
	}
	return nsList.Items[0].Name, nil
}

func clusterSetStoreCondition(store *akeylessv1alpha1.ClusterAkeylessSecretStore, validateErr error) {
	now := metav1.Now()
	if validateErr != nil {
		store.Status.Conditions = []akeylessv1alpha1.AkeylessSecretStoreCondition{{
			Type:               akeylessv1alpha1.StoreConditionReady,
			Status:             corev1.ConditionFalse,
			Reason:             reasonStoreInvalid,
			Message:            validateErr.Error(),
			LastTransitionTime: now,
		}}
		return
	}
	store.Status.Conditions = []akeylessv1alpha1.AkeylessSecretStoreCondition{{
		Type:               akeylessv1alpha1.StoreConditionReady,
		Status:             corev1.ConditionTrue,
		Reason:             reasonStoreValid,
		Message:            msgStoreValid,
		LastTransitionTime: now,
	}}
}
