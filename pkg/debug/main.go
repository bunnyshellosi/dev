package debug

import (
	"fmt"
	"os"
	"os/signal"

	"bunnyshell.com/dev/pkg/util"
)

func (d *DebugComponent) CanUp(forceRecreateResource bool) error {
    resource, err := d.getResource()
   	if err != nil {
   		return err
   	}

    labels := resource.GetLabels()
    if active, found := labels[RemoteDevMetadataActive]; found {
        if active == "true" {
            return fmt.Errorf("Cannot start debug session, Pod already in a remote-development session")
        }
    }

    if active, found := labels[MetadataActive]; found {
        if active == "true" {
            annotations := resource.GetAnnotations()
            if containerName, found := annotations[MetadataContainer]; found {
                if (containerName == d.container.Name) {
                    d.shouldPrepareResource = forceRecreateResource

                    return nil;
                }

                if forceRecreateResource {
                    d.shouldPrepareResource = true

                    return nil;
                } else {
                    return fmt.Errorf("Cannot start debug session, Pod already in another debug session on container %s.\nRun \"bns debug down\" command then try again.", containerName)
                }
            }
        }
    }

    d.shouldPrepareResource = true

    return nil
}

func (d *DebugComponent) Up() error {
    if (d.shouldPrepareResource) {
        if err := d.prepareResource(); err != nil {
            return err
        }
    } else {
        fmt.Print("Skip recreating Pod\n")
    }

	if err := d.waitPodReady(); err != nil {
		return err
	}

	return nil
}

func (d *DebugComponent) Down() error {
	if err := d.restoreDeployment(); err != nil {
		return err
	}

    return nil
}

func (d *DebugComponent) Wait() error {
	// close channels on cli signal interrupt
	signalTermination := make(chan os.Signal, 1)
	signal.Notify(signalTermination, util.TerminationSignals...)
	defer signal.Stop(signalTermination)

	select {
	case sig := <-signalTermination:
		d.Close()
		return fmt.Errorf("terminated by signal: %s", sig)
	case <-d.stopChannel:
		return nil
	}
}

func (d *DebugComponent) Close() {
	// close cli command
	if d.stopChannel != nil {
		close(d.stopChannel)
	}
}
