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
	"github.com/gravitational/gravity/lib/loc"

	admissionv1beta1 "k8s.io/api/admission/v1beta1"
	admissionregistrationv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	admissionv1beta1client "k8s.io/client-go/kubernetes/typed/admissionregistration/v1beta1"
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
	admissionClient *admissionv1beta1client.AdmissionregistrationV1beta1Client
	clusterClient   *clusterv1beta1client.ClusterV1beta1Client
}

func NewMutator(config MutatorConfig) (*Mutator, error) {
	if err := config.CheckAndSetDefaults(); err != nil {
		return nil, trace.Wrap(err)
	}
	admissionClient, err := admissionv1beta1client.NewForConfig(config.KubeConfig)
	if err != nil {
		return nil, trace.Wrap(err)
	}
	clusterClient, err := clusterv1beta1client.NewForConfig(config.KubeConfig)
	if err != nil {
		return nil, trace.Wrap(err)
	}
	return &Mutator{
		MutatorConfig:   config,
		FieldLogger:     logrus.WithField(trace.Component, "lens:mutator"),
		admissionClient: admissionClient,
		clusterClient:   clusterClient,
	}, nil
}

type WebhookConfig struct {
	CAPEM            []byte
	ServiceNamespace string
	ServiceName      string
}

func (m *Mutator) RegisterWebhook(config WebhookConfig) error {
	webhookConfiguration := &admissionregistrationv1beta1.MutatingWebhookConfiguration{
		TypeMeta: metav1.TypeMeta{
			Kind:       "MutatingWebhookConfiguration",
			APIVersion: admissionregistrationv1beta1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "lens-admission-server",
		},
		Webhooks: []admissionregistrationv1beta1.MutatingWebhook{
			{},
		},
	}
}

func (m *Mutator) loadImageSets(namespace string) ([]clusterv1beta1.ImageSet, error) {
	imageSets, err := m.clusterClient.ImageSets(namespace).List(metav1.ListOptions{})
	if err != nil {
		return nil, trace.Wrap(err)
	}
	return imageSets.Items, nil
}

func (m *Mutator) rewriteImage(namespace, image string, log logrus.FieldLogger) (string, bool, error) {
	parsed, err := loc.ParseDockerImage(image)
	if err != nil {
		return "", false, trace.Wrap(err)
	}
	imageSets, err := m.loadImageSets(namespace)
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
				registry := m.registryFor(image)
				if registry == parsed.Registry {
					log.Debugf("Image %v already points to registry %v.", image, registry)
					return image, false, nil
				}
				parsed.Registry = registry
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

// registryFor returns the registry specified in the provided image spec
// or the default one.
func (m *Mutator) registryFor(image clusterv1beta1.ImageSetImage) string {
	if image.Registry != "" {
		return image.Registry
	}
	return m.DefaultRegistry
}

func (m *Mutator) processSpec(namespace string, spec corev1.PodSpec, log logrus.FieldLogger) (patches []patch, err error) {
	for i, container := range spec.InitContainers {
		image, rewritten, err := m.rewriteImage(namespace, container.Image, log)
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
		image, rewritten, err := m.rewriteImage(namespace, container.Image, log)
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
		patches, err = m.processSpec(req.Namespace, object.Spec, log)
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
