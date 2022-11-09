package patch

import (
	applyCoreV1 "k8s.io/client-go/applyconfigurations/core/v1"
	applyMetaV1 "k8s.io/client-go/applyconfigurations/meta/v1"
)

type Resource interface {
	WithAnnotations(entries map[string]string) *applyMetaV1.ObjectMetaApplyConfiguration
	WithLabels(entries map[string]string) *applyMetaV1.ObjectMetaApplyConfiguration
	WithSpecTemplate(value *applyCoreV1.PodTemplateSpecApplyConfiguration)
}
