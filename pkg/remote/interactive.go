package remote

import (
	"fmt"
	"os"

	mutagenConfig "bunnyshell.com/dev/pkg/mutagen/config"
	coreV1 "k8s.io/api/core/v1"

	"bunnyshell.com/dev/pkg/util"
)

var (
	ErrNoNamespaces = fmt.Errorf("no namespaces available")

	ErrNoResources = fmt.Errorf("no resources available")

	ErrNoDeployments  = fmt.Errorf("no deployments available")
	ErrNoStatefulSets = fmt.Errorf("no statefulsets available")
	ErrNoDaemonSets   = fmt.Errorf("no daemonset available")

	ErrNoNamespaceSelected = fmt.Errorf("no namespace selected")
	ErrNoResourceSelected  = fmt.Errorf("no resource selected")

	ErrContainerNotFound = fmt.Errorf("container not found")
)

func (r *RemoteDevelopment) SelectNamespace() error {
	namespaces, err := r.kubernetesClient.ListNamespaces()
	if err != nil {
		return err
	}

	if len(namespaces.Items) == 0 {
		return ErrNoNamespaces
	}

	if len(namespaces.Items) == 1 {
		r.namespace = namespaces.Items[0].DeepCopy()
		return nil
	}

	items := []string{}
	for _, item := range namespaces.Items {
		items = append(items, item.GetName())
	}

	namespace, err := util.Select("Select namespace", items)
	if err != nil {
		return err
	}

	for _, item := range namespaces.Items {
		if item.GetName() != namespace {
			continue
		}

		r.WithNamespace(item.DeepCopy())
		return nil
	}

	return nil
}

func (r *RemoteDevelopment) SelectResource() error {
	availableResources, err := r.getAvailableResourceFromNamespace(r.namespace.GetName())
	if err != nil {
		return err
	}

	if len(availableResources) == 0 {
		return ErrNoResources
	}

	if len(availableResources) == 1 {
		r.WithResource(availableResources[0])

		return nil
	}

	selectItems := []string{}
	resourcesItemsMap := map[string]Resource{}
	for _, resourceItem := range availableResources {
		resourceType, err := r.getResourceType(resourceItem)
		if err != nil {
			return err
		}

		resourceItemLabel := fmt.Sprintf("%s / %s", resourceType, resourceItem.GetName())
		selectItems = append(selectItems, resourceItemLabel)
		resourcesItemsMap[resourceItemLabel] = resourceItem
	}

	selectedResourceItemLabel, err := util.Select("Select resource", selectItems)
	if err != nil {
		return err
	}

	r.WithResource(resourcesItemsMap[selectedResourceItemLabel])
	return nil
}

func (r *RemoteDevelopment) getAvailableResourceFromNamespace(namespace string) ([]Resource, error) {
	availableResources := []Resource{}

	deployments, err := r.kubernetesClient.ListDeployments(namespace)
	if err != nil {
		return nil, err
	}
	for _, deploymentItem := range deployments.Items {
		item := deploymentItem
		availableResources = append(availableResources, &item)
	}

	statefulsets, err := r.kubernetesClient.ListStatefulSets(namespace)
	if err != nil {
		return nil, err
	}
	for _, statefulsetItem := range statefulsets.Items {
		item := statefulsetItem
		availableResources = append(availableResources, &item)
	}

	daemonsets, err := r.kubernetesClient.ListDaemonSets(namespace)
	if err != nil {
		return nil, err
	}
	for _, daemonsetItem := range daemonsets.Items {
		item := daemonsetItem
		availableResources = append(availableResources, &item)
	}

	return availableResources, nil
}

