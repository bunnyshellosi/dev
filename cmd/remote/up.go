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
		containerName   string

		localSyncPath  string
		remoteSyncPath string
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
			} else {
				if err := remoteDevelopment.SelectResourceType(); err != nil {
					return err
				}

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
	command.Flags().StringVar(&containerName, "container", "", "Kubernetes Container")
	command.Flags().StringVarP(&localSyncPath, "local-sync-path", "l", "", "Local folder path to sync")
	command.Flags().StringVarP(&remoteSyncPath, "remote-sync-path", "r", "", "Remote folder path to sync")

	mainCmd.AddCommand(command)
}
