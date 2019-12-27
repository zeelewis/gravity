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
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"

	"github.com/gravitational/gravity/lib/utils"
	"github.com/gravitational/gravity/lib/utils/fields"

	admissionv1beta1 "k8s.io/api/admission/v1beta1"

	"github.com/gravitational/roundtrip"
	"github.com/gravitational/trace"
	"github.com/julienschmidt/httprouter"
	"github.com/sirupsen/logrus"
)

type AdmissionServerConfig struct {
	ListenAddress  string
	CertificatePEM []byte
	KeyPEM         []byte
	Mutator        *Mutator
}

func (c *AdmissionServerConfig) CheckAndSetDefaults() error {
	if c.ListenAddress == "" {
		c.ListenAddress = fmt.Sprintf("%v:%v", DefaultListenAddress, DefaultListenPort)
	} else {
		c.ListenAddress = utils.EnsurePort(c.ListenAddress, DefaultListenPort)
	}
	if len(c.CertificatePEM) == 0 {
		return trace.BadParameter("missing CertificatePEM")
	}
	if len(c.KeyPEM) == 0 {
		return trace.BadParameter("missing KeyPEM")
	}
	if c.Mutator == nil {
		return trace.BadParameter("missing Mutator")
	}
	return nil
}

type AdmissionServer struct {
	AdmissionServerConfig
	http.Server
	logrus.FieldLogger
}

func NewAdmissionServer(config AdmissionServerConfig) (*AdmissionServer, error) {
	if err := config.CheckAndSetDefaults(); err != nil {
		return nil, trace.Wrap(err)
	}
	tlsCertificate, err := tls.X509KeyPair(config.CertificatePEM, config.KeyPEM)
	if err != nil {
		return nil, trace.Wrap(err)
	}
	router := httprouter.New()
	server := &AdmissionServer{
		FieldLogger: logrus.WithField(trace.Component, "lens:server"),
		Server: http.Server{
			Handler: router,
			TLSConfig: &tls.Config{
				Certificates: []tls.Certificate{tlsCertificate},
			},
		},
		AdmissionServerConfig: config,
	}
	router.GET("/healthz", server.handler(server.healthz))
	router.POST("/mutate", server.handler(server.mutate))
	return server, nil
}

func (s *AdmissionServer) ListenAndServe() error {
	listener, err := net.Listen("tcp", s.ListenAddress)
	if err != nil {
		return trace.Wrap(err)
	}
	s.Infof("Admission webhook server is listening on %v.", s.ListenAddress)
	err = s.Server.Serve(tls.NewListener(listener, s.Server.TLSConfig))
	if err != nil {
		return trace.Wrap(err)
	}
	return nil
}

func (s *AdmissionServer) Shutdown(ctx context.Context) error {
	s.Info("Admission webhook server is shutting down.")
	err := s.Server.Shutdown(ctx)
	if err != nil {
		return trace.Wrap(err)
	}
	return nil
}

func (s *AdmissionServer) healthz(w http.ResponseWriter, r *http.Request, p httprouter.Params) error {
	roundtrip.ReplyJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	return nil
}

func (s *AdmissionServer) mutate(w http.ResponseWriter, r *http.Request, p httprouter.Params) error {
	requestBody, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return trace.Wrap(err)
	}
	var admissionReview admissionv1beta1.AdmissionReview
	if err := json.Unmarshal(requestBody, &admissionReview); err != nil {
		return trace.Wrap(err)
	}
	admissionReview.Response, err = s.Mutator.Mutate(admissionReview.Request)
	if err != nil {
		return trace.Wrap(err)
	}
	roundtrip.ReplyJSON(w, http.StatusOK, admissionReview)
	return nil
}

func (s *AdmissionServer) handler(fn func(http.ResponseWriter, *http.Request, httprouter.Params) error) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		log := s.WithFields(fields.FromRequest(r))
		if err := fn(w, r, p); err != nil {
			log.WithError(err).Error("Handler error.")
			trace.WriteError(w, err)
		}
	}
}