func (r *RemoteDevelopment) SelectDeployment() error {
	if r.namespace == nil {
		return ErrNoNamespaceSelected
	}

	deployments, err := r.kubernetesClient.ListDeployments(r.namespace.GetName())
	if err != nil {
		return err
	}

	if len(deployments.Items) == 0 {
		return ErrNoDeployments
	}

	if len(deployments.Items) == 1 {
		r.WithDeployment(deployments.Items[0].DeepCopy())

		return nil
	}

	items := []string{}
	for _, item := range deployments.Items {
		items = append(items, item.GetName())
	}

	deployment, err := util.Select("Select deployment", items)
	if err != nil {
		return err
	}

	for _, item := range deployments.Items {
		if item.GetName() != deployment {
			continue
		}

		r.WithDeployment(item.DeepCopy())
		return nil
	}

	return nil
}

func (r *RemoteDevelopment) SelectStatefulSet() error {
	if r.namespace == nil {
		return ErrNoNamespaceSelected
	}

	statefulSets, err := r.kubernetesClient.ListStatefulSets(r.namespace.GetName())
	if err != nil {
		return err
	}

	if len(statefulSets.Items) == 0 {
		return ErrNoStatefulSets
	}

	if len(statefulSets.Items) == 1 {
		r.WithStatefulSet(statefulSets.Items[0].DeepCopy())

		return nil
	}

	items := []string{}
	for _, item := range statefulSets.Items {
		items = append(items, item.GetName())
	}

	statefulSet, err := util.Select("Select statefulset", items)
	if err != nil {
		return err
	}

	for _, item := range statefulSets.Items {
		if item.GetName() != statefulSet {
			continue
		}

		r.WithStatefulSet(item.DeepCopy())
		return nil
	}

	return nil
}

func (r *RemoteDevelopment) SelectDaemonSet() error {
	if r.namespace == nil {
		return ErrNoNamespaceSelected
	}

	daemonSets, err := r.kubernetesClient.ListDaemonSets(r.namespace.GetName())
	if err != nil {
		return err
	}

	if len(daemonSets.Items) == 0 {
		return ErrNoDaemonSets
	}

	if len(daemonSets.Items) == 1 {
		r.WithDaemonSet(daemonSets.Items[0].DeepCopy())

		return nil
	}

	items := []string{}
	for _, item := range daemonSets.Items {
		items = append(items, item.GetName())
	}

	daemonSet, err := util.Select("Select daemonset", items)
	if err != nil {
		return err
	}

	for _, item := range daemonSets.Items {
		if item.GetName() != daemonSet {
			continue
		}

		r.WithDaemonSet(item.DeepCopy())
		return nil
	}

	return nil
}

func (r *RemoteDevelopment) SelectContainer() error {
	containers, err := r.getResourceContainers()
	if err != nil {
		return err
	}

	if r.ContainerName != "" {
		for _, container := range containers {
			if container.Name == r.ContainerName {
				r.WithContainer(container.DeepCopy())
				return nil
			}
		}

		return ErrContainerNotFound
	}

	container, err := r.selectContainer(containers)
	if err != nil {
		return err
	}

	r.WithContainer(container.DeepCopy())

	return nil
}

func (r *RemoteDevelopment) SelectLocalSyncPath() error {
	if r.syncMode == mutagenConfig.None {
		return nil
	}

	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	syncPath, err := util.AskPath("Local Path", cwd, util.IsDirectoryValidator)
	if err != nil {
		return err
	}

	r.WithLocalSyncPath(syncPath)
	return nil
}

func (r *RemoteDevelopment) SelectRemoteSyncPath() error {
	syncPath, err := util.Ask("Remote Path", "")
	if err != nil {
		return err
	}

	r.WithRemoteSyncPath(syncPath)
	return nil
}

func (r *RemoteDevelopment) selectContainer(containers []coreV1.Container) (*coreV1.Container, error) {
	if len(containers) == 1 {
		return &containers[0], nil
	}

	items := []string{}
	for _, item := range containers {
		items = append(items, item.Name)
	}

	container, err := util.Select("Select container", items)
	if err != nil {
		return nil, err
	}

	for _, item := range containers {
		if item.Name == container {
			return &item, nil
		}
	}

	return nil, ErrContainerNotFound
}
