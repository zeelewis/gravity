/*
Copyright 2018 Gravitational, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/gravitational/gravity/lib/app/service"
	"github.com/gravitational/gravity/lib/builder"
	"github.com/gravitational/gravity/lib/utils"
	"github.com/gravitational/gravity/tool/common"

	"github.com/buger/goterm"
	"github.com/fatih/color"
	teleutils "github.com/gravitational/teleport/lib/utils"
	"github.com/gravitational/trace"
)

// BuildParameters represents the arguments provided for building an application
type BuildParameters struct {
	// StateDir is build state directory, if was specified
	StateDir string
	// SourcePath is the path to a manifest file or a Helm chart to build image from
	SourcePath string
	// OutPath holds the path to the installer tarball to be output
	OutPath string
	// Overwrite indicates whether or not to overwrite an existing installer file
	Overwrite bool
	// SkipVersionCheck indicates whether or not to perform the version check of the tele binary with the application's runtime at build time
	SkipVersionCheck bool
	// Silent is whether builder should report progress to the console
	Silent bool
	// Verbose turns on more detailed progress output
	Verbose bool
	// Insecure turns on insecure verify mode
	Insecure bool
	// Vendor combines vendoring parameters
	Vendor service.VendorRequest
	// BaseImage sets base image for the cluster image
	BaseImage string
	// UpgradeFrom
	UpgradeFrom string
	// Diff
	Diff bool
}

// Level returns level at which the progress should be reported based on the CLI parameters.
func (p BuildParameters) Level() utils.ProgressLevel {
	if p.Silent { // No output.
		return utils.ProgressLevelNone
	} else if p.Verbose { // Detailed output.
		return utils.ProgressLevelDebug
	}
	return utils.ProgressLevelInfo // Normal output.
}

// BuilderConfig makes builder config from CLI parameters.
func (p BuildParameters) BuilderConfig() builder.Config {
	return builder.Config{
		StateDir:         p.StateDir,
		Insecure:         p.Insecure,
		SkipVersionCheck: p.SkipVersionCheck,
		Parallel:         p.Vendor.Parallel,
		Level:            p.Level(),
	}
}

func buildClusterImage(ctx context.Context, params BuildParameters) error {
	if params.Diff {
		return diffClusterImage(params)
	}
	clusterBuilder, err := builder.NewClusterBuilder(params.BuilderConfig())
	if err != nil {
		return trace.Wrap(err)
	}
	defer clusterBuilder.Close()
	return clusterBuilder.Build(ctx, builder.ClusterRequest{
		SourcePath: params.SourcePath,
		OutputPath: params.OutPath,
		Overwrite:  params.Overwrite,
		BaseImage:  params.BaseImage,
		Vendor:     params.Vendor,
		From:       params.UpgradeFrom,
	})
}

func buildApplicationImage(ctx context.Context, params BuildParameters) error {
	if params.Diff {
		return diffApplicationImage(params)
	}
	appBuilder, err := builder.NewApplicationBuilder(params.BuilderConfig())
	if err != nil {
		return trace.Wrap(err)
	}
	defer appBuilder.Close()
	return appBuilder.Build(ctx, builder.ApplicationRequest{
		ChartPath:  params.SourcePath,
		OutputPath: params.OutPath,
		Overwrite:  params.Overwrite,
		Vendor:     params.Vendor,
		From:       params.UpgradeFrom,
	})
}

func diffClusterImage(params BuildParameters) error {
	clusterBuilder, err := builder.NewClusterBuilder(params.BuilderConfig())
	if err != nil {
		return trace.Wrap(err)
	}
	defer clusterBuilder.Close()

	newImage, err := clusterBuilder.Inspect(builder.ClusterRequest{
		SourcePath: params.SourcePath,
		Vendor:     params.Vendor,
	})
	if err != nil {
		return trace.Wrap(err)
	}

	oldImage, err := builder.GetImages(context.TODO(), params.UpgradeFrom)
	if err != nil {
		return trace.Wrap(err)
	}

	diffImages(oldImage, newImage)
	return nil
}

func diffApplicationImage(params BuildParameters) error {
	appBuilder, err := builder.NewApplicationBuilder(params.BuilderConfig())
	if err != nil {
		return trace.Wrap(err)
	}
	defer appBuilder.Close()

	newImage, err := appBuilder.Inspect(builder.ApplicationRequest{
		ChartPath: params.SourcePath,
		Vendor:    params.Vendor,
	})
	if err != nil {
		return trace.Wrap(err)
	}

	oldImage, err := builder.GetImages(context.TODO(), params.UpgradeFrom)
	if err != nil {
		return trace.Wrap(err)
	}

	diffImages(oldImage, newImage)
	return nil
}

func diffImages(oldImage, newImage *builder.InspectResponse) {
	newImages := make(map[string][]string)
	oldImages := make(map[string][]string)
	allImages := []string{}

	for _, image := range newImage.Images {
		newImages[image.Repository] = append(newImages[image.Repository], image.Tag)
		allImages = append(allImages, image.Repository)
	}
	for _, image := range oldImage.Images {
		oldImages[image.Repository] = append(oldImages[image.Repository], image.Tag)
		allImages = append(allImages, image.Repository)
	}

	allImages = teleutils.Deduplicate(allImages)
	sort.Strings(allImages)

	t := goterm.NewTable(0, 10, 5, ' ', 0)
	common.PrintTableHeader(t, []string{
		"",
		fmt.Sprintf("%v:%v", oldImage.Manifest.Locator().Name, oldImage.Manifest.Locator().Version),
		fmt.Sprintf("%v:%v", newImage.Manifest.Locator().Name, newImage.Manifest.Locator().Version),
	})
	for _, image := range allImages {
		oldTags, isOld := oldImages[image]
		newTags, isNew := newImages[image]
		if isOld && isNew {
			for i, newTag := range newTags {
				if !utils.StringInSlice(oldTags, newTag) {
					newTags[i] = color.New(color.Bold, color.FgGreen).Sprint(newTag)
				}
			}
			fmt.Fprintf(t, "%v\t%v\t%v\n", image, strings.Join(oldTags, ", "), strings.Join(newTags, ", "))
		} else if isOld {
			fmt.Fprintf(t, "%v\t%v\t%v\n", image, strings.Join(oldTags, ", "), "")
		} else {
			for i, newTag := range newTags {
				newTags[i] = color.New(color.Bold, color.FgGreen).Sprint(newTag)
			}
			fmt.Fprintf(t, "%v\t%v\t%v\n", image, "", strings.Join(newTags, ", "))
		}
	}

	io.WriteString(os.Stdout, t.String())
}
