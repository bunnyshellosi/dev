package debug

func (d *DebugComponent) StartSpinner(suffix string) {
	if suffix != "" {
		d.spinner.Suffix = suffix
	}

	d.spinner.Start()
}

func (d *DebugComponent) StopSpinner() {
	d.spinner.Stop()
}
