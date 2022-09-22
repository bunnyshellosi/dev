package config

type Configuration struct {
	Sync *Sync `yaml:",omitempty"`
}

func NewConfiguration() *Configuration {
	return &Configuration{}
}

func (c *Configuration) WithSync(sync *Sync) *Configuration {
	c.Sync = sync
	return c
}
