package remote

import (
	"fmt"
	"regexp"
	"strconv"
	"time"

	"bunnyshell.com/dev/pkg/k8s"
	"bunnyshell.com/dev/pkg/ssh"
	"bunnyshell.com/dev/pkg/util"

	"github.com/briandowns/spinner"
	appsV1 "k8s.io/api/apps/v1"
	coreV1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/portforward"
)

// +enum
type ResourceType string

const (
	Deployment  ResourceType = "deployment"
	StatefulSet ResourceType = "statefulset"
	DaemonSet   ResourceType = "daemonset"
)

type RemoteDevelopment struct {
	sshPrivateKeyPath string
	sshPublicKeyPath  string

	spinner *spinner.Spinner

	kubernetesClient      *k8s.KubernetesClient
	sshPortForwardOptions *k8s.PortForwardOptions
	sshPortForwarder      *portforward.PortForwarder

	sshTunnels []*ssh.SSHTunnel

	namespace    *coreV1.Namespace
	resourceType ResourceType
	deployment   *appsV1.Deployment
	statefulSet  *appsV1.StatefulSet
	daemonSet    *appsV1.DaemonSet
	container    *coreV1.Container

	localSyncPath  string
	remoteSyncPath string

	stopChannel chan bool

	startedAt   int64
	waitTimeout int64
}

