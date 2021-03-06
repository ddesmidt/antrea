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

package agentinfo

import (
	"encoding/json"
	"net/http"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/klog"

	"github.com/vmware-tanzu/antrea/pkg/antctl/transform/common"
	"github.com/vmware-tanzu/antrea/pkg/apis/clusterinformation/v1beta1"
	"github.com/vmware-tanzu/antrea/pkg/monitor"
)

// AntreaAgentInfoResponse is the struct for the response of agentinfo command.
// It includes all fields except meta info from v1beta1.AntreaAgentInfo struct.
type AntreaAgentInfoResponse struct {
	Version                     string                              `json:"version,omitempty"`                     // Antrea binary version
	PodRef                      corev1.ObjectReference              `json:"podRef,omitempty"`                      // The Pod that Antrea Agent is running in
	NodeRef                     corev1.ObjectReference              `json:"nodeRef,omitempty"`                     // The Node that Antrea Agent is running in
	NodeSubnet                  []string                            `json:"nodeSubnet,omitempty"`                  // Node subnet
	OVSInfo                     v1beta1.OVSInfo                     `json:"ovsInfo,omitempty"`                     // OVS Information
	NetworkPolicyControllerInfo v1beta1.NetworkPolicyControllerInfo `json:"networkPolicyControllerInfo,omitempty"` // Antrea Agent NetworkPolicy information
	LocalPodNum                 int32                               `json:"localPodNum,omitempty"`                 // The number of Pods which the agent is in charge of
	AgentConditions             []v1beta1.AgentCondition            `json:"agentConditions,omitempty"`             // Agent condition contains types like AgentHealthy
}

// HandleFunc returns the function which can handle queries issued by agentinfo commands.
// The handler function populates Antrea agent information to the response.
func HandleFunc(aq monitor.AgentQuerier) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var info *AntreaAgentInfoResponse
		allInfo := aq.GetAgentInfo()
		info = &AntreaAgentInfoResponse{
			Version:                     allInfo.Version,
			PodRef:                      allInfo.PodRef,
			NodeRef:                     allInfo.NodeRef,
			OVSInfo:                     allInfo.OVSInfo,
			NetworkPolicyControllerInfo: allInfo.NetworkPolicyControllerInfo,
			LocalPodNum:                 allInfo.LocalPodNum,
			AgentConditions:             allInfo.AgentConditions,
		}
		err := json.NewEncoder(w).Encode(info)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			klog.Errorf("Error when encoding AntreaAgentInfo to json: %v", err)
		}
	}
}

var _ common.TableOutput = new(AntreaAgentInfoResponse)

func (r AntreaAgentInfoResponse) GetTableHeader() []string {
	return []string{"POD", "NODE", "STATUS", "NODE-SUBNET", "NETWORK-POLICIES", "ADDRESS-GROUPS", "APPLIED-TO-GROUPS", "LOCAL-PODS"}
}

func (r AntreaAgentInfoResponse) GetAgentConditionStr() string {
	if r.AgentConditions == nil {
		return ""
	}
	agentCondition := "Healthy"
	for _, cond := range r.AgentConditions {
		if cond.Status == corev1.ConditionUnknown {
			agentCondition = "Unknown"
		}
		if cond.Status == corev1.ConditionFalse {
			return "Unhealthy"
		}
	}
	return agentCondition
}

func (r AntreaAgentInfoResponse) GetTableRow(maxColumnLength int) []string {
	return []string{r.PodRef.Namespace + "/" + r.PodRef.Name,
		r.NodeRef.Name,
		r.GetAgentConditionStr(),
		common.GenerateTableElementWithSummary(r.NodeSubnet, maxColumnLength),
		common.Int32ToString(r.NetworkPolicyControllerInfo.NetworkPolicyNum),
		common.Int32ToString(r.NetworkPolicyControllerInfo.AddressGroupNum),
		common.Int32ToString(r.NetworkPolicyControllerInfo.AppliedToGroupNum),
		common.Int32ToString(r.LocalPodNum)}
}
