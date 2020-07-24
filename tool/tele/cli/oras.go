package cli

import (
	"context"

	"github.com/gravitational/gravity/lib/oras"
)

func pushToRegistry(ctx context.Context, reference, path string) error {
	return oras.Push(ctx, oras.PushRequest{
		Reference: reference,
		Path:      path,
	})
}

func pullFromRegistry(ctx context.Context, reference, outPath string) error {
	return oras.Pull(ctx, oras.PullRequest{
		Reference: reference,
		Path:      outPath,
	})
}
