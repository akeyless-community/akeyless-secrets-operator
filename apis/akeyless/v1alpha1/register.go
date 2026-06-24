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

package v1alpha1

import (
	"reflect"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/scheme"
)

const (
	Group   = "secrets.akeyless.io"
	Version = "v1alpha1"
)

var (
	SchemeGroupVersion = schema.GroupVersion{Group: Group, Version: Version}
	SchemeBuilder      = &scheme.Builder{GroupVersion: SchemeGroupVersion}
	AddToScheme          = SchemeBuilder.AddToScheme
)

var (
	AkeylessSecretKind             = reflect.TypeFor[AkeylessSecret]().Name()
	AkeylessDynamicSecretKind      = reflect.TypeFor[AkeylessDynamicSecret]().Name()
	AkeylessSecretStoreKind        = reflect.TypeFor[AkeylessSecretStore]().Name()
	ClusterAkeylessSecretStoreKind = reflect.TypeFor[ClusterAkeylessSecretStore]().Name()
)

func init() {
	SchemeBuilder.Register(&AkeylessSecret{}, &AkeylessSecretList{})
	SchemeBuilder.Register(&AkeylessDynamicSecret{}, &AkeylessDynamicSecretList{})
	SchemeBuilder.Register(&AkeylessSecretStore{}, &AkeylessSecretStoreList{})
	SchemeBuilder.Register(&ClusterAkeylessSecretStore{}, &ClusterAkeylessSecretStoreList{})
}
