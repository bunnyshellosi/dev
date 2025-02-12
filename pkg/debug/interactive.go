package debug


import (
	"fmt"
	"strings"

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

func (d *DebugComponent) SelectNamespace() error {
	namespaces, err := d.kubernetesClient.ListNamespaces()
	if err != nil {
		return err
	}

	if len(namespaces.Items) == 0 {
		return ErrNoNamespaces
	}

	if len(namespaces.Items) == 1 && d.AutoSelectSingleResource {
		d.namespace = namespaces.Items[0].DeepCopy()
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

		d.WithNamespace(item.DeepCopy())
		return nil
	}

	return nil
}

func (d *DebugComponent) SelectResource() error {
	availableResources, err := d.getAvailableResourceFromNamespace(d.namespace.GetName())
	if err != nil {
		return err
	}

	if len(availableResources) == 0 {
		return ErrNoResources
	}

	if len(availableResources) == 1 && d.AutoSelectSingleResource {
		d.WithResource(availableResources[0])

		return nil
	}

	selectItems := []string{}
	resourcesItemsMap := map[string]Resource{}
	for _, resourceItem := range availableResources {
		resourceType, err := d.getResourceType(resourceItem)
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

	d.WithResource(resourcesItemsMap[selectedResourceItemLabel])
	return nil
}

func (d *DebugComponent) getAvailableResourceFromNamespace(namespace string) ([]Resource, error) {
	availableResources := []Resource{}

	deployments, err := d.kubernetesClient.ListDeployments(namespace)
	if err != nil {
		return nil, err
	}
	for _, deploymentItem := range deployments.Items {
		item := deploymentItem
		availableResources = append(availableResources, &item)
	}

	statefulsets, err := d.kubernetesClient.ListStatefulSets(namespace)
	if err != nil {
		return nil, err
	}
	for _, statefulsetItem := range statefulsets.Items {
		item := statefulsetItem
		availableResources = append(availableResources, &item)
	}

	daemonsets, err := d.kubernetesClient.ListDaemonSets(namespace)
	if err != nil {
		return nil, err
	}
	for _, daemonsetItem := range daemonsets.Items {
		item := daemonsetItem
		availableResources = append(availableResources, &item)
	}

	return availableResources, nil
}

func (d *DebugComponent) SelectDeployment() error {
	if d.namespace == nil {
		return ErrNoNamespaceSelected
	}

	deployments, err := d.kubernetesClient.ListDeployments(d.namespace.GetName())
	if err != nil {
		return err
	}

	if len(deployments.Items) == 0 {
		return ErrNoDeployments
	}

	if len(deployments.Items) == 1 && d.AutoSelectSingleResource {
		d.WithDeployment(deployments.Items[0].DeepCopy())

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

		d.WithDeployment(item.DeepCopy())
		return nil
	}

	return nil
}

func (d *DebugComponent) SelectStatefulSet() error {
	if d.namespace == nil {
		return ErrNoNamespaceSelected
	}

	statefulSets, err := d.kubernetesClient.ListStatefulSets(d.namespace.GetName())
	if err != nil {
		return err
	}

	if len(statefulSets.Items) == 0 {
		return ErrNoStatefulSets
	}

	if len(statefulSets.Items) == 1 && d.AutoSelectSingleResource {
		d.WithStatefulSet(statefulSets.Items[0].DeepCopy())

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

		d.WithStatefulSet(item.DeepCopy())
		return nil
	}

	return nil
}

func (d *DebugComponent) SelectDaemonSet() error {
	if d.namespace == nil {
		return ErrNoNamespaceSelected
	}

	daemonSets, err := d.kubernetesClient.ListDaemonSets(d.namespace.GetName())
	if err != nil {
		return err
	}

	if len(daemonSets.Items) == 0 {
		return ErrNoDaemonSets
	}

	if len(daemonSets.Items) == 1 && d.AutoSelectSingleResource {
		d.WithDaemonSet(daemonSets.Items[0].DeepCopy())

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

		d.WithDaemonSet(item.DeepCopy())
		return nil
	}

	return nil
}

func (d *DebugComponent) SelectContainer() error {
	containers, err := d.getResourceContainers()
	if err != nil {
		return err
	}


	initContainers, err := d.getResourceInitContainers()
    if err != nil {
        return err
    }

    initContainers = d.excludeRestrictedInitContainers(initContainers)

    // Pods created from Bunnyshell have unique names in the unified initContainers and containers collection
	if d.ContainerName != "" {
		for _, container := range containers {
			if container.Name == d.ContainerName {
				d.WithContainer(container.DeepCopy())
				return nil
			}
		}

		for _, initContainer := range initContainers {
            if initContainer.Name == d.ContainerName {
                d.WithInitContainer(initContainer.DeepCopy())
                return nil
            }
        }

		return ErrContainerNotFound
	}

	container, isInit, err := d.selectContainer(containers, initContainers)
	if err != nil {
		return err
	}

    if isInit {
        d.WithInitContainer(container.DeepCopy())
    } else {
        d.WithContainer(container.DeepCopy())
    }

	return nil
}

func (d *DebugComponent) selectContainer(containers []coreV1.Container, initContainers []coreV1.Container) (*coreV1.Container, bool, error) {
	if (len(containers) + len(initContainers)) == 1 && d.AutoSelectSingleResource {
	    if len(containers) == 1 {
		    return &containers[0], false, nil
		}

        return &initContainers[0], true, nil
	}

    initPrefix := "init - "
	items := []string{}
	for _, item := range initContainers {
        items = append(items, initPrefix + item.Name)
    }
	for _, item := range containers {
        items = append(items, item.Name)
    }

	container, err := util.Select("Select container", items)
	if err != nil {
		return nil, false, err
	}

    for _, item := range initContainers {
        if initPrefix + item.Name == container {
            return &item, true, nil
        }
    }
	for _, item := range containers {
		if item.Name == container {
			return &item, false, nil
		}
	}

	return nil, false, ErrContainerNotFound
}

func (d *DebugComponent) excludeRestrictedInitContainers(containers []coreV1.Container) []coreV1.Container {
    restrictedNames := []string{"bns-volume-permissions"}

	// Convert restricted names to a map for O(1) lookups
	restrictedMap := make(map[string]struct{})
	for _, name := range restrictedNames {
		restrictedMap[name] = struct{}{}
	}

	var result []coreV1.Container
	for _, container := range containers {
		if _, isRestricted := restrictedMap[container.Name]; isRestricted {
			continue
		}

        if strings.HasPrefix(container.Name, "init-shared-path-") {
            continue
        }

        result = append(result, container)
	}

	return result
}
