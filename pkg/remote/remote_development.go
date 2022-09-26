package remote

import (
	"fmt"
	"time"

	"bunnyshell.com/dev/pkg/k8s"
	k8sTools "bunnyshell.com/dev/pkg/k8s/tools"
	"bunnyshell.com/dev/pkg/util"

	"github.com/briandowns/spinner"
	appsV1 "k8s.io/api/apps/v1"
	coreV1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/portforward"
)

type RemoteDevelopment struct {
	sshPrivateKeyPath string
	sshPublicKeyPath  string

	spinner *spinner.Spinner

	kubernetesClient   *k8s.KubernetesClient
	remoteSSHForwarder *portforward.PortForwarder

	namespace  *coreV1.Namespace
	deployment *appsV1.Deployment
	container  *coreV1.Container

	localSyncPath  string
	remoteSyncPath string

	stopChannel chan bool

	startedAt int64
}

func NewRemoteDevelopment() *RemoteDevelopment {
	return &RemoteDevelopment{
		stopChannel: make(chan bool),
		spinner:     util.MakeSpinner(" Remote Development"),
		startedAt:   time.Now().Unix(),
	}
}

func (r *RemoteDevelopment) WithLocalSyncPath(localSyncPath string) *RemoteDevelopment {
	r.localSyncPath = localSyncPath
	return r
}

func (r *RemoteDevelopment) WithRemoteSyncPath(remoteSyncPath string) *RemoteDevelopment {
	r.remoteSyncPath = remoteSyncPath
	return r
}

func (r *RemoteDevelopment) WithSSH(sshPrivateKeyPath, sshPublicKeyPath string) *RemoteDevelopment {
	r.sshPrivateKeyPath = sshPrivateKeyPath
	r.sshPublicKeyPath = sshPublicKeyPath
	return r
}

func (r *RemoteDevelopment) WithKubernetesClient(kubeConfigPath string) *RemoteDevelopment {
	kubernetesClient, err := k8s.NewKubernetesClient(kubeConfigPath)
	if err != nil {
		panic(err)
	}

	r.kubernetesClient = kubernetesClient

	return r
}

func (r *RemoteDevelopment) WithNamespace(namespace *coreV1.Namespace) *RemoteDevelopment {
	r.namespace = namespace
	return r
}

func (r *RemoteDevelopment) WithNamespaceName(namespaceName string) *RemoteDevelopment {
	namespace, err := r.kubernetesClient.GetNamespace(namespaceName)
	if err != nil {
		panic(err)
	}

	return r.WithNamespace(namespace)
}

func (r *RemoteDevelopment) WithNamespaceFromKubeConfig() *RemoteDevelopment {
	namespace, err := r.kubernetesClient.GetKubeConfigNamespace()
	if err != nil {
		panic(err)
	}

	return r.WithNamespaceName(namespace)
}

func (r *RemoteDevelopment) WithDeployment(deployment *appsV1.Deployment) *RemoteDevelopment {
	if r.namespace == nil {
		panic(fmt.Errorf("you have to select a namespace before selecting a deployment"))
	}

	if r.namespace.GetName() != deployment.GetNamespace() {
		panic(fmt.Errorf(
			"the deployment's namespace(\"%s\") doesn't match the selected namespace \"%s\"",
			deployment.GetNamespace(),
			r.namespace.GetName(),
		))
	}

	r.deployment = deployment
	return r
}

func (r *RemoteDevelopment) WithDeploymentName(deploymentName string) *RemoteDevelopment {
	deployment, err := r.kubernetesClient.GetDeployment(r.namespace.GetName(), deploymentName)
	if err != nil {
		panic(err)
	}

	return r.WithDeployment(deployment)
}

func (r *RemoteDevelopment) WithContainer(container *coreV1.Container) *RemoteDevelopment {
	if r.deployment == nil {
		panic(fmt.Errorf("please select a deployment first"))
	}

	deploymentContainer := k8sTools.GetDeploymentContainerByName(r.deployment, container.Name)
	if deploymentContainer == nil {
		panic(fmt.Errorf(
			"the deployment \"%s\" has no container named \"%s\"",
			r.deployment.GetName(),
			container.Name,
		))
	}

	r.container = container
	return r
}

func (r *RemoteDevelopment) WithContainerName(containerName string) *RemoteDevelopment {
	container := k8sTools.GetDeploymentContainerByName(r.deployment, containerName)
	if container == nil {
		panic(fmt.Errorf("container \"%s\" not found", container.Name))
	}

	return r.WithContainer(container)
}
