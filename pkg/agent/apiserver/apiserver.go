// Copyright 2020 Antrea Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package apiserver

import (
	"fmt"
	"net"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	k8sversion "k8s.io/apimachinery/pkg/version"
	genericapiserver "k8s.io/apiserver/pkg/server"
	genericoptions "k8s.io/apiserver/pkg/server/options"

	"github.com/vmware-tanzu/antrea/pkg/agent/apiserver/handlers/addressgroup"
	"github.com/vmware-tanzu/antrea/pkg/agent/apiserver/handlers/agentinfo"
	"github.com/vmware-tanzu/antrea/pkg/agent/apiserver/handlers/appliedtogroup"
	"github.com/vmware-tanzu/antrea/pkg/agent/apiserver/handlers/networkpolicy"
	"github.com/vmware-tanzu/antrea/pkg/monitor"
	antreaversion "github.com/vmware-tanzu/antrea/pkg/version"
)

const (
	Name = "antrea-agent-api"
	Port = 10443
)

var (
	scheme = runtime.NewScheme()
	codecs = serializer.NewCodecFactory(scheme)
)

type agentAPIServer struct {
	GenericAPIServer *genericapiserver.GenericAPIServer
}

func (s *agentAPIServer) Run(stopCh <-chan struct{}) error {
	return s.GenericAPIServer.PrepareRun().Run(stopCh)
}

func installHandlers(aq monitor.AgentQuerier, npq monitor.AgentNetworkPolicyInfoQuerier, s *genericapiserver.GenericAPIServer) {
	s.Handler.NonGoRestfulMux.HandleFunc("/agentinfo", agentinfo.HandleFunc(aq))
	s.Handler.NonGoRestfulMux.HandleFunc("/networkpolicies", networkpolicy.HandleFunc(npq))
	s.Handler.NonGoRestfulMux.HandleFunc("/appliedtogroups", appliedtogroup.HandleFunc(npq))
	s.Handler.NonGoRestfulMux.HandleFunc("/addressgroups", addressgroup.HandleFunc(npq))
}

// New creates an APIServer for running in antrea agent.
func New(aq monitor.AgentQuerier, npq monitor.AgentNetworkPolicyInfoQuerier) (*agentAPIServer, error) {
	cfg, err := newConfig()
	if err != nil {
		return nil, err
	}
	s, err := cfg.New(Name, genericapiserver.NewEmptyDelegate())
	if err != nil {
		return nil, err
	}
	installHandlers(aq, npq, s)
	return &agentAPIServer{GenericAPIServer: s}, nil
}

func newConfig() (*genericapiserver.CompletedConfig, error) {
	secureServing := genericoptions.NewSecureServingOptions().WithLoopback()
	// Set the PairName but leave certificate directory blank to generate in-memory by default.
	secureServing.ServerCert.CertDirectory = ""
	secureServing.ServerCert.PairName = Name
	ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", Port))
	if err != nil {
		return nil, err
	}
	secureServing.Listener = ln
	if err := secureServing.MaybeDefaultWithSelfSignedCerts("localhost", nil, []net.IP{net.ParseIP("127.0.0.1")}); err != nil {
		return nil, fmt.Errorf("error creating self-signed certificates: %v", err)
	}
	v := antreaversion.GetVersion()
	serverCfg := genericapiserver.NewConfig(codecs)
	serverCfg.Version = &k8sversion.Info{
		Major:        fmt.Sprint(v.Major),
		Minor:        fmt.Sprint(v.Minor),
		GitVersion:   v.String(),
		GitTreeState: antreaversion.GitTreeState,
		GitCommit:    antreaversion.GetGitSHA(),
	}
	if err := secureServing.ApplyTo(&serverCfg.SecureServing, &serverCfg.LoopbackClientConfig); err != nil {
		return nil, err
	}

	completedServerCfg := serverCfg.Complete(nil)
	return &completedServerCfg, nil
}