package container

import (
	"fmt"
	"strings"

	applyCoreV1 "k8s.io/client-go/applyconfigurations/core/v1"
)

const environSplitSize = 2

var ErrInvalidEnvVarDefinition = fmt.Errorf("invalid environment variable definition")

type Environ struct {
	data map[string]string
}

func NewEnviron() *Environ {
	return &Environ{
		data: map[string]string{},
	}
}

func (c *Environ) Set(name, value string) {
	c.data[name] = value
}

func (c *Environ) AddFromDefinition(definition string) error {
	name, value, err := parseDefinition(definition)
	if err != nil {
		return err
	}

	c.Set(name, value)

	return nil
}

func (c *Environ) GetK8SConfiguration() []*applyCoreV1.EnvVarApplyConfiguration {
	list := make([]*applyCoreV1.EnvVarApplyConfiguration, 0, len(c.data))

	for key, value := range c.data {
		list = append(list, &applyCoreV1.EnvVarApplyConfiguration{
			Name:  &key,
			Value: &value,
		})
	}

	return list
}

func parseDefinition(definition string) (key string, value string, err error) {
	parts := strings.SplitN(definition, "=", environSplitSize)
	if len(parts) != environSplitSize {
		return "", "", fmt.Errorf("%w: %s", ErrInvalidEnvVarDefinition, definition)
	}

	return parts[0], parts[1], nil
}
