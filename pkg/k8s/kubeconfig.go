package k8s

import (
	"os"

	"k8s.io/client-go/tools/clientcmd"
)

func GetKubeConfigFilePath() string {
	kubeConfigPath := os.Getenv(clientcmd.RecommendedConfigPathEnvVar)
	if kubeConfigPath != "" {
		return kubeConfigPath
	}

	return clientcmd.RecommendedHomeFile
}
