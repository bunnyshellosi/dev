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
			} else if statefulSetName != "" {
				remoteDevelopment.WithStatefulSetName(statefulSetName)
			} else if daemonSetName != "" {
				remoteDevelopment.WithDaemonSetName(daemonSetName)
			} else {
				if err := remoteDevelopment.SelectResourceType(); err != nil {
					return err
				}

				if err := remoteDevelopment.SelectResource(); err != nil {
					return err
				}
			}

			return remoteDevelopment.Down()
		},
	}

	command.Flags().StringVarP(&namespaceName, "namespace", "n", "", "Kubernetes Namespace")
	command.Flags().StringVarP(&deploymentName, "deployment", "d", "", "Kubernetes Deployment")
	command.Flags().StringVarP(&statefulSetName, "statefulset", "s", "", "Kubernetes StatefulSet")
	command.Flags().StringVar(&daemonSetName, "daemonset", "s", "Kubernetes DaemonSet")

	mainCmd.AddCommand(command)
}
