package config

type Ignore struct {
	Paths []string `yaml:",omitempty"`
	Vcs   *bool    `yaml:",omitempty"`
}

func NewIgnore() *Ignore {
	return &Ignore{}
}

func (i *Ignore) WithVCS(vcs *bool) *Ignore {
	i.Vcs = vcs
	return i
}

func (i *Ignore) WithPaths(paths []string) *Ignore {
	i.Paths = append(i.Paths, paths...)
	return i
}
