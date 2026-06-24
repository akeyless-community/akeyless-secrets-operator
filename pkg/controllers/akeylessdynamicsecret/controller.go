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

// Package akeylessdynamicsecret reconciles AkeylessDynamicSecret resources.
package akeylessdynamicsecret

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

// Reconciler syncs AkeylessDynamicSecret resources.
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
		r.Recorder = mgr.GetEventRecorderFor("akeyless-dynamic-secrets-controller")
	}
	return ctrl.NewControllerManagedBy(mgr).
		WithOptions(opts).
		For(&akeylessv1alpha1.AkeylessDynamicSecret{}, builder.WithPredicates(predicate.Or(
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

// Reconcile fetches a dynamic or rotated credential from Akeyless.
func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("akeylessdynamicsecret", req.NamespacedName)

	ads := &akeylessv1alpha1.AkeylessDynamicSecret{}
	if err := r.Get(ctx, req.NamespacedName, ads); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if !ads.DeletionTimestamp.IsZero() {
		return ctrl.Result{}, nil
	}

	if controllerutil.AddFinalizer(ads, finalizer) {
		if err := r.Update(ctx, ads); err != nil {
			return ctrl.Result{}, err
		}
	}

	targetName := ads.Spec.Target.Name
	if targetName == "" {
		targetName = ads.Name
	}

	existing := &corev1.Secret{}
	err := r.Get(ctx, client.ObjectKey{Name: targetName, Namespace: ads.Namespace}, existing)
	secretExists := err == nil
	if err != nil && !apierrors.IsNotFound(err) {
		return ctrl.Result{}, err
	}

	oldHash := ""
	if secretExists {
		oldHash = existing.Annotations[akeylessv1alpha1.AnnotationDataHash]
	}

	if err := akeylesssync.EnsureStoreReady(ctx, r.Client, ads.Namespace, ads.Spec.StoreRef); err != nil {
		r.markFailed(ads, err)
		_ = r.Status().Update(ctx, ads)
		return ctrl.Result{RequeueAfter: time.Minute}, nil
	}

	cfg := refreshConfig(ads)
	policy := cfg.Policy
	if policy == "" {
		policy = akeylessv1alpha1.SyncPolicyOnRemoteChange
	}

	if !akeylesssync.ShouldSync(cfg) && secretExists && akeylesssync.IsManagedSecretValid(existing) {
		if policy == akeylessv1alpha1.SyncPolicyOnRemoteChange || policy == akeylessv1alpha1.SyncPolicyOnWebhook {
			changed, version, checkErr := r.checkRemoteVersion(ctx, ads)
			if checkErr != nil {
				r.markFailed(ads, checkErr)
				_ = r.Status().Update(ctx, ads)
				return ctrl.Result{}, checkErr
			}
			if !changed {
				return ctrl.Result{RequeueAfter: akeylesssync.RequeueAfter(cfg)}, nil
			}
			if ads.Status.RemoteVersions == nil {
				ads.Status.RemoteVersions = map[string]int32{}
			}
			ads.Status.RemoteVersions[ads.Spec.Path] = version
		} else {
			return ctrl.Result{RequeueAfter: akeylesssync.RequeueAfter(cfg)}, nil
		}
	}

	store, _, err := akeylesssync.ResolveStore(ctx, r.Client, ads.Namespace, ads.Spec.StoreRef)
	if err != nil {
		r.markFailed(ads, err)
		_ = r.Status().Update(ctx, ads)
		return ctrl.Result{}, err
	}

	secretsClient, err := r.Provider.NewClient(ctx, store, r.Client, ads.Namespace)
	if err != nil {
		r.markFailed(ads, err)
		_ = r.Status().Update(ctx, ads)
		return ctrl.Result{}, err
	}
	defer func() { _ = secretsClient.Close(ctx) }()

	secretKey := ads.Spec.SecretKey
	if secretKey == "" {
		secretKey = "value"
	}

	ref := adapter.RemoteRef(akeylessv1alpha1.RemoteRef{
		Key:      ads.Spec.Path,
		Property: ads.Spec.Property,
	})
	val, getErr := secretsClient.GetSecret(ctx, ref)
	if getErr != nil {
		r.markFailed(ads, fmt.Errorf("fetch %q: %w", ads.Spec.Path, getErr))
		_ = r.Status().Update(ctx, ads)
		return ctrl.Result{}, getErr
	}

	secretData := map[string][]byte{secretKey: val}
	remoteVersions := map[string]int32{}
	if versionClient, ok := secretsClient.(akeylessprovider.RemoteVersionClient); ok {
		if version, verErr := versionClient.GetRemoteItemVersion(ctx, ads.Spec.Path); verErr != nil {
			log.V(1).Info("could not read remote version", "path", ads.Spec.Path, "error", verErr)
		} else {
			remoteVersions[ads.Spec.Path] = version
		}
	}

	if err := akeylesssync.ApplyManagedSecret(ctx, r.Client, r.Scheme, ads, ads.Namespace, targetName, ads.Spec.Target, secretData); err != nil {
		r.markFailed(ads, err)
		_ = r.Status().Update(ctx, ads)
		return ctrl.Result{}, err
	}

	newHash := esutils.ObjectHash(secretData)
	if len(ads.Spec.RolloutRestartTargets) > 0 && oldHash != newHash {
		if restartErr := akeylesssync.TriggerRolloutRestarts(ctx, r.Client, ads.Namespace, ads.Spec.RolloutRestartTargets); restartErr != nil {
			r.markFailed(ads, restartErr)
			_ = r.Status().Update(ctx, ads)
			return ctrl.Result{}, restartErr
		}
	}

	ads.Status.RefreshTime = metav1.NewTime(time.Now())
	ads.Status.SyncedGeneration = ads.Generation
	if len(remoteVersions) > 0 {
		ads.Status.RemoteVersions = remoteVersions
	}
	r.markReady(ads)
	if err := r.Status().Update(ctx, ads); err != nil {
		return ctrl.Result{}, err
	}

	r.Recorder.Event(ads, corev1.EventTypeNormal, akeylessv1alpha1.ReasonSecretSynced, "dynamic secret synced")
	return ctrl.Result{RequeueAfter: akeylesssync.RequeueAfter(cfg)}, nil
}

func (r *Reconciler) checkRemoteVersion(ctx context.Context, ads *akeylessv1alpha1.AkeylessDynamicSecret) (bool, int32, error) {
	store, _, err := akeylesssync.ResolveStore(ctx, r.Client, ads.Namespace, ads.Spec.StoreRef)
	if err != nil {
		return false, 0, err
	}

	secretsClient, err := r.Provider.NewClient(ctx, store, r.Client, ads.Namespace)
	if err != nil {
		return false, 0, err
	}
	defer func() { _ = secretsClient.Close(ctx) }()

	versionClient, ok := secretsClient.(akeylessprovider.RemoteVersionClient)
	if !ok {
		return true, 0, nil
	}

	version, err := versionClient.GetRemoteItemVersion(ctx, ads.Spec.Path)
	if err != nil {
		return false, 0, fmt.Errorf("describe %q: %w", ads.Spec.Path, err)
	}

	current := int32(0)
	if ads.Status.RemoteVersions != nil {
		current = ads.Status.RemoteVersions[ads.Spec.Path]
	}
	return current != version, version, nil
}

func refreshConfig(ads *akeylessv1alpha1.AkeylessDynamicSecret) akeylesssync.RefreshConfig {
	return akeylesssync.RefreshConfig{
		Policy:       ads.Spec.SyncPolicy,
		SyncInterval: ads.Spec.SyncInterval,
		DefaultPoll:  time.Minute,
		Generation:   ads.Generation,
		SyncedGen:    ads.Status.SyncedGeneration,
		RefreshTime:  ads.Status.RefreshTime,
		Annotations:  ads.Annotations,
	}
}

func (r *Reconciler) markReady(ads *akeylessv1alpha1.AkeylessDynamicSecret) {
	setCondition(ads, akeylessv1alpha1.SecretConditionReady, corev1.ConditionTrue, akeylessv1alpha1.ReasonSecretSynced, "dynamic secret synced")
}

func (r *Reconciler) markFailed(ads *akeylessv1alpha1.AkeylessDynamicSecret, err error) {
	setCondition(ads, akeylessv1alpha1.SecretConditionReady, corev1.ConditionFalse, akeylessv1alpha1.ReasonSecretSyncedError, err.Error())
	r.Recorder.Event(ads, corev1.EventTypeWarning, akeylessv1alpha1.ReasonSecretSyncedError, err.Error())
}

func setCondition(ads *akeylessv1alpha1.AkeylessDynamicSecret, t string, status corev1.ConditionStatus, reason, message string) {
	now := metav1.NewTime(time.Now())
	for i := range ads.Status.Conditions {
		if ads.Status.Conditions[i].Type == t {
			ads.Status.Conditions[i].Status = status
			ads.Status.Conditions[i].Reason = reason
			ads.Status.Conditions[i].Message = message
			ads.Status.Conditions[i].LastTransitionTime = now
			return
		}
	}
	ads.Status.Conditions = append(ads.Status.Conditions, akeylessv1alpha1.AkeylessDynamicSecretCondition{
		Type:               t,
		Status:             status,
		Reason:             reason,
		Message:            message,
		LastTransitionTime: now,
	})
}
