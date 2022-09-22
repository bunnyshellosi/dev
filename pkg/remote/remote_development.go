package remote

import (
	"fmt"

	"bunnyshell.com/dev/pkg/k8s"
	k8sTools "bunnyshell.com/dev/pkg/k8s/tools"
	"bunnyshell.com/dev/pkg/util"

	"github.com/briandowns/spinner"
	appsV1 "k8s.io/api/apps/v1"
	coreV1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/portforward"
)

type RemoteDevelopment struct {
	SSHPrivateKeyPath  string
	SSHPublicKeyPath   string
	Spinner            *spinner.Spinner
	KubernetesClient   *k8s.KubernetesClient
	RemoteSSHForwarder *portforward.PortForwarder

	Namespace  *coreV1.Namespace
	Deployment *appsV1.Deployment
	Container  *coreV1.Container

	LocalSyncPath  string
	RemoteSyncPath string

	StopChannel chan bool
}

func NewRemoteDevelopment() *RemoteDevelopment {
	return &RemoteDevelopment{
		StopChannel: make(chan bool),
		Spinner:     util.MakeSpinner(" Remote Development"),
	}
}

func (r *RemoteDevelopment) WithLocalSyncPath(localSyncPath string) *RemoteDevelopment {
	r.LocalSyncPath = localSyncPath
	return r
}

func (r *RemoteDevelopment) WithRemoteSyncPath(remoteSyncPath string) *RemoteDevelopment {
	r.RemoteSyncPath = remoteSyncPath
	return r
}

func (r *RemoteDevelopment) WithSSH(sshPrivateKeyPath, sshPublicKeyPath string) *RemoteDevelopment {
	r.SSHPrivateKeyPath = sshPrivateKeyPath
	r.SSHPublicKeyPath = sshPublicKeyPath
	return r
}

func (r *RemoteDevelopment) WithKubernetesClient(kubeConfigPath string) *RemoteDevelopment {
	kubernetesClient, err := k8s.NewKubernetesClient(kubeConfigPath)
	if err != nil {
		panic(err)
	}

	r.KubernetesClient = kubernetesClient

	return r
}

func (r *RemoteDevelopment) WithNamespace(namespace *coreV1.Namespace) *RemoteDevelopment {
	r.Namespace = namespace
	return r
}

func (r *RemoteDevelopment) WithDeployment(deployment *appsV1.Deployment) *RemoteDevelopment {
	if r.Namespace == nil {
		panic(fmt.Errorf("please select a namespace first"))
	}
	if r.Namespace.GetName() != deployment.GetNamespace() {
		panic(fmt.Errorf(
			"the deployment's namespace(\"%s\") doesn't match the selected namespace \"%s\"",
			deployment.GetNamespace(),
			r.Namespace.GetName(),
		))
	}

	r.Deployment = deployment
	return r
}

func (r *RemoteDevelopment) WithContainer(container *coreV1.Container) *RemoteDevelopment {
	if r.Deployment == nil {
		panic(fmt.Errorf("please select a deployment first"))
	}

	deploymentContainer := k8sTools.GetDeploymentContainerByName(r.Deployment, container.Name)
	if deploymentContainer == nil {
		panic(fmt.Errorf(
			"the deployment \"%s\" has no container named \"%s\"",
			r.Deployment.GetName(),
			container.Name,
		))
	}

	r.Container = container
	return r
}
