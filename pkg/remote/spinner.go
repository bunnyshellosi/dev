package remote

func (r *RemoteDevelopment) StartSpinner(suffix string) {
	if suffix != "" {
		r.spinner.Suffix = suffix
	}

	r.spinner.Start()
}

func (r *RemoteDevelopment) StopSpinner() {
	r.spinner.Stop()
}
