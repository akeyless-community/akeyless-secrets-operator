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

// Package akeylesssecret reconciles AkeylessSecret resources into Kubernetes Secrets.
package akeylesssecret

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	akeylessv1alpha1 "github.com/external-secrets/external-secrets/apis/akeyless/v1alpha1"
	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	"github.com/external-secrets/external-secrets/pkg/akeyless/adapter"
	akeylesssync "github.com/external-secrets/external-secrets/pkg/akeyless/sync"
	akeylessprovider "github.com/external-secrets/external-secrets/providers/v1/akeyless"
	"github.com/external-secrets/external-secrets/runtime/esutils"

	_ "github.com/external-secrets/external-secrets/pkg/register"
)

const finalizer = "secrets.akeyless.io/finalizer"

// Reconciler syncs AkeylessSecret resources.
type Reconciler struct {
	client.Client
	Log      logr.Logger
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
	Provider esv1.Provider
}

// SetupWithManager registers the controller.
func (r *Reconciler) SetupWithManager(mgr ctrl.Manager, opts controller.Options) error {
	if r.Provider == nil {
		r.Provider = akeylessprovider.NewProvider()
	}
	if r.Recorder == nil {
		r.Recorder = mgr.GetEventRecorderFor("akeyless-secrets-controller")
	}
	return ctrl.NewControllerManagedBy(mgr).
		WithOptions(opts).
		For(&akeylessv1alpha1.AkeylessSecret{}, builder.WithPredicates(predicate.Or(
			predicate.GenerationChangedPredicate{},
			forceSyncChangedPredicate{},
		))).
		Complete(r)
}

type forceSyncChangedPredicate struct{}

func (forceSyncChangedPredicate) Update(e event.UpdateEvent) bool {
	if e.ObjectOld == nil || e.ObjectNew == nil {
		return true
	}
	oldAnn := e.ObjectOld.GetAnnotations()
	newAnn := e.ObjectNew.GetAnnotations()
	return oldAnn[akeylessv1alpha1.AnnotationForceSync] != newAnn[akeylessv1alpha1.AnnotationForceSync]
}

func (forceSyncChangedPredicate) Create(_ event.CreateEvent) bool  { return true }
func (forceSyncChangedPredicate) Delete(_ event.DeleteEvent) bool  { return true }
func (forceSyncChangedPredicate) Generic(_ event.GenericEvent) bool { return true }

// Reconcile fetches secrets from Akeyless and updates the target Kubernetes Secret.
func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("akeylesssecret", req.NamespacedName)

	as := &akeylessv1alpha1.AkeylessSecret{}
	if err := r.Get(ctx, req.NamespacedName, as); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if !as.DeletionTimestamp.IsZero() {
		return ctrl.Result{}, nil
	}

	if controllerutil.AddFinalizer(as, finalizer) {
		if err := r.Update(ctx, as); err != nil {
			return ctrl.Result{}, err
		}
	}

	targetName := as.Spec.Target.Name
	if targetName == "" {
		targetName = as.Name
	}

	existing := &corev1.Secret{}
	err := r.Get(ctx, client.ObjectKey{Name: targetName, Namespace: as.Namespace}, existing)
	secretExists := err == nil
	if err != nil && !apierrors.IsNotFound(err) {
		return ctrl.Result{}, err
	}

	oldHash := ""
	if secretExists {
		oldHash = existing.Annotations[akeylessv1alpha1.AnnotationDataHash]
	}

	if err := akeylesssync.EnsureStoreReady(ctx, r.Client, as.Namespace, as.Spec.StoreRef); err != nil {
		r.markFailed(as, err)
		_ = r.Status().Update(ctx, as)
		return ctrl.Result{RequeueAfter: time.Minute}, nil
	}

	policy := as.Spec.SyncPolicy
	if policy == "" {
		policy = akeylessv1alpha1.SyncPolicyOnRemoteChange
	}

	if !shouldSync(as) && secretExists && akeylesssync.IsManagedSecretValid(existing) {
		if policy == akeylessv1alpha1.SyncPolicyOnRemoteChange || policy == akeylessv1alpha1.SyncPolicyOnWebhook {
			changed, versions, checkErr := r.checkRemoteVersions(ctx, as)
			if checkErr != nil {
				r.markFailed(as, checkErr)
				_ = r.Status().Update(ctx, as)
				return ctrl.Result{}, checkErr
			}
			if !changed {
				return ctrl.Result{RequeueAfter: requeueAfter(as)}, nil
			}
			if as.Status.RemoteVersions == nil {
				as.Status.RemoteVersions = map[string]int32{}
			}
			for k, v := range versions {
				as.Status.RemoteVersions[k] = v
			}
		} else {
			return ctrl.Result{RequeueAfter: requeueAfter(as)}, nil
		}
	}

	store, _, err := akeylesssync.ResolveStore(ctx, r.Client, as.Namespace, as.Spec.StoreRef)
	if err != nil {
		r.markFailed(as, err)
		_ = r.Status().Update(ctx, as)
		return ctrl.Result{}, err
	}

	secretsClient, err := r.Provider.NewClient(ctx, store, r.Client, as.Namespace)
	if err != nil {
		r.markFailed(as, err)
		_ = r.Status().Update(ctx, as)
		return ctrl.Result{}, err
	}
	defer func() { _ = secretsClient.Close(ctx) }()

	secretData := make(map[string][]byte, len(as.Spec.Data))
	remoteVersions := map[string]int32{}
	versionClient, _ := secretsClient.(akeylessprovider.RemoteVersionClient)

	for _, mapping := range as.Spec.Data {
		ref := adapter.RemoteRef(mapping.RemoteRef)
		val, getErr := secretsClient.GetSecret(ctx, ref)
		if getErr != nil {
			r.markFailed(as, fmt.Errorf("fetch %q: %w", mapping.RemoteRef.Key, getErr))
			_ = r.Status().Update(ctx, as)
			return ctrl.Result{}, getErr
		}
		secretData[mapping.SecretKey] = val

		if versionClient != nil {
			version, verErr := versionClient.GetRemoteItemVersion(ctx, mapping.RemoteRef.Key)
			if verErr != nil {
				log.V(1).Info("could not read remote version", "key", mapping.RemoteRef.Key, "error", verErr)
			} else {
				remoteVersions[mapping.RemoteRef.Key] = version
			}
		}
	}

	if err := akeylesssync.ApplyManagedSecret(ctx, r.Client, r.Scheme, as, as.Namespace, targetName, as.Spec.Target, secretData); err != nil {
		r.markFailed(as, err)
		_ = r.Status().Update(ctx, as)
		return ctrl.Result{}, err
	}

	newHash := esutils.ObjectHash(secretData)
	if len(as.Spec.RolloutRestartTargets) > 0 && oldHash != newHash {
		if restartErr := akeylesssync.TriggerRolloutRestarts(ctx, r.Client, as.Namespace, as.Spec.RolloutRestartTargets); restartErr != nil {
			r.markFailed(as, restartErr)
			_ = r.Status().Update(ctx, as)
			return ctrl.Result{}, restartErr
		}
	}

	as.Status.RefreshTime = now()
	as.Status.SyncedGeneration = as.Generation
	if len(remoteVersions) > 0 {
		as.Status.RemoteVersions = remoteVersions
	}
	r.markReady(as)
	if err := r.Status().Update(ctx, as); err != nil {
		return ctrl.Result{}, err
	}

	r.Recorder.Event(as, corev1.EventTypeNormal, akeylessv1alpha1.ReasonSecretSynced, "secret synced")
	return ctrl.Result{RequeueAfter: requeueAfter(as)}, nil
}

