/*
Copyright © The ESO Authors

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

import (
	"context"
	"testing"
	"time"

	"github.com/akeylesslabs/akeyless-go/v4"
	"github.com/stretchr/testify/require"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	fakeakeyless "github.com/external-secrets/external-secrets/providers/v1/akeyless/fake"
)

// json.Unmarshal errors quote the offending character of the input, so the
// error returned for an invalid secret value must not wrap them.
func TestGetSecretMapErrorDoesNotLeakSecretValue(t *testing.T) {
	mock := fakeakeyless.New().SetGetSecretFn(func(_ string, _ int32) (string, error) {
		return `{"password": hunter2}`, nil
	})
	client := &Akeyless{Client: mock}
	ctx := context.WithValue(context.Background(), aKeylessToken, "token")

	_, err := client.GetSecretMap(ctx, esv1.ExternalSecretDataRemoteRef{Key: "my-secret"})

	require.EqualError(t, err, errJSONSecretUnmarshal)
	require.NotContains(t, err.Error(), "hunter2")
	require.NotContains(t, err.Error(), "invalid character")
}

// OnRemoteChange (the default syncPolicy) calls GetRemoteItemVersion with a plain
// context, so it has to obtain the token itself.
func TestGetRemoteItemVersionObtainsToken(t *testing.T) {
	token := jwtWithExp(t, time.Now().Add(time.Hour))
	mock := &countingTokenClient{AkeylessMockClient: fakeakeyless.New(), token: token}
	var seenToken any
	mock.SetDescribeItemFn(func(ctx context.Context, _ string) (*akeyless.Item, error) {
		seenToken = ctx.Value(aKeylessToken)
		item := akeyless.NewItem()
		item.SetLastVersion(3)
		return item, nil
	})
	client := &Akeyless{Client: mock}

	version, err := client.GetRemoteItemVersion(context.Background(), "my-item")

	require.NoError(t, err)
	require.Equal(t, int32(3), version)
	require.Equal(t, token, seenToken)
	require.Equal(t, int32(1), mock.calls.Load())
}
