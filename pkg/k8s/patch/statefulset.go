package patch

import (
	appsV1 "k8s.io/api/apps/v1"
	applyAppsV1 "k8s.io/client-go/applyconfigurations/apps/v1"
	applyCoreV1 "k8s.io/client-go/applyconfigurations/core/v1"
	applyMetaV1 "k8s.io/client-go/applyconfigurations/meta/v1"
)

type StatefulSetPatchConfiguration struct {
	*applyMetaV1.ObjectMetaApplyConfiguration `json:"metadata,omitempty"`

	Spec *StatefulSetSpecPatchConfiguration `json:"spec,omitempty"`
}

type StatefulSetSpecPatchConfiguration struct {
	Replicas       *int32                                         `json:"replicas,omitempty"`
	Template       *applyCoreV1.PodTemplateSpecApplyConfiguration `json:"template,omitempty"`
	UpdateStrategy *StatefulSetStrategyPatchConfiguration         `json:"strategy,omitempty"`
}

type StatefulSetStrategyPatchConfiguration struct {
	Type *appsV1.StatefulSetUpdateStrategyType `json:"type,omitempty"`

	// allow "RollingUpdate: null" propagation(remove omitempty)
	RollingUpdate *applyAppsV1.RollingUpdateStatefulSetStrategyApplyConfiguration `json:"rollingUpdate"`
}

func (s *StatefulSetPatchConfiguration) WithSpecTemplate(value *applyCoreV1.PodTemplateSpecApplyConfiguration) {
	s.Spec.Template = value
}
