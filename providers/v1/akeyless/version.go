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

package akeyless

import "context"

// RemoteVersionClient reports Akeyless item versions without fetching secret values.
type RemoteVersionClient interface {
	GetRemoteItemVersion(ctx context.Context, itemName string) (int32, error)
}

// GetRemoteItemVersion returns the Akeyless last_version for an item path.
func (a *Akeyless) GetRemoteItemVersion(ctx context.Context, itemName string) (int32, error) {
	item, err := a.Client.DescribeItem(ctx, itemName)
	if err != nil {
		return 0, err
	}
	return item.GetLastVersion(), nil
}
