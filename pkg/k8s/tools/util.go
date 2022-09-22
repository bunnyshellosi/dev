package tools

import (
	appsV1 "k8s.io/api/apps/v1"
	coreV1 "k8s.io/api/core/v1"
)

func GetDeploymentContainers(deployment *appsV1.Deployment) []coreV1.Container {
	return deployment.Spec.Template.Spec.Containers
}

func GetDeploymentContainerByName(deployment *appsV1.Deployment, containerName string) *coreV1.Container {
	containers := GetDeploymentContainers(deployment)
	for _, item := range containers {
		if item.Name == containerName {
			return item.DeepCopy()
		}
	}
	return nil
}
