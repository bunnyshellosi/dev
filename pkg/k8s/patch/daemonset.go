package patch

import (
	appsV1 "k8s.io/api/apps/v1"
	applyAppsV1 "k8s.io/client-go/applyconfigurations/apps/v1"
	applyCoreV1 "k8s.io/client-go/applyconfigurations/core/v1"
	applyMetaV1 "k8s.io/client-go/applyconfigurations/meta/v1"
)

type DaemonSetPatchConfiguration struct {
	*applyMetaV1.ObjectMetaApplyConfiguration `json:"metadata,omitempty"`

	Spec *DaemonSetSpecPatchConfiguration `json:"spec,omitempty"`
}

type DaemonSetSpecPatchConfiguration struct {
	Template       *applyCoreV1.PodTemplateSpecApplyConfiguration `json:"template,omitempty"`
	UpdateStrategy *DaemonSetStrategyPatchConfiguration           `json:"strategy,omitempty"`
}

type DaemonSetStrategyPatchConfiguration struct {
	Type *appsV1.DaemonSetUpdateStrategyType `json:"type,omitempty"`

	// allow "RollingUpdate: null" propagation(remove omitempty)
	RollingUpdate *applyAppsV1.RollingUpdateDaemonSetApplyConfiguration `json:"rollingUpdate"`
}

func (s *DaemonSetPatchConfiguration) WithSpecTemplate(value *applyCoreV1.PodTemplateSpecApplyConfiguration) {
	s.Spec.Template = value
}
