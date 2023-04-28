package container

import (
	applyCoreV1 "k8s.io/client-go/applyconfigurations/core/v1"
)

type Resources struct {
	limits   ResourceList
	requests ResourceList
}

func NewResources() *Resources {
	return &Resources{
		limits:   *NewList(),
		requests: *NewList(),
	}
}

func (c *Resources) SetLimitsCPU(value string) error {
	return c.limits.SetCPU(value)
}

func (c *Resources) SetLimitsMemory(value string) error {
	return c.limits.SetMemory(value)
}

func (c *Resources) SetRequestsCPU(value string) error {
	return c.requests.SetCPU(value)
}

func (c *Resources) SetRequestsMemory(value string) error {
	return c.requests.SetMemory(value)
}

func (c *Resources) GetK8SConfiguration() *applyCoreV1.ResourceRequirementsApplyConfiguration {
	limits := c.limits.GetK8SConfiguration()
	requests := c.requests.GetK8SConfiguration()

	if limits == nil && requests == nil {
		return nil
	}

	resourceRequirements := applyCoreV1.ResourceRequirements()
	if limits != nil {
		resourceRequirements.WithLimits(limits)
	}

	if requests != nil {
		resourceRequirements.WithRequests(requests)
	}

	return resourceRequirements
}
