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
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	akeylessv1alpha1 "github.com/external-secrets/external-secrets/apis/akeyless/v1alpha1"
)

func TestForceSyncPending(t *testing.T) {
	t.Parallel()
	refresh := metav1.NewTime(time.Now().Add(-time.Hour))
	if !ForceSyncPending(map[string]string{
		akeylessv1alpha1.AnnotationForceSync: time.Now().Format(time.RFC3339),
	}, refresh) {
		t.Fatal("expected force sync after webhook annotation")
	}
	if ForceSyncPending(map[string]string{}, refresh) {
		t.Fatal("expected no force sync without annotation")
	}
}

func TestOnWebhookPolicyUsesFallbackPolling(t *testing.T) {
	t.Parallel()
	cfg := RefreshConfig{
		Policy:      akeylessv1alpha1.SyncPolicyOnWebhook,
		SyncedGen:   1,
		Generation:  1,
		RefreshTime: metav1.NewTime(time.Now().Add(-10 * time.Minute)),
	}
	if !ShouldSync(cfg) {
		t.Fatal("OnWebhook should fall back to periodic polling")
	}
}
