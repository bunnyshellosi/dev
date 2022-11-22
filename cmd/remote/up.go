package remote

import (
	"github.com/spf13/cobra"

	"bunnyshell.com/dev/pkg/k8s"
	"bunnyshell.com/dev/pkg/remote"
)

func init() {
	var (
		namespaceName   string
		deploymentName  string
		statefulSetName string
		daemonSetName   string
		containerName   string

		localSyncPath  string
		remoteSyncPath string

		portMappings []string
	)

	command := &cobra.Command{
		Use: "up",
		RunE: func(_ *cobra.Command, _ []string) error {
			remoteDevelopment := remote.NewRemoteDevelopment()
			remoteDevelopment.WithKubernetesClient(k8s.GetKubeConfigFilePath())

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
			if err := remoteDevelopment.StartSSHTerminal(); err != nil {
				return err
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

	mainCmd.AddCommand(command)
}
