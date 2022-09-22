package remote

func (r *RemoteDevelopment) StartSpinner(suffix string) {
	if suffix != "" {
		r.Spinner.Suffix = suffix
	}

	r.Spinner.Start()
}

func (r *RemoteDevelopment) StopSpinner() {
	r.Spinner.Stop()
}
