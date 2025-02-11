package tools

import (
	"fmt"

	appsV1 "k8s.io/api/apps/v1"
	coreV1 "k8s.io/api/core/v1"
)

func FilterContainerByName(containers []coreV1.Container, containerName string) (*coreV1.Container, error) {
	for _, item := range containers {
		if item.Name == containerName {
			return item.DeepCopy(), nil
		}
	}

	return nil, fmt.Errorf("container \"%s\" not found", containerName)
}

func GetDeploymentContainers(deployment *appsV1.Deployment) []coreV1.Container {
	return deployment.Spec.Template.Spec.Containers
}

func GetDeploymentContainerByName(deployment *appsV1.Deployment, containerName string) (*coreV1.Container, error) {
	containers := GetDeploymentContainers(deployment)
	return FilterContainerByName(containers, containerName)
}

func GetStatefulSetContainers(statefulSet *appsV1.StatefulSet) []coreV1.Container {
	return statefulSet.Spec.Template.Spec.Containers
}

func GetStatefulSetContainerByName(statefulSet *appsV1.StatefulSet, containerName string) (*coreV1.Container, error) {
	containers := GetStatefulSetContainers(statefulSet)
	return FilterContainerByName(containers, containerName)
}

func GetDaemonSetContainers(daemonSet *appsV1.DaemonSet) []coreV1.Container {
	return daemonSet.Spec.Template.Spec.Containers
}

func GetDaemonSetContainerByName(daemonSet *appsV1.DaemonSet, containerName string) (*coreV1.Container, error) {
	containers := GetDaemonSetContainers(daemonSet)
	return FilterContainerByName(containers, containerName)
}

func GetDeploymentInitContainers(deployment *appsV1.Deployment) []coreV1.Container {
	return deployment.Spec.Template.Spec.InitContainers
}

func GetDeploymentInitContainerByName(deployment *appsV1.Deployment, initContainerName string) (*coreV1.Container, error) {
	containers := GetDeploymentInitContainers(deployment)
	return FilterContainerByName(containers, initContainerName)
}

func GetStatefulSetInitContainers(statefulSet *appsV1.StatefulSet) []coreV1.Container {
	return statefulSet.Spec.Template.Spec.InitContainers
}

func GetStatefulSetInitContainerByName(statefulSet *appsV1.StatefulSet, initContainerName string) (*coreV1.Container, error) {
	containers := GetStatefulSetInitContainers(statefulSet)
	return FilterContainerByName(containers, initContainerName)
}

func GetDaemonSetInitContainers(daemonSet *appsV1.DaemonSet) []coreV1.Container {
	return daemonSet.Spec.Template.Spec.InitContainers
}

func GetDaemonSetInitContainerByName(daemonSet *appsV1.DaemonSet, initContainerName string) (*coreV1.Container, error) {
	containers := GetDaemonSetInitContainers(daemonSet)
	return FilterContainerByName(containers, initContainerName)
}