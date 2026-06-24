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

// Package akeylessstore validates AkeylessSecretStore connectivity.
package akeylessstore

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
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

	_ "github.com/external-secrets/external-secrets/pkg/register"
)

const (
	reasonStoreValid   = "Valid"
	reasonStoreInvalid = "Invalid"
	msgStoreValid      = "Akeyless store validated"
)

// StoreReconciler validates namespaced AkeylessSecretStore resources.
type StoreReconciler struct {
	client.Client
	Log             logr.Logger
	Scheme          *runtime.Scheme
	Recorder        record.EventRecorder
	Provider        esv1.Provider
	RequeueInterval time.Duration
}

// SetupWithManager registers the controller.
func (r *StoreReconciler) SetupWithManager(mgr ctrl.Manager, opts controller.Options) error {
	if r.Provider == nil {
		r.Provider = akeylessprovider.NewProvider()
	}
	if r.Recorder == nil {
		r.Recorder = mgr.GetEventRecorderFor("akeyless-store-controller")
	}
	if r.RequeueInterval == 0 {
		r.RequeueInterval = 5 * time.Minute
	}
	return ctrl.NewControllerManagedBy(mgr).
		WithOptions(opts).
		For(&akeylessv1alpha1.AkeylessSecretStore{}).
		Complete(r)
}

// Reconcile validates store configuration against Akeyless.
func (r *StoreReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("akeylesssecretstore", req.NamespacedName)

	store := &akeylessv1alpha1.AkeylessSecretStore{}
	if err := r.Get(ctx, req.NamespacedName, store); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	patch := client.MergeFrom(store.DeepCopy())
	err := validateNamespacedStore(ctx, r.Client, r.Provider, store, req.Namespace)
	setStoreCondition(store, err)
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

func validateNamespacedStore(ctx context.Context, kube client.Client, provider esv1.Provider, store *akeylessv1alpha1.AkeylessSecretStore, namespace string) error {
	generic := adapter.SecretStoreFromNamespaced(store)
	if warn, err := provider.ValidateStore(generic); err != nil {
		return fmt.Errorf("invalid store spec: %w", err)
	} else {
		_ = warn
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

func setStoreCondition(store *akeylessv1alpha1.AkeylessSecretStore, validateErr error) {
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

func storeIsReady(conditions []akeylessv1alpha1.AkeylessSecretStoreCondition) bool {
	for _, c := range conditions {
		if c.Type == akeylessv1alpha1.StoreConditionReady && c.Status == corev1.ConditionTrue {
			return true
		}
	}
	return false
}

// IsNamespacedStoreReady returns true when the referenced namespaced store exists and is Ready.
func IsNamespacedStoreReady(ctx context.Context, c client.Client, namespace, name string) (bool, error) {
	store := &akeylessv1alpha1.AkeylessSecretStore{}
	if err := c.Get(ctx, client.ObjectKey{Name: name, Namespace: namespace}, store); err != nil {
		if apierrors.IsNotFound(err) {
			return false, fmt.Errorf("AkeylessSecretStore %q not found", name)
		}
		return false, err
	}
	if !storeIsReady(store.Status.Conditions) {
		return false, fmt.Errorf("AkeylessSecretStore %q is not ready", name)
	}
	return true, nil
}

// IsClusterStoreReady returns true when the cluster store exists, is Ready, and allows the namespace.
func IsClusterStoreReady(ctx context.Context, c client.Client, name, namespace string) (bool, error) {
	store := &akeylessv1alpha1.ClusterAkeylessSecretStore{}
	if err := c.Get(ctx, client.ObjectKey{Name: name}, store); err != nil {
		if apierrors.IsNotFound(err) {
			return false, fmt.Errorf("ClusterAkeylessSecretStore %q not found", name)
		}
		return false, err
	}
	if !clusterStoreIsReady(store.Status.Conditions) {
		return false, fmt.Errorf("ClusterAkeylessSecretStore %q is not ready", name)
	}
	allowed, err := clusterStoreAllowed(ctx, c, store, namespace)
	if err != nil {
		return false, err
	}
	if !allowed {
		return false, fmt.Errorf("namespace %q is not allowed by ClusterAkeylessSecretStore %q", namespace, name)
	}
	return true, nil
}

func clusterStoreIsReady(conditions []akeylessv1alpha1.AkeylessSecretStoreCondition) bool {
	return storeIsReady(conditions)
}

func clusterStoreAllowed(ctx context.Context, c client.Client, store *akeylessv1alpha1.ClusterAkeylessSecretStore, namespace string) (bool, error) {
	if len(store.Spec.Conditions) == 0 {
		return true, nil
	}
	ns := &corev1.Namespace{}
	if err := c.Get(ctx, client.ObjectKey{Name: namespace}, ns); err != nil {
		return false, err
	}
	for _, cond := range store.Spec.Conditions {
		for _, name := range cond.Namespaces {
			if name == namespace {
				return true, nil
			}
		}
		if cond.NamespaceSelector != nil {
			selector, err := metav1.LabelSelectorAsSelector(cond.NamespaceSelector)
			if err == nil && selector.Matches(labels.Set(ns.Labels)) {
				return true, nil
			}
		}
	}
	return false, nil
}
