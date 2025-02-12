package debug

import (
	"fmt"
	"time"
	"strings"

	"bunnyshell.com/dev/pkg/k8s"
	"bunnyshell.com/dev/pkg/remote/container"
	"bunnyshell.com/dev/pkg/util"

	"github.com/briandowns/spinner"
	appsV1 "k8s.io/api/apps/v1"
	coreV1 "k8s.io/api/core/v1"
)

// +enum
type ResourceType string

const (
	Deployment  ResourceType = "deployment"
	StatefulSet ResourceType = "statefulset"
	DaemonSet   ResourceType = "daemonset"
)

type DebugComponent struct {
	ContainerName   string
	ContainerConfig container.Config

	AutoSelectSingleResource bool

	spinner *spinner.Spinner

	kubernetesClient      *k8s.KubernetesClient

	namespace    *coreV1.Namespace
	resourceType ResourceType
	deployment   *appsV1.Deployment
	statefulSet  *appsV1.StatefulSet
	daemonSet    *appsV1.DaemonSet
	container    *coreV1.Container

	isInitContainer bool

	shouldPrepareResource bool

	stopChannel chan bool

	startedAt   int64
	waitTimeout int64
}

func NewDebugComponent() *DebugComponent {
	return &DebugComponent{
		ContainerConfig: *container.NewConfig(),

		AutoSelectSingleResource: true,

		shouldPrepareResource: true,

		stopChannel: make(chan bool),
		spinner:     util.MakeSpinner(" Debug"),
		startedAt:   time.Now().Unix(),
		waitTimeout: 120,
	}
}

func (d *DebugComponent) WithKubernetesClient(kubeConfigPath string) *DebugComponent {
	kubernetesClient, err := k8s.NewKubernetesClient(kubeConfigPath)
	if err != nil {
		panic(err)
	}

	d.kubernetesClient = kubernetesClient

	return d
}

func (d *DebugComponent) WithNamespace(namespace *coreV1.Namespace) *DebugComponent {
	d.namespace = namespace
	return d
}

func (d *DebugComponent) WithNamespaceName(namespaceName string) *DebugComponent {
	namespace, err := d.kubernetesClient.GetNamespace(namespaceName)
	if err != nil {
		panic(err)
	}

	return d.WithNamespace(namespace)
}

func (d *DebugComponent) WithNamespaceFromKubeConfig() *DebugComponent {
	namespace, err := d.kubernetesClient.GetKubeConfigNamespace()
	if err != nil {
		panic(err)
	}

	return d.WithNamespaceName(namespace)
}

func (d *DebugComponent) WithResourceType(resourceType ResourceType) *DebugComponent {
	d.resourceType = resourceType
	return d
}

func (d *DebugComponent) WithDeployment(deployment *appsV1.Deployment) *DebugComponent {
	if d.namespace == nil {
		panic(ErrNoNamespaceSelected)
	}

	if d.namespace.GetName() != deployment.GetNamespace() {
		panic(fmt.Errorf(
			"the deployment's namespace(\"%s\") doesn't match the selected namespace \"%s\"",
			deployment.GetNamespace(),
			d.namespace.GetName(),
		))
	}

	d.WithResourceType(Deployment)
	d.deployment = deployment
	return d
}

func (d *DebugComponent) WithDeploymentName(deploymentName string) *DebugComponent {
	deployment, err := d.kubernetesClient.GetDeployment(d.namespace.GetName(), deploymentName)
	if err != nil {
		panic(err)
	}

	return d.WithDeployment(deployment)
}

func (d *DebugComponent) WithStatefulSet(statefulSet *appsV1.StatefulSet) *DebugComponent {
	if d.namespace == nil {
		panic(ErrNoNamespaceSelected)
	}

	if d.namespace.GetName() != statefulSet.GetNamespace() {
		panic(fmt.Errorf(
			"the statefulset's namespace(\"%s\") doesn't match the selected namespace \"%s\"",
			statefulSet.GetNamespace(),
			d.namespace.GetName(),
		))
	}

	d.WithResourceType(StatefulSet)
	d.statefulSet = statefulSet
	return d
}

func (d *DebugComponent) WithStatefulSetName(name string) *DebugComponent {
	statefulSet, err := d.kubernetesClient.GetStatefulSet(d.namespace.GetName(), name)
	if err != nil {
		panic(err)
	}

	return d.WithStatefulSet(statefulSet)
}

func (d *DebugComponent) WithDaemonSet(daemonSet *appsV1.DaemonSet) *DebugComponent {
	if d.namespace == nil {
		panic(ErrNoNamespaceSelected)
	}

	if d.namespace.GetName() != daemonSet.GetNamespace() {
		panic(fmt.Errorf(
			"the daemonset's namespace(\"%s\") doesn't match the selected namespace \"%s\"",
			daemonSet.GetNamespace(),
			d.namespace.GetName(),
		))
	}

	d.WithResourceType(DaemonSet)
	d.daemonSet = daemonSet
	return d
}

func (d *DebugComponent) WithDaemonSetName(name string) *DebugComponent {
	daemonSet, err := d.kubernetesClient.GetDaemonSet(d.namespace.GetName(), name)
	if err != nil {
		panic(err)
	}

	return d.WithDaemonSet(daemonSet)
}

func (d *DebugComponent) WithContainer(container *coreV1.Container) *DebugComponent {
	if d.resourceType == "" {
		panic(fmt.Errorf("please select a resource first"))
	}

	d.container = container
	d.isInitContainer = false
	return d
}

func (d *DebugComponent) WithInitContainer(container *coreV1.Container) *DebugComponent {
	if d.resourceType == "" {
		panic(fmt.Errorf("please select a resource first"))
	}

	d.container = container
	d.isInitContainer = true
	return d
}

func (d *DebugComponent) WithContainerName(containerName string) *DebugComponent {
	container, err := d.getResourceContainer(containerName)
	if err != nil {
	    if !strings.HasSuffix(err.Error(), " not found") {
	        panic(err)
	    }

		initContainer, err := d.getResourceInitContainer(containerName)
		if err != nil {
		    panic(err)
		}

        return d.WithInitContainer(initContainer)
	}

	return d.WithContainer(container)
}

func (d *DebugComponent) getResource() (Resource, error) {
	switch d.resourceType {
	case Deployment:
		return d.deployment, nil
	case StatefulSet:
		return d.statefulSet, nil
	case DaemonSet:
		return d.daemonSet, nil
	default:
		return nil, d.resourceTypeNotSupportedError()
	}
}

func (d *DebugComponent) getResourceType(resource Resource) (ResourceType, error) {
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

func (d *DebugComponent) WithResource(resource Resource) *DebugComponent {
	resourceType, err := d.getResourceType(resource)
	if err != nil {
		panic(err)
	}

	switch resourceType {
	case Deployment:
		d.WithDeployment(resource.(*appsV1.Deployment))
	case StatefulSet:
		d.WithStatefulSet(resource.(*appsV1.StatefulSet))
	case DaemonSet:
		d.WithDaemonSet(resource.(*appsV1.DaemonSet))
	default:
		panic(fmt.Errorf(
			"could not determine the resource Kind for resource type \"%s\"",
			resourceType,
		))
	}

	return d
}

func (d *DebugComponent) WithWaitTimeout(waitTimeout int64) *DebugComponent {
	d.waitTimeout = waitTimeout
	return d
}

func (d *DebugComponent) GetSelectedContainerName() (string, error) {
	if d.container == nil {
		panic(fmt.Errorf("please select a container first"))
	}

	return d.container.Name, nil
}
