package remote

import (
	"fmt"
	"os"

	"bunnyshell.com/dev/pkg/util"
)

var (
	ErrNoNamespaces   = fmt.Errorf("no namespaces available")
	ErrNoDeployments  = fmt.Errorf("no deployments available")
	ErrNoStatefulSets = fmt.Errorf("no statefulsets available")

	ErrNoNamespaceSelected = fmt.Errorf("no namespace selected")
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

func (r *RemoteDevelopment) SelectResourceType() error {
	resourceTypeMap := map[string]ResourceType{
		string(Deployment):  Deployment,
		string(StatefulSet): StatefulSet,
	}

	items := []string{string(Deployment), string(StatefulSet)}
	resourceType, err := util.Select("Select resource type", items)
	if err != nil {
		return err
	}

	r.WithResourceType(resourceTypeMap[resourceType])
	return nil
}

func (r *RemoteDevelopment) SelectResource() error {
	switch r.resourceType {
	case Deployment:
		return r.SelectDeployment()
	case StatefulSet:
		return r.SelectStatefulSet()
	}

	return nil
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

func (r *RemoteDevelopment) SelectContainer() error {
	podContainers, err := r.getResourceContainers()
	if err != nil {
		return err
	}
	if len(podContainers) == 1 {
		r.WithContainer(podContainers[0].DeepCopy())
		return nil
	}

	items := []string{}
	for _, item := range podContainers {
		items = append(items, item.Name)
	}

	container, err := util.Select("Select container", items)
	if err != nil {
		return err
	}

	for _, item := range podContainers {
		if item.Name != container {
			continue
		}

		r.WithContainer(item.DeepCopy())
		return nil
	}

	return nil
}

func (r *RemoteDevelopment) SelectLocalSyncPath() error {
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
