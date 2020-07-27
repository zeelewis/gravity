package oras

import (
	"context"
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gravitational/gravity/lib/utils"

	"github.com/containerd/containerd/remotes"
	"github.com/containerd/containerd/remotes/docker"
	auth "github.com/deislabs/oras/pkg/auth/docker"
	"github.com/deislabs/oras/pkg/content"
	"github.com/deislabs/oras/pkg/oras"
	"github.com/gravitational/trace"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

const mediaType = "gravitational.application"

type PushRequest struct {
	Reference string
	Path      string
	Progress  utils.Progress
}

func (r *PushRequest) CheckAndSetDefaults(ctx context.Context) error {
	if r.Progress == nil {
		r.Progress = utils.NewProgressWithConfig(ctx, "Push",
			utils.ProgressConfig{
				StepPrinter: utils.TimestampedStepPrinter,
			})
	}
	return nil
}

type PullRequest struct {
	Reference string
	Path      string
	Progress  utils.Progress
}

func (r *PullRequest) CheckAndSetDefaults(ctx context.Context) error {
	if r.Progress == nil {
		r.Progress = utils.NewProgressWithConfig(ctx, "Pull",
			utils.ProgressConfig{
				StepPrinter: utils.TimestampedStepPrinter,
			})
	}
	return nil
}

func Push(ctx context.Context, req PushRequest) error {
	if err := req.CheckAndSetDefaults(ctx); err != nil {
		return trace.Wrap(err)
	}

	resolver := newResolver("", "", true, true)

	fileStore := content.NewFileStore("")
	defer fileStore.Close()

	desc, err := fileStore.Add(filepath.Base(req.Path), mediaType, req.Path)
	if err != nil {
		return trace.Wrap(err)
	}

	pushContents := []ocispec.Descriptor{desc}

	req.Progress.NextStep("Pushing %v to %v...", req.Path, req.Reference)

	desc, err = oras.Push(ctx, resolver, req.Reference, fileStore, pushContents)
	if err != nil {
		return trace.Wrap(err)
	}

	req.Progress.NextStep("Pushed to %v with digest %s", req.Reference, desc.Digest)
	return nil
}

func Pull(ctx context.Context, req PullRequest) error {
	if err := req.CheckAndSetDefaults(ctx); err != nil {
		return trace.Wrap(err)
	}

	resolver := newResolver("", "", true, true)

	tempDir, err := ioutil.TempDir("", "oras-pull")
	if err != nil {
		return trace.Wrap(err)
	}
	defer os.RemoveAll(tempDir)

	fileStore := content.NewFileStore(tempDir)
	defer fileStore.Close()

	allowedMediaTypes := []string{mediaType}

	req.Progress.NextStep("Downloading %v...", req.Reference)

	desc, descs, err := oras.Pull(ctx, resolver, req.Reference, fileStore, oras.WithAllowedMediaTypes(allowedMediaTypes))
	if err != nil {
		return trace.Wrap(err)
	}

	var name string
	for _, d := range descs {
		if v, ok := d.Annotations[ocispec.AnnotationTitle]; ok {
			name = v
			break
		}
	}

	err = os.Rename(filepath.Join(tempDir, name), req.Path)
	if err != nil {
		return trace.Wrap(err)
	}

	req.Progress.NextStep("Downloaded from %v into %v with digest %s", req.Reference, req.Path, desc.Digest)
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
