package config

type SyncDefaults struct {
	Mode   Mode    `yaml:",omitempty"`
	Ignore *Ignore `yaml:",omitempty"`
}

func NewSyncDefaults() *SyncDefaults {
	return &SyncDefaults{}
}

func (d *SyncDefaults) WithMode(mode Mode) *SyncDefaults {
	d.Mode = mode
	return d
}

func (d *SyncDefaults) WithIgnore(ignore *Ignore) *SyncDefaults {
	d.Ignore = ignore
	return d
}
