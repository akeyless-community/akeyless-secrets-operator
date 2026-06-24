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

package akeylesssecret

import (
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	akeylessv1alpha1 "github.com/external-secrets/external-secrets/apis/akeyless/v1alpha1"
)

func TestRemoteVersionsChanged(t *testing.T) {
	t.Parallel()
	if !remoteVersionsChanged(nil, map[string]int32{"a": 1}) {
		t.Fatal("expected change when current is nil")
	}
	if !remoteVersionsChanged(map[string]int32{"a": 1}, map[string]int32{"a": 2}) {
		t.Fatal("expected change on version bump")
	}
	if remoteVersionsChanged(map[string]int32{"a": 1}, map[string]int32{"a": 1}) {
		t.Fatal("expected no change")
	}
}

func TestShouldSyncOnRemoteChange(t *testing.T) {
	t.Parallel()
	as := &akeylessv1alpha1.AkeylessSecret{
		ObjectMeta: metav1.ObjectMeta{Generation: 1},
		Spec: akeylessv1alpha1.AkeylessSecretSpec{
			SyncPolicy: akeylessv1alpha1.SyncPolicyOnRemoteChange,
			SyncInterval: &metav1.Duration{Duration: time.Minute},
		},
		Status: akeylessv1alpha1.AkeylessSecretStatus{
			SyncedGeneration: 1,
			RefreshTime:      metav1.NewTime(time.Now().Add(-2 * time.Minute)),
		},
	}
	if !shouldSync(as) {
		t.Fatal("expected periodic version-check window to trigger sync evaluation")
	}
}
