package remote

import (
	"fmt"

	"github.com/spf13/cobra"

	"bunnyshell.com/dev/pkg/k8s"
	k8sTools "bunnyshell.com/dev/pkg/k8s/tools"
	"bunnyshell.com/dev/pkg/remote"
)

func init() {
	var (
		namespaceName  string
		deploymentName string
		containerName  string

		localSyncPath  string
		remoteSyncPath string
	)

	command := &cobra.Command{
		Use: "up",
		RunE: func(_ *cobra.Command, _ []string) error {
			remoteDevelopment := remote.NewRemoteDevelopment()
			remoteDevelopment.WithKubernetesClient(k8s.GetKubeConfigFilePath())

			// input
			if namespaceName != "" {
				namespace, err := remoteDevelopment.KubernetesClient.GetNamespace(namespaceName)
				if err != nil {
					return err
				}
				remoteDevelopment.WithNamespace(namespace)
			} else if err := remoteDevelopment.SelectNamespace(); err != nil {
				return err
			}

			if deploymentName != "" {
				deployment, err := remoteDevelopment.KubernetesClient.GetDeployment(
					remoteDevelopment.Namespace.GetName(),
					deploymentName,
				)
				if err != nil {
					return err
				}
				remoteDevelopment.WithDeployment(deployment)
			} else if err := remoteDevelopment.SelectDeployment(); err != nil {
				return err
			}

			if containerName != "" {
				container := k8sTools.GetDeploymentContainerByName(remoteDevelopment.Deployment, containerName)
				if container == nil {
					return fmt.Errorf(
						"the deployment \"%s\" has no container named \"%s\"",
						remoteDevelopment.Deployment.GetName(),
						container.Name,
					)
				}
				remoteDevelopment.WithContainer(container)
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

			if err := remoteDevelopment.StartSSHTerminal(); err != nil {
				return err
			}

			return remoteDevelopment.Wait()
		},
	}

	command.Flags().StringVarP(&namespaceName, "namespace", "n", "", "Kubernetes Namespace")
	command.Flags().StringVarP(&deploymentName, "deployment", "d", "", "Kubernetes Deployment")
	command.Flags().StringVarP(&containerName, "container", "c", "", "Kubernetes Container")
	command.Flags().StringVarP(&localSyncPath, "local-sync-path", "l", "", "Local folder path to sync")
	command.Flags().StringVarP(&remoteSyncPath, "remote-sync-path", "r", "", "Remote folder path to sync")

	mainCmd.AddCommand(command)
}
