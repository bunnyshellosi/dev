package container

import (
	coreV1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

type ResourceList struct {
	configuration coreV1.ResourceList
}

func NewList() *ResourceList {
	return &ResourceList{
		configuration: coreV1.ResourceList{},
	}
}

func (c *ResourceList) SetCPU(value string) error {
	return c.set(coreV1.ResourceCPU, value)
}

func (c *ResourceList) SetMemory(value string) error {
	return c.set(coreV1.ResourceMemory, value)
}

func (c *ResourceList) GetK8SConfiguration() coreV1.ResourceList {
	if len(c.configuration) == 0 {
		return nil
	}

	return c.configuration
}

func (c *ResourceList) set(name coreV1.ResourceName, value string) error {
	if value == "" {
		return nil
	}

	q, err := resource.ParseQuantity(value)
	if err != nil {
		return err
	}

	c.configuration[name] = q

	return nil
}
