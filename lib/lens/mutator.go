/*
Copyright 2019 Gravitational, Inc.

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

package lens

import (
	"bytes"
	"encoding/json"
	"fmt"

	clusterv1beta1 "github.com/gravitational/gravity/lib/apis/cluster/v1beta1"
	"github.com/gravitational/gravity/lib/app/resources"
	clusterv1beta1client "github.com/gravitational/gravity/lib/client/clientset/versioned/typed/cluster/v1beta1"
	"github.com/gravitational/gravity/lib/constants"
	"github.com/gravitational/gravity/lib/loc"

	admissionv1beta1 "k8s.io/api/admission/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"

	"github.com/gravitational/trace"
	"github.com/sirupsen/logrus"
)

type MutatorConfig struct {
	KubeConfig      *rest.Config
	DefaultRegistry string
}

func (c *MutatorConfig) CheckAndSetDefaults() error {
	if c.KubeConfig == nil {
		return trace.BadParameter("missing KubeConfig")
	}
	if c.DefaultRegistry == "" {
		c.DefaultRegistry = DefaultRegistry
	}
	return nil
}

type Mutator struct {
	MutatorConfig
	logrus.FieldLogger
	client *clusterv1beta1client.ClusterV1beta1Client
}

func NewMutator(config MutatorConfig) (*Mutator, error) {
	if err := config.CheckAndSetDefaults(); err != nil {
		return nil, trace.Wrap(err)
	}
	client, err := clusterv1beta1client.NewForConfig(config.KubeConfig)
	if err != nil {
		return nil, trace.Wrap(err)
	}
	return &Mutator{
		MutatorConfig: config,
		FieldLogger:   logrus.WithField(trace.Component, "lens:mutator"),
		client:        client,
	}, nil
}

func (m *Mutator) loadImageSets() ([]clusterv1beta1.ImageSet, error) {
	imageSets, err := m.client.ImageSets(constants.AllNamespaces).List(metav1.ListOptions{})
	if err != nil {
		return nil, trace.Wrap(err)
	}
	return imageSets.Items, nil
}

func (m *Mutator) rewriteImage(image string, log logrus.FieldLogger) (string, bool, error) {
	parsed, err := loc.ParseDockerImage(image)
	if err != nil {
		return "", false, trace.Wrap(err)
	}
	imageSets, err := m.loadImageSets()
	if err != nil {
		return "", false, trace.Wrap(err)
	}
	// Search image sets for a match with the provided image.
	for _, imageSet := range imageSets {
		for _, image := range imageSet.Spec.Images {
			parsedFromSet, err := loc.ParseDockerImage(image.Image)
			if err != nil {
				return "", false, trace.Wrap(err)
			}
			if parsedFromSet.Repository == parsed.Repository && parsedFromSet.Tag == parsed.Tag {
				// Found a match.
				if image.Registry != "" {
					parsed.Registry = image.Registry
				} else {
					parsed.Registry = m.DefaultRegistry
				}
				log.Debugf("Image %v matched ImageSet %v/%v, will rewrite registry to %v.",
					image, imageSet.Namespace, imageSet.Name, parsed.Registry)
				return parsed.String(), true, nil
			}
		}
	}
	// No match found among any of the image sets.
	log.Debugf("No match for image %v.", image)
	return image, false, nil
}

func (m *Mutator) processSpec(spec corev1.PodSpec, log logrus.FieldLogger) (patches []patch, err error) {
	for i, container := range spec.InitContainers {
		image, rewritten, err := m.rewriteImage(container.Image, log)
		if err != nil {
			return nil, trace.Wrap(err)
		}
		if rewritten {
			log.WithField("container", container.Name).Infof("Will rewrite: %v -> %v.",
				container.Image, image)
			patches = append(patches, patch{
				Op:    "replace",
				Path:  fmt.Sprintf("/spec/initContainers/%v/image", i),
				Value: image,
			})
		}
	}
	for i, container := range spec.Containers {
		image, rewritten, err := m.rewriteImage(container.Image, log)
		if err != nil {
			return nil, trace.Wrap(err)
		}
		if rewritten {
			log.WithField("container", container.Name).Infof("Will rewrite: %v -> %v.",
				container.Image, image)
			patches = append(patches, patch{
				Op:    "replace",
				Path:  fmt.Sprintf("/spec/containers/%v/image", i),
				Value: image,
			})
		}
	}
	return patches, nil
}

func (m *Mutator) Mutate(req *admissionv1beta1.AdmissionRequest) (*admissionv1beta1.AdmissionResponse, error) {
	log := m.WithFields(logrus.Fields{
		"kind": req.Kind,
		"ns":   req.Namespace,
		"name": req.Name,
		"uid":  req.UID,
		"op":   req.Operation,
		"user": req.UserInfo,
	})
	log.Info("Got admission request.")
	resource, err := resources.Decode(bytes.NewReader(req.Object.Raw))
	if err != nil {
		return nil, trace.Wrap(err)
	}
	if len(resource.Objects) != 1 {
		return nil, trace.BadParameter("expected 1 object, got: %s", req.Object.Raw)
	}
	var patches []patch
	switch object := resource.Objects[0].(type) {
	case *corev1.Pod:
		patches, err = m.processSpec(object.Spec, log)
		if err != nil {
			return nil, trace.Wrap(err)
		}
	default:
		return nil, trace.BadParameter("unrecognized object %[1]T %[1]v", object)
	}
	patchBytes, err := json.Marshal(patches)
	if err != nil {
		return nil, trace.Wrap(err)
	}
	return &admissionv1beta1.AdmissionResponse{
		UID:       req.UID,
		Allowed:   true,
		Patch:     patchBytes,
		PatchType: patchType(),
	}, nil
}

type patch struct {
	Op    string      `json:"op"`
	Path  string      `json:"path"`
	Value interface{} `json:"value,omitempty"`
}

func patchType() *admissionv1beta1.PatchType {
	patchType := admissionv1beta1.PatchTypeJSONPatch
	return &patchType
}