func NewRemoteDevelopment() *RemoteDevelopment {
	return &RemoteDevelopment{
		stopChannel: make(chan bool),
		spinner:     util.MakeSpinner(" Remote Development"),
		startedAt:   time.Now().Unix(),
		waitTimeout: 120,
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

func (r *RemoteDevelopment) WithResourceType(resourceType ResourceType) *RemoteDevelopment {
	r.resourceType = resourceType
	return r
}

func (r *RemoteDevelopment) WithDeployment(deployment *appsV1.Deployment) *RemoteDevelopment {
	if r.namespace == nil {
		panic(ErrNoNamespaceSelected)
	}

	if r.namespace.GetName() != deployment.GetNamespace() {
		panic(fmt.Errorf(
			"the deployment's namespace(\"%s\") doesn't match the selected namespace \"%s\"",
			deployment.GetNamespace(),
			r.namespace.GetName(),
		))
	}

	r.WithResourceType(Deployment)
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

func (r *RemoteDevelopment) WithStatefulSet(statefulSet *appsV1.StatefulSet) *RemoteDevelopment {
	if r.namespace == nil {
		panic(ErrNoNamespaceSelected)
	}

	if r.namespace.GetName() != statefulSet.GetNamespace() {
		panic(fmt.Errorf(
			"the statefulset's namespace(\"%s\") doesn't match the selected namespace \"%s\"",
			statefulSet.GetNamespace(),
			r.namespace.GetName(),
		))
	}

	r.WithResourceType(StatefulSet)
	r.statefulSet = statefulSet
	return r
}

func (r *RemoteDevelopment) WithStatefulSetName(name string) *RemoteDevelopment {
	statefulSet, err := r.kubernetesClient.GetStatefulSet(r.namespace.GetName(), name)
	if err != nil {
		panic(err)
	}

	return r.WithStatefulSet(statefulSet)
}

func (r *RemoteDevelopment) WithDaemonSet(daemonSet *appsV1.DaemonSet) *RemoteDevelopment {
	if r.namespace == nil {
		panic(ErrNoNamespaceSelected)
	}

	if r.namespace.GetName() != daemonSet.GetNamespace() {
		panic(fmt.Errorf(
			"the daemonset's namespace(\"%s\") doesn't match the selected namespace \"%s\"",
			daemonSet.GetNamespace(),
			r.namespace.GetName(),
		))
	}

	r.WithResourceType(DaemonSet)
	r.daemonSet = daemonSet
	return r
}

func (r *RemoteDevelopment) WithDaemonSetName(name string) *RemoteDevelopment {
	daemonSet, err := r.kubernetesClient.GetDaemonSet(r.namespace.GetName(), name)
	if err != nil {
		panic(err)
	}

	return r.WithDaemonSet(daemonSet)
}

func (r *RemoteDevelopment) WithContainer(container *coreV1.Container) *RemoteDevelopment {
	if r.resourceType == "" {
		panic(fmt.Errorf("please select a resource first"))
	}

	r.container = container
	return r
}

func (r *RemoteDevelopment) WithContainerName(containerName string) *RemoteDevelopment {
	container, err := r.getResourceContainer(containerName)
	if err != nil {
		panic(err)
	}

	return r.WithContainer(container)
}

func (r *RemoteDevelopment) getResource() (Resource, error) {
	switch r.resourceType {
	case Deployment:
		return r.deployment, nil
	case StatefulSet:
		return r.statefulSet, nil
	case DaemonSet:
		return r.daemonSet, nil
	default:
		return nil, r.resourceTypeNotSupportedError()
	}
}

func (r *RemoteDevelopment) getResourceType(resource Resource) (ResourceType, error) {
	switch resource.(type) {
	case *appsV1.Deployment:
		return Deployment, nil
	case *appsV1.StatefulSet:
		return StatefulSet, nil
	case *appsV1.DaemonSet:
		return DaemonSet, nil
	default:
		return "", ErrInvalidResourceType
	}
}

func (r *RemoteDevelopment) WithResource(resource Resource) *RemoteDevelopment {
	resourceType, err := r.getResourceType(resource)
	if err != nil {
		panic(err)
	}

	switch resourceType {
	case Deployment:
		r.WithDeployment(resource.(*appsV1.Deployment))
	case StatefulSet:
		r.WithStatefulSet(resource.(*appsV1.StatefulSet))
	case DaemonSet:
		r.WithDaemonSet(resource.(*appsV1.DaemonSet))
	default:
		panic(fmt.Errorf(
			"could not determine the resource Kind for resource type \"%s\"",
			resourceType,
		))
	}

	return r
}

func (r *RemoteDevelopment) WithSSHTunnels(values ...*ssh.SSHTunnel) *RemoteDevelopment {
	for i := range values {
		if values[i] == nil {
			panic("nil value passed to WithSSHTunnels")
		}
		r.sshTunnels = append(r.sshTunnels, values[i])
	}
	return r
}

var portForwardExp = regexp.MustCompile("^(?P<local>[0-9]+)(?P<direction>>|<)(?P<remote>[0-9]+)$")

func (r *RemoteDevelopment) PrepareSSHTunnels(portMappings []string) error {
	for i := range portMappings {
		portMapping := portMappings[i]

		if !portForwardExp.MatchString(portMapping) {
			return fmt.Errorf("invalid port mapping: %s", portMapping)
		}

		match := portForwardExp.FindStringSubmatch(portMapping)
		result := make(map[string]string)
		for i, name := range portForwardExp.SubexpNames() {
			if i != 0 && name != "" {
				result[name] = match[i]
			}
		}

		localPort, err := strconv.Atoi(result["local"])
		if err != nil {
			return err
		}
		remotePort, err := strconv.Atoi(result["remote"])
		if err != nil {
			return err
		}
		localEndpoint := ssh.NewEndpoint("127.0.0.1", localPort)
		remoteEndpoint := ssh.NewEndpoint("0.0.0.0", remotePort)

		tunnel := ssh.NewSSHTunnel().
			WithLocalEndpoint(localEndpoint).
			WithRemoteEndpoint(remoteEndpoint)

		switch result["direction"] {
		case ">":
			tunnel.WithMode(ssh.ForwardModeForward)
		case "<":
			tunnel.WithMode(ssh.ForwardModeReverse)
		}

		r.WithSSHTunnels(tunnel)
	}
	return nil
}

func (r *RemoteDevelopment) WithWaitTimeout(waitTimeout int64) *RemoteDevelopment {
	r.waitTimeout = waitTimeout
	return r
}
