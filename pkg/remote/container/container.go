package container

import (
	applyCoreV1 "k8s.io/client-go/applyconfigurations/core/v1"
)

type Config struct {
	Environ

	Resources

	Command []string
}

func NewConfig() *Config {
	return &Config{
		Environ:   *NewEnviron(),
		Resources: *NewResources(),

		Command: []string{},
	}
}

func (c *Config) ApplyTo(container *applyCoreV1.ContainerApplyConfiguration) {
	// This is a part of a patch operation
	container.WithEnv(c.Environ.GetK8SConfiguration()...)

	resources := c.Resources.GetK8SConfiguration()
	if resources != nil {
		container.WithResources(resources)
	}

	if len(c.Command) > 0 {
		container.WithArgs(c.Command...)
	}
}
