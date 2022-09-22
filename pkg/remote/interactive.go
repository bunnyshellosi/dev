package remote

import (
	"fmt"
	"os"

	"bunnyshell.com/dev/pkg/util"
)

var (
	ErrNoNamespaces  = fmt.Errorf("no namespaces available")
	ErrNoDeployments = fmt.Errorf("no namespaces available")

	ErrNoNamespaceSelected  = fmt.Errorf("no namespace selected")
	ErrNoDeploymentSelected = fmt.Errorf("no deployment selected")
)

func (r *RemoteDevelopment) SelectNamespace() error {
	namespaces, err := r.KubernetesClient.ListNamespaces()
	if err != nil {
		return err
	}

	if len(namespaces.Items) == 0 {
		return ErrNoNamespaces
	}

	if len(namespaces.Items) == 1 {
		r.Namespace = namespaces.Items[0].DeepCopy()
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

func (r *RemoteDevelopment) SelectDeployment() error {
	if r.Namespace == nil {
		return ErrNoNamespaceSelected
	}

	deployments, err := r.KubernetesClient.ListDeployments(r.Namespace.GetName())
	if err != nil {
		return err
	}

	if len(deployments.Items) == 0 {
		return ErrNoDeployments
	}

	if len(deployments.Items) == 1 {
		r.Deployment = deployments.Items[0].DeepCopy()
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

func (r *RemoteDevelopment) SelectContainer() error {
	if r.Deployment == nil {
		return ErrNoDeploymentSelected
	}

	podContainers := r.Deployment.Spec.Template.Spec.Containers
	if len(podContainers) == 1 {
		r.Container = podContainers[0].DeepCopy()
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
	syncPath, err := util.AskPath("Remote Path", "", util.IsDirectoryValidator)
	if err != nil {
		return err
	}

	r.WithRemoteSyncPath(syncPath)
	return nil
}
