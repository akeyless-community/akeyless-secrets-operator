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
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	akeylessv1alpha1 "github.com/external-secrets/external-secrets/apis/akeyless/v1alpha1"
	akeylesssync "github.com/external-secrets/external-secrets/pkg/akeyless/sync"
)

func refreshConfig(as *akeylessv1alpha1.AkeylessSecret) akeylesssync.RefreshConfig {
	return akeylesssync.RefreshConfig{
		Policy:       as.Spec.SyncPolicy,
		SyncInterval: as.Spec.SyncInterval,
		Generation:   as.Generation,
		SyncedGen:    as.Status.SyncedGeneration,
		RefreshTime:  as.Status.RefreshTime,
		Annotations:  as.Annotations,
	}
}

func shouldSync(as *akeylessv1alpha1.AkeylessSecret) bool {
	return akeylesssync.ShouldSync(refreshConfig(as))
}

func requeueAfter(as *akeylessv1alpha1.AkeylessSecret) time.Duration {
	return akeylesssync.RequeueAfter(refreshConfig(as))
}

func remoteVersionsChanged(current, observed map[string]int32) bool {
	return akeylesssync.RemoteVersionsChanged(current, observed)
}

func now() metav1.Time {
	return metav1.NewTime(time.Now())
}
