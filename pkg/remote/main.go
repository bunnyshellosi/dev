package remote

import (
	"fmt"
	"os"
	"os/signal"

	"bunnyshell.com/dev/pkg/util"
)

func (r *RemoteDevelopment) CanUp() error {
    resource, err := r.getResource()
   	if err != nil {
   		return err
   	}

    labels := resource.GetLabels()
    if active, found := labels[DebugMetadataActive]; found {
        if active == "true" {
            return fmt.Errorf("cannot start remote-development session, Pod already in a debug session")
        }
    }

    return nil
}

func (r *RemoteDevelopment) Up() error {
	if err := r.ensureSSHKeys(); err != nil {
		return err
	}

	if err := r.ensureMutagen(); err != nil {
		return err
	}

	if err := r.ensureSecret(); err != nil {
		return err
	}

	if err := r.ensurePVC(); err != nil {
		return err
	}

	if err := r.prepareResource(); err != nil {
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

	if err := r.startSSHTunnels(); err != nil {
		return err
	}

	return r.startMutagenSession()
}

func (r *RemoteDevelopment) Down() error {
	if err := r.restoreDeployment(); err != nil {
		return err
	}

	if err := r.deletePVC(); err != nil {
		return err
	}

	return r.terminateMutagenDaemon()
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
	case <-r.stopChannel:
		return nil
	}
}

func (r *RemoteDevelopment) Close() {
	r.terminateMutagenSession()

	// close ssh tunnels
	for i := range r.sshTunnels {
		r.sshTunnels[i].Stop()
	}

	// close k8s remote ssh portforwarding
	if r.sshPortForwarder != nil {
		r.sshPortForwarder.Close()
	}

	// close cli command
	if r.stopChannel != nil {
		close(r.stopChannel)
	}
}
