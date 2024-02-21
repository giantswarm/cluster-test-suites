package helper

import (
	"context"

	tc "github.com/gravitational/teleport/api/client"
)

var NewTeleportClient = func(ctx context.Context, identityFilePath string) (*tc.Client, error) {
	proxyAddr := "teleport.giantswarm.io:443"
	client, err := tc.New(ctx, tc.Config{
		Addrs: []string{
			proxyAddr,
		},
		Credentials: []tc.Credentials{
			tc.LoadIdentityFile(identityFilePath),
		},
	})
	if err != nil {
		return client, err
	}

	_, err = client.Ping(ctx)
	if err != nil {
		return client, err
	}

	return client, nil
}
