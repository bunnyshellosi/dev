package config

type Sync struct {
	Defaults *SyncDefaults `yaml:",omitempty"`
}

func NewSync() *Sync {
	return &Sync{}
}

func (s *Sync) WithDefaults(defaults *SyncDefaults) *Sync {
	s.Defaults = defaults
	return s
}
