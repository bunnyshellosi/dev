package remote

import (
	"github.com/spf13/cobra"

	"bunnyshell.com/dev/pkg/k8s"
	"bunnyshell.com/dev/pkg/remote"
)

func init() {
	var (
		namespaceName  string
		deploymentName string
	)

	command := &cobra.Command{
		Use: "down",
		RunE: func(_ *cobra.Command, _ []string) error {
			remoteDevelopment := remote.NewRemoteDevelopment()
			remoteDevelopment.WithKubernetesClient(k8s.GetKubeConfigFilePath())

			// input
			if namespaceName != "" {
				remoteDevelopment.WithNamespaceName(namespaceName)
			} else if err := remoteDevelopment.SelectNamespace(); err != nil {
				return err
			}

			if deploymentName != "" {
				remoteDevelopment.WithDeploymentName(deploymentName)
			} else if err := remoteDevelopment.SelectDeployment(); err != nil {
				return err
			}

			return remoteDevelopment.Down()
		},
	}

	command.Flags().StringVarP(&namespaceName, "namespace", "n", "", "Kubernetes Namespace")
	command.Flags().StringVarP(&deploymentName, "deployment", "d", "", "Kubernetes Deployment")

	mainCmd.AddCommand(command)
}