func (r *Reconciler) checkRemoteVersions(ctx context.Context, as *akeylessv1alpha1.AkeylessSecret) (bool, map[string]int32, error) {
	store, _, err := akeylesssync.ResolveStore(ctx, r.Client, as.Namespace, as.Spec.StoreRef)
	if err != nil {
		return false, nil, err
	}

	secretsClient, err := r.Provider.NewClient(ctx, store, r.Client, as.Namespace)
	if err != nil {
		return false, nil, err
	}
	defer func() { _ = secretsClient.Close(ctx) }()

	versionClient, ok := secretsClient.(akeylessprovider.RemoteVersionClient)
	if !ok {
		return true, nil, nil
	}

	observed := map[string]int32{}
	for _, mapping := range as.Spec.Data {
		version, err := versionClient.GetRemoteItemVersion(ctx, mapping.RemoteRef.Key)
		if err != nil {
			return false, nil, fmt.Errorf("describe %q: %w", mapping.RemoteRef.Key, err)
		}
		observed[mapping.RemoteRef.Key] = version
	}

	return remoteVersionsChanged(as.Status.RemoteVersions, observed), observed, nil
}

func (r *Reconciler) markReady(as *akeylessv1alpha1.AkeylessSecret) {
	setCondition(as, akeylessv1alpha1.SecretConditionReady, corev1.ConditionTrue, akeylessv1alpha1.ReasonSecretSynced, "secret synced")
}

func (r *Reconciler) markFailed(as *akeylessv1alpha1.AkeylessSecret, err error) {
	setCondition(as, akeylessv1alpha1.SecretConditionReady, corev1.ConditionFalse, akeylessv1alpha1.ReasonSecretSyncedError, err.Error())
	r.Recorder.Event(as, corev1.EventTypeWarning, akeylessv1alpha1.ReasonSecretSyncedError, err.Error())
}

func setCondition(as *akeylessv1alpha1.AkeylessSecret, t string, status corev1.ConditionStatus, reason, message string) {
	now := metav1.NewTime(time.Now())
	for i := range as.Status.Conditions {
		if as.Status.Conditions[i].Type == t {
			as.Status.Conditions[i].Status = status
			as.Status.Conditions[i].Reason = reason
			as.Status.Conditions[i].Message = message
			as.Status.Conditions[i].LastTransitionTime = now
			return
		}
	}
	as.Status.Conditions = append(as.Status.Conditions, akeylessv1alpha1.AkeylessSecretCondition{
		Type:               t,
		Status:             status,
		Reason:             reason,
		Message:            message,
		LastTransitionTime: now,
	})
}
