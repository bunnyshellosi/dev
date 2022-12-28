package remote

import (
	"github.com/spf13/cobra"
	"github.com/thediveo/enumflag/v2"

	"bunnyshell.com/dev/pkg/k8s"
	mutagenConfig "bunnyshell.com/dev/pkg/mutagen/config"
	"bunnyshell.com/dev/pkg/remote"
)

// +enum
type SyncMode enumflag.Flag

const (
	None SyncMode = iota
	TwoWaySafe
	TwoWayResolved
	OneWaySafe
	OneWayReplica
)

var SyncModeIds = map[SyncMode][]string{
	None:           {string(mutagenConfig.None)},
	TwoWaySafe:     {string(mutagenConfig.TwoWaySafe)},
	TwoWayResolved: {string(mutagenConfig.TwoWayResolved)},
	OneWaySafe:     {string(mutagenConfig.OneWaySafe)},
	OneWayReplica:  {string(mutagenConfig.OneWayReplica)},
}

var SyncModeToMutagenMode = map[SyncMode]mutagenConfig.Mode{
	None:           mutagenConfig.None,
	TwoWaySafe:     mutagenConfig.TwoWaySafe,
	TwoWayResolved: mutagenConfig.TwoWayResolved,
	OneWaySafe:     mutagenConfig.OneWaySafe,
	OneWayReplica:  mutagenConfig.OneWayReplica,
}

func init() {
	var (
		namespaceName   string
		deploymentName  string
		statefulSetName string
		daemonSetName   string
		containerName   string

		syncMode       SyncMode = TwoWayResolved
		localSyncPath  string
		remoteSyncPath string

		portMappings []string

		waitTimeout int
		noTTY       bool
	)

	command := &cobra.Command{
		Use: "up",
		RunE: func(_ *cobra.Command, _ []string) error {
			remoteDevelopment := remote.NewRemoteDevelopment()
			remoteDevelopment.
				WithKubernetesClient(k8s.GetKubeConfigFilePath()).
				WithWaitTimeout(int64(waitTimeout)).
				WithSyncMode(SyncModeToMutagenMode[syncMode])

			// wizard
			if namespaceName != "" {
				remoteDevelopment.WithNamespaceName(namespaceName)
			} else if err := remoteDevelopment.SelectNamespace(); err != nil {
				return err
			}

			if deploymentName != "" {
				remoteDevelopment.WithDeploymentName(deploymentName)
			} else if statefulSetName != "" {
				remoteDevelopment.WithStatefulSetName(statefulSetName)
			} else if daemonSetName != "" {
				remoteDevelopment.WithDaemonSetName(daemonSetName)
			} else {
				if err := remoteDevelopment.SelectResource(); err != nil {
					return err
				}
			}

			if containerName != "" {
				remoteDevelopment.WithContainerName(containerName)
			} else if err := remoteDevelopment.SelectContainer(); err != nil {
				return err
			}

			if localSyncPath != "" {
				remoteDevelopment.WithLocalSyncPath(localSyncPath)
			} else if err := remoteDevelopment.SelectLocalSyncPath(); err != nil {
				return err
			}

			if remoteSyncPath != "" {
				remoteDevelopment.WithRemoteSyncPath(remoteSyncPath)
			} else if err := remoteDevelopment.SelectRemoteSyncPath(); err != nil {
				return err
			}

			if len(portMappings) > 0 {
				if err := remoteDevelopment.PrepareSSHTunnels(portMappings); err != nil {
					return err
				}
			}

			// bootstrap
			if err := remoteDevelopment.Up(); err != nil {
				return err
			}

			// start
			if !noTTY {
				if err := remoteDevelopment.StartSSHTerminal(); err != nil {
					return err
				}
			}

			return remoteDevelopment.Wait()
		},
	}

	command.Flags().StringVarP(&namespaceName, "namespace", "n", "", "Kubernetes Namespace")
	command.Flags().StringVarP(&deploymentName, "deployment", "d", "", "Kubernetes Deployment")
	command.Flags().StringVarP(&statefulSetName, "statefulset", "s", "", "Kubernetes StatefulSet")
	command.Flags().StringVarP(&daemonSetName, "daemonset", "t", "", "Kubernetes DaemonSet")
	command.Flags().StringVar(&containerName, "container", "", "Kubernetes Container")
	command.Flags().StringVarP(&localSyncPath, "local-sync-path", "l", "", "Local folder path to sync")
	command.Flags().StringVarP(&remoteSyncPath, "remote-sync-path", "r", "", "Remote folder path to sync")
	command.Flags().StringSliceVarP(&portMappings, "portforward", "p", []string{}, "Port forward: '8080>3000'\nReverse port forward: '9003<9003'\nComma separated: '8080>3000,9003<9003'")
	command.Flags().IntVarP(&waitTimeout, "wait-timeout", "w", 120, "Time to wait for pod to be ready")
	command.Flags().BoolVar(&noTTY, "no-tty", false, "Start remote development with no ssh terminal")
	command.Flags().Var(
		enumflag.New(&syncMode, "sync-mode", SyncModeIds, enumflag.EnumCaseSensitive),
		"sync-mode",
		"Mutagen sync mode.\nAvailable sync modes: none, two-way-safe, two-way-resolved, one-way-safe, one-way-replica.\n\"none\" sync mode disables mutagen.",
	)

	mainCmd.AddCommand(command)
}
