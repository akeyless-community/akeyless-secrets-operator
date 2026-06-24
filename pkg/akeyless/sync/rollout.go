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
	"fmt"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	akeylessv1alpha1 "github.com/external-secrets/external-secrets/apis/akeyless/v1alpha1"
)

// TriggerRolloutRestarts patches pod templates to restart referenced workloads.
func TriggerRolloutRestarts(ctx context.Context, c client.Client, namespace string, targets []akeylessv1alpha1.RolloutRestartTarget) error {
	for _, target := range targets {
		if err := restartTarget(ctx, c, namespace, target); err != nil {
			return err
		}
	}
	return nil
}

func restartTarget(ctx context.Context, c client.Client, namespace string, target akeylessv1alpha1.RolloutRestartTarget) error {
	restartedAt := time.Now().Format(time.RFC3339)
	key := client.ObjectKey{Name: target.Name, Namespace: namespace}

	switch target.Kind {
	case "Deployment", "":
		deploy := &appsv1.Deployment{}
		if err := c.Get(ctx, key, deploy); err != nil {
			return fmt.Errorf("get Deployment %q: %w", target.Name, err)
		}
		return patchPodTemplateRestart(ctx, c, deploy, func(d *appsv1.Deployment) *metav1.ObjectMeta {
			return &d.Spec.Template.ObjectMeta
		}, restartedAt)
	case "StatefulSet":
		sts := &appsv1.StatefulSet{}
		if err := c.Get(ctx, key, sts); err != nil {
			return fmt.Errorf("get StatefulSet %q: %w", target.Name, err)
		}
		return patchPodTemplateRestart(ctx, c, sts, func(s *appsv1.StatefulSet) *metav1.ObjectMeta {
			return &s.Spec.Template.ObjectMeta
		}, restartedAt)
	case "DaemonSet":
		ds := &appsv1.DaemonSet{}
		if err := c.Get(ctx, key, ds); err != nil {
			return fmt.Errorf("get DaemonSet %q: %w", target.Name, err)
		}
		return patchPodTemplateRestart(ctx, c, ds, func(d *appsv1.DaemonSet) *metav1.ObjectMeta {
			return &d.Spec.Template.ObjectMeta
		}, restartedAt)
	default:
		return fmt.Errorf("unsupported rollout restart kind %q", target.Kind)
	}
}

func patchPodTemplateRestart[T client.Object](
	ctx context.Context,
	c client.Client,
	obj T,
	templateMeta func(T) *metav1.ObjectMeta,
	restartedAt string,
) error {
	meta := templateMeta(obj)
	if meta.Annotations == nil {
		meta.Annotations = map[string]string{}
	}
	meta.Annotations[akeylessv1alpha1.AnnotationRestarted] = restartedAt
	if err := c.Update(ctx, obj); err != nil {
		if apierrors.IsConflict(err) {
			return fmt.Errorf("conflict restarting %T %q: %w", obj, obj.GetName(), err)
		}
		return err
	}
	return nil
}
