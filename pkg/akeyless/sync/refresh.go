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
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	akeylessv1alpha1 "github.com/external-secrets/external-secrets/apis/akeyless/v1alpha1"
)

const defaultSyncInterval = time.Hour

// RefreshConfig controls sync timing for Akeyless sync resources.
type RefreshConfig struct {
	Policy       akeylessv1alpha1.SyncPolicy
	SyncInterval *metav1.Duration
	DefaultPoll  time.Duration
	Generation   int64
	SyncedGen    int64
	RefreshTime  metav1.Time
	Annotations  map[string]string
}

// ShouldSync reports whether a full fetch from Akeyless is needed.
func ShouldSync(cfg RefreshConfig) bool {
	if ForceSyncPending(cfg.Annotations, cfg.RefreshTime) && cfg.Policy != akeylessv1alpha1.SyncPolicyCreatedOnce {
		return true
	}

	switch normalizedPolicy(cfg) {
	case akeylessv1alpha1.SyncPolicyCreatedOnce:
		return cfg.SyncedGen == 0 || cfg.RefreshTime.IsZero()
	case akeylessv1alpha1.SyncPolicyOnSpecChange:
		return cfg.SyncedGen != cfg.Generation
	case akeylessv1alpha1.SyncPolicyOnWebhook:
		return shouldSyncPeriodic(cfg)
	case akeylessv1alpha1.SyncPolicyPeriodic, akeylessv1alpha1.SyncPolicyOnRemoteChange:
		return shouldSyncPeriodic(cfg)
	default:
		return shouldSyncPeriodic(cfg)
	}
}

// RequeueAfter returns the delay until the next reconcile.
func RequeueAfter(cfg RefreshConfig) time.Duration {
	if cfg.RefreshTime.IsZero() {
		return interval(cfg)
	}
	next := cfg.RefreshTime.Add(interval(cfg))
	if d := time.Until(next); d > 0 {
		return d
	}
	return interval(cfg)
}

// Interval returns the configured sync interval with defaults applied.
func Interval(cfg RefreshConfig) time.Duration {
	return interval(cfg)
}

func normalizedPolicy(cfg RefreshConfig) akeylessv1alpha1.SyncPolicy {
	if cfg.Policy == "" {
		return akeylessv1alpha1.SyncPolicyOnRemoteChange
	}
	return cfg.Policy
}

func interval(cfg RefreshConfig) time.Duration {
	if cfg.SyncInterval != nil && cfg.SyncInterval.Duration > 0 {
		return cfg.SyncInterval.Duration
	}
	if cfg.DefaultPoll > 0 {
		return cfg.DefaultPoll
	}
	if normalizedPolicy(cfg) == akeylessv1alpha1.SyncPolicyOnRemoteChange ||
		normalizedPolicy(cfg) == akeylessv1alpha1.SyncPolicyOnWebhook {
		return 5 * time.Minute
	}
	return defaultSyncInterval
}

func shouldSyncPeriodic(cfg RefreshConfig) bool {
	if cfg.SyncedGen == 0 || cfg.RefreshTime.IsZero() {
		return true
	}
	if cfg.SyncedGen != cfg.Generation {
		return true
	}
	if cfg.RefreshTime.Time.After(time.Now()) {
		return true
	}
	return cfg.RefreshTime.Add(interval(cfg)).Before(time.Now())
}

// ForceSyncPending is true when a webhook bumped the force-sync annotation after the last sync.
func ForceSyncPending(annotations map[string]string, refreshTime metav1.Time) bool {
	if annotations == nil {
		return false
	}
	ts, ok := annotations[akeylessv1alpha1.AnnotationForceSync]
	if !ok || ts == "" {
		return false
	}
	t, err := time.Parse(time.RFC3339, ts)
	if err != nil {
		return true
	}
	return refreshTime.IsZero() || t.After(refreshTime.Time)
}

// RemoteVersionsChanged compares observed Akeyless item versions.
func RemoteVersionsChanged(current, observed map[string]int32) bool {
	if len(current) != len(observed) {
		return true
	}
	for k, v := range current {
		if observed[k] != v {
			return true
		}
	}
	return false
}
