package cli

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/containerd/containerd/remotes"
	"github.com/containerd/containerd/remotes/docker"
	auth "github.com/deislabs/oras/pkg/auth/docker"
	"github.com/deislabs/oras/pkg/content"
	"github.com/deislabs/oras/pkg/oras"
	"github.com/gravitational/trace"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

var mediaType = "gravitational.application"

func pushToRegistry(ctx context.Context, reference, path string) error {
	resolver := newResolver("", "", true, true)

	fileStore := content.NewFileStore("")
	defer fileStore.Close()

	desc, err := fileStore.Add(filepath.Base(path), mediaType, path)
	if err != nil {
		return trace.Wrap(err)
	}

	pushContents := []ocispec.Descriptor{desc}

	fmt.Printf("Pushing %s to %s...\n", path, reference)
	desc, err = oras.Push(ctx, resolver, reference, fileStore, pushContents)
	if err != nil {
		return trace.Wrap(err)
	}

	fmt.Printf("Pushed to %s with digest %s\n", reference, desc.Digest)
	return nil
}

func pullFromRegistry(ctx context.Context, reference, outPath string) error {
	resolver := newResolver("", "", true, true)

	fileStore := content.NewFileStore("")
	defer fileStore.Close()

	allowedMediaTypes := []string{mediaType}

	desc, _, err := oras.Pull(ctx, resolver, reference, fileStore, oras.WithAllowedMediaTypes(allowedMediaTypes))
	if err != nil {
		return trace.Wrap(err)
	}

	fmt.Printf("Pulled from %s with digest %s\n", reference, desc.Digest)
	//	fmt.Printf("Try running 'cat %s'\n", fileName)

	return nil
}

func newResolver(username, password string, insecure bool, plainHTTP bool, configs ...string) remotes.Resolver {

	opts := docker.ResolverOptions{
		PlainHTTP: plainHTTP,
	}

	client := http.DefaultClient
	if insecure {
		client.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		}
	}
	opts.Client = client

	if username != "" || password != "" {
		opts.Credentials = func(hostName string) (string, string, error) {
			return username, password, nil
		}
		return docker.NewResolver(opts)
	}
	cli, err := auth.NewClient(configs...)
	if err != nil {
		fmt.Fprintf(os.Stderr, "WARNING: Error loading auth file: %v\n", err)
	}
	resolver, err := cli.Resolver(context.Background(), client, plainHTTP)
	if err != nil {
		fmt.Fprintf(os.Stderr, "WARNING: Error loading resolver: %v\n", err)
		resolver = docker.NewResolver(opts)
	}
	return resolver
}
