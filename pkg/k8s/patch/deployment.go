package patch

import (
	appsV1 "k8s.io/api/apps/v1"
	applyAppsV1 "k8s.io/client-go/applyconfigurations/apps/v1"
	applyCoreV1 "k8s.io/client-go/applyconfigurations/core/v1"
	applyMetaV1 "k8s.io/client-go/applyconfigurations/meta/v1"
)

type DeploymentPatchConfiguration struct {
	*applyMetaV1.ObjectMetaApplyConfiguration `json:"metadata,omitempty"`

	Spec *DeploymentSpecPatchConfiguration `json:"spec,omitempty"`
}

type DeploymentSpecPatchConfiguration struct {
	Replicas *int32                                         `json:"replicas,omitempty"`
	Template *applyCoreV1.PodTemplateSpecApplyConfiguration `json:"template,omitempty"`
	Strategy *DeploymentStrategyPatchConfiguration          `json:"strategy,omitempty"`
}

type DeploymentStrategyPatchConfiguration struct {
	Type          *appsV1.DeploymentStrategyType                         `json:"type,omitempty"`
	RollingUpdate *applyAppsV1.RollingUpdateDeploymentApplyConfiguration `json:"rollingUpdate"`
}
