package builder

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/gravitational/gravity/lib/apis/cluster/v1beta1"
	"github.com/gravitational/gravity/lib/defaults"
	"github.com/gravitational/gravity/lib/docker"
	"github.com/gravitational/gravity/lib/loc"
	"github.com/gravitational/gravity/lib/localenv"
	"github.com/gravitational/gravity/lib/pack"

	"github.com/gravitational/trace"
)

// TODO(r0mant): Instead of unpacking provided image (which is slow), see if
// we can modify the build procedure to save a list of embedded images somewhere,
// and fall back to unpacking image for older images.
func GetImages(ctx context.Context, imagePath string) (*InspectResponse, error) {
	env, err := localenv.NewImageEnvironment(imagePath)
	if err != nil {
		return nil, trace.Wrap(err)
	}
	defer env.Close()
	response, err := getImagesFromImageSet(env)
	if err != nil && !trace.IsNotFound(err) {
		return nil, trace.Wrap(err)
	}
	if trace.IsNotFound(err) {
		return getImagesFromRegistry(ctx, env)
	}
	return response, nil
}

func getImagesFromImageSet(env *localenv.ImageEnvironment) (*InspectResponse, error) {
	_, reader, err := env.Packages.ReadPackage(loc.ImageSet)
	if err != nil {
		return nil, trace.Wrap(err)
	}
	defer reader.Close()
	data, err := ioutil.ReadAll(reader)
	if err != nil {
		return nil, trace.Wrap(err)
	}
	var imageSet v1beta1.ImageSet
	if err := json.Unmarshal(data, &imageSet); err != nil {
		return nil, trace.Wrap(err)
	}
	var images []loc.DockerImage
	for _, item := range imageSet.Spec.Images {
		parsed, err := loc.ParseDockerImage(item.Image)
		if err != nil {
			return nil, trace.Wrap(err)
		}
		images = append(images, *parsed)
	}
	return &InspectResponse{
		Manifest: env.Manifest,
		Images:   images,
	}, nil
}

func getImagesFromRegistry(ctx context.Context, env *localenv.ImageEnvironment) (*InspectResponse, error) {
	dir, err := ioutil.TempDir("", "image")
	if err != nil {
		return nil, trace.Wrap(err)
	}
	defer os.RemoveAll(dir)
	err = pack.Unpack(env.Packages, env.Manifest.Locator(), dir, nil)
	if err != nil {
		return nil, trace.Wrap(err)
	}
	images, err := docker.ListImages(ctx, filepath.Join(dir, defaults.RegistryDir))
	if err != nil {
		return nil, trace.Wrap(err)
	}
	return &InspectResponse{
		Manifest: env.Manifest,
		Images:   images,
	}, nil
}
