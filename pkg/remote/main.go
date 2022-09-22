package remote

import (
	"fmt"
	"os"
	"os/signal"

	"bunnyshell.com/dev/pkg/util"
)

func (r *RemoteDevelopment) Up() error {
	if err := r.ensureSSHKeys(); err != nil {
		return err
	}

	if err := r.ensureMutagen(); err != nil {
		return err
	}

	if err := r.prepareDeployment(); err != nil {
		return err
	}

	if err := r.waitPodReady(); err != nil {
		return err
	}

	if err := r.ensureRemoteSSHPortForward(); err != nil {
		return err
	}

	if err := r.ensureSSHConfigEntry(); err != nil {
		return err
	}

	return r.startMutagenSession()
}

func (r *RemoteDevelopment) Down() error {
	return nil
}

func (r *RemoteDevelopment) Wait() error {
	// close channels on cli signal interrupt
	signalTermination := make(chan os.Signal, 1)
	signal.Notify(signalTermination, util.TerminationSignals...)
	defer signal.Stop(signalTermination)

	select {
	case sig := <-signalTermination:
		r.Close()
		return fmt.Errorf("terminated by signal: %s", sig)
	case <-r.StopChannel:
		return nil
	}
}

func (r *RemoteDevelopment) Close() {
	r.terminateMutagenSession()

	// close k8s remote ssh portforwarding
	if r.RemoteSSHForwarder != nil {
		r.RemoteSSHForwarder.Close()
	}

	// close cli command
	if r.StopChannel != nil {
		close(r.StopChannel)
	}
}
