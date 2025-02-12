package debug

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"bunnyshell.com/dev/pkg/k8s/patch"

	k8sTools "bunnyshell.com/dev/pkg/k8s/tools"
	appsV1 "k8s.io/api/apps/v1"
	coreV1 "k8s.io/api/core/v1"
	apiMetaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	appsCoreV1 "k8s.io/client-go/applyconfigurations/apps/v1"
	applyCoreV1 "k8s.io/client-go/applyconfigurations/core/v1"
	applyMetaV1 "k8s.io/client-go/applyconfigurations/meta/v1"
)

const (
	MetadataPrefix    = "debug.bunnyshell.com/"
	MetadataActive    = MetadataPrefix + "active"
	MetadataStartedAt = MetadataPrefix + "started-at"
	MetadataService   = MetadataPrefix + "service"
	MetadataContainer = MetadataPrefix + "container"
	MetadataRollback  = MetadataPrefix + "rollback-manifest"

	MetadataKubeCTLLastAppliedConf = "kubectl.kubernetes.io/last-applied-configuration"
	MetadataK8SRevision            = "deployment.kubernetes.io/revision"

	RemoteDevMetadataActive = "remote-dev.bunnyshell.com/active"
)

var (
	ErrInvalidResourceType = fmt.Errorf("invalid resource type")
)

type Resource interface {
	GetName() string
	GetNamespace() string
	GetAnnotations() map[string]string
	GetLabels() map[string]string
}

func (d *DebugComponent) resourceTypeNotSupportedError() error {
	return fmt.Errorf("resource type \"%s\" not supported", d.resourceType)
}

func (d *DebugComponent) getResourcePatch() (patch.Resource, error) {
	switch d.resourceType {
	case Deployment:
		var replicas int32 = 1
		strategy := appsV1.RecreateDeploymentStrategyType
		return &patch.DeploymentPatchConfiguration{
			ObjectMetaApplyConfiguration: &applyMetaV1.ObjectMetaApplyConfiguration{},
			Spec: &patch.DeploymentSpecPatchConfiguration{
				Strategy: &patch.DeploymentStrategyPatchConfiguration{
					Type:          &strategy,
					RollingUpdate: nil,
				},
				Replicas: &replicas,
			},
		}, nil
	case StatefulSet:
		var replicas int32 = 1
		strategy := appsV1.OnDeleteStatefulSetStrategyType
		return &patch.StatefulSetPatchConfiguration{
			ObjectMetaApplyConfiguration: &applyMetaV1.ObjectMetaApplyConfiguration{},
			Spec: &patch.StatefulSetSpecPatchConfiguration{
				UpdateStrategy: &patch.StatefulSetStrategyPatchConfiguration{
					Type:          &strategy,
					RollingUpdate: nil,
				},
				Replicas: &replicas,
			},
		}, nil
	case DaemonSet:
		strategy := appsV1.OnDeleteDaemonSetStrategyType
		return &patch.DaemonSetPatchConfiguration{
			ObjectMetaApplyConfiguration: &applyMetaV1.ObjectMetaApplyConfiguration{},
			Spec: &patch.DaemonSetSpecPatchConfiguration{
				UpdateStrategy: &patch.DaemonSetStrategyPatchConfiguration{
					Type:          &strategy,
					RollingUpdate: nil,
				},
			},
		}, nil
	default:
		return nil, d.resourceTypeNotSupportedError()
	}
}

func (d *DebugComponent) prepareResource() error {
	d.StartSpinner(" Setup k8s pod for debugging")
	defer d.StopSpinner()

	currentManifestSnapshot, err := d.getCurrentManifestSnapshot()
	if err != nil {
		return err
	}

	resource, err := d.getResource()
	if err != nil {
		return err
	}

	resourcePatch, err := d.getResourcePatch()
	if err != nil {
		return err
	}

	annotations := make(map[string]string)
	annotations[MetadataStartedAt] = strconv.FormatInt(d.startedAt, 10)
	annotations[MetadataContainer] = d.container.Name
	_, ok := resource.GetAnnotations()[MetadataRollback]
	if !ok {
		annotations[MetadataRollback] = string(currentManifestSnapshot)
	}
	labels := make(map[string]string)
	labels[MetadataActive] = "true"

	resourcePatch.WithAnnotations(annotations).WithLabels(labels)

	podTemplateSpec := applyCoreV1.PodTemplateSpec()
	if err := d.preparePodTemplateSpec(podTemplateSpec); err != nil {
		return err
	}
	resourcePatch.WithSpecTemplate(podTemplateSpec)

	data, err := json.Marshal(resourcePatch)
	if err != nil {
		return err
	}

	if err := d.resetResourceContainer(resource); err != nil {
		return fmt.Errorf("cannot reset container: %w", err)
	}

	switch d.resourceType {
	case Deployment:
		return d.kubernetesClient.PatchDeployment(resource.GetNamespace(), resource.GetName(), data)
	case StatefulSet:
		return d.kubernetesClient.PatchStatefulSet(resource.GetNamespace(), resource.GetName(), data)
	case DaemonSet:
		return d.kubernetesClient.PatchDaemonSet(resource.GetNamespace(), resource.GetName(), data)
	default:
		return d.resourceTypeNotSupportedError()
	}
}

func (d *DebugComponent) resetResourceContainer(resource Resource) error {
	containerIndex, isInit, err := d.getContainerIndex()
	if err != nil {
		return err
	}

    specPath := "containers"
    if isInit {
        specPath = "initContainers"
    }

	// we need to use replace because remove fails if the path is missing
	resetJSON, err := json.Marshal([]map[string]any{
		{
			"op":    "replace",
			"path":  fmt.Sprintf("/spec/template/spec/%s/%d/args", specPath, containerIndex),
			"value": []string{},
		},
		{
			"op":    "replace",
			"path":  fmt.Sprintf("/spec/template/spec/%s/%d/readinessProbe", specPath, containerIndex),
			"value": nil,
		},
		{
			"op":    "replace",
			"path":  fmt.Sprintf("/spec/template/spec/%s/%d/livenessProbe", specPath, containerIndex),
			"value": nil,
		},
		{
			"op":    "replace",
			"path":  fmt.Sprintf("/spec/template/spec/%s/%d/startupProbe", specPath, containerIndex),
			"value": nil,
		},
	})

	if err != nil {
		return err
	}

	switch d.resourceType {
	case Deployment:
		return d.kubernetesClient.BatchPatchDeployment(resource.GetNamespace(), resource.GetName(), resetJSON)
	case StatefulSet:
		return d.kubernetesClient.BatchPatchStatefulSet(resource.GetNamespace(), resource.GetName(), resetJSON)
	case DaemonSet:
		return d.kubernetesClient.BatchPatchDaemonSet(resource.GetNamespace(), resource.GetName(), resetJSON)
	default:
		return d.resourceTypeNotSupportedError()
	}
}

func (d *DebugComponent) restoreDeployment() error {
	resource, err := d.getResource()
	if err != nil {
		return err
	}

	snapshot, ok := resource.GetAnnotations()[MetadataRollback]
	if !ok {
		return fmt.Errorf("no rollback manifest available")
	}

	switch d.resourceType {
	case Deployment:
		deployment := &appsV1.Deployment{}
		if err := json.Unmarshal([]byte(snapshot), deployment); err != nil {
			return err
		}

		_, err = d.kubernetesClient.UpdateDeployment(deployment.GetNamespace(), deployment)
		return err
	case StatefulSet:
		statefulSet := &appsV1.StatefulSet{}
		if err := json.Unmarshal([]byte(snapshot), statefulSet); err != nil {
			return err
		}

		_, err = d.kubernetesClient.UpdateStatefulSet(statefulSet.GetNamespace(), statefulSet)
		return err
	case DaemonSet:
		daemonSet := &appsV1.DaemonSet{}
		if err := json.Unmarshal([]byte(snapshot), daemonSet); err != nil {
			return err
		}

		_, err = d.kubernetesClient.UpdateDaemonSet(daemonSet.GetNamespace(), daemonSet)
		return err
	default:
		return d.resourceTypeNotSupportedError()
	}
}

func (d *DebugComponent) getResourceManifest() ([]byte, error) {
	resource, err := d.getResource()
	if err != nil {
		return nil, err
	}

	return json.Marshal(resource)
}

func (d *DebugComponent) getCurrentManifestSnapshot() (string, error) {
	fullSnapshot, err := d.getResourceManifest()
	if err != nil {
		return "", err
	}

	var snapshot []byte
	switch d.resourceType {
	case Deployment:
		applyResource := &appsCoreV1.DeploymentApplyConfiguration{}
		if err := json.Unmarshal(fullSnapshot, applyResource); err != nil {
			return "", err
		}

		// strip unnecessary data
		applyResource.WithStatus(nil)
		applyResource.Generation = nil
		applyResource.UID = nil
		applyResource.ResourceVersion = nil
		annotations := make(map[string]string)
		for key, value := range applyResource.Annotations {
			if key == MetadataK8SRevision || key == MetadataKubeCTLLastAppliedConf {
				continue
			}

			annotations[key] = value
		}
		applyResource.Annotations = annotations

		snapshot, err = json.Marshal(applyResource)
		if err != nil {
			return "", err
		}
	case StatefulSet:
		applyResource := &appsCoreV1.StatefulSetApplyConfiguration{}
		if err := json.Unmarshal(fullSnapshot, applyResource); err != nil {
			return "", err
		}

		// strip unnecessary data
		applyResource.WithStatus(nil)
		applyResource.Generation = nil
		applyResource.UID = nil
		applyResource.ResourceVersion = nil
		annotations := make(map[string]string)
		for key, value := range applyResource.Annotations {
			if key == MetadataK8SRevision || key == MetadataKubeCTLLastAppliedConf {
				continue
			}

			annotations[key] = value
		}
		applyResource.Annotations = annotations

		snapshot, err = json.Marshal(applyResource)
		if err != nil {
			return "", err
		}
	case DaemonSet:
		applyResource := &appsCoreV1.DaemonSetApplyConfiguration{}
		if err := json.Unmarshal(fullSnapshot, applyResource); err != nil {
			return "", err
		}

		// strip unnecessary data
		applyResource.WithStatus(nil)
		applyResource.Generation = nil
		applyResource.UID = nil
		applyResource.ResourceVersion = nil
		annotations := make(map[string]string)
		for key, value := range applyResource.Annotations {
			if key == MetadataK8SRevision || key == MetadataKubeCTLLastAppliedConf {
				continue
			}

			annotations[key] = value
		}
		applyResource.Annotations = annotations

		snapshot, err = json.Marshal(applyResource)
		if err != nil {
			return "", err
		}
	default:
		return "", d.resourceTypeNotSupportedError()
	}

	return string(snapshot), nil
}

func (d *DebugComponent) preparePodTemplateSpec(podTemplateSpec *applyCoreV1.PodTemplateSpecApplyConfiguration) error {
	resource, err := d.getResource()
	if err != nil {
		return err
	}

	podAnnotations := make(map[string]string)
	podAnnotations[MetadataStartedAt] = strconv.FormatInt(d.startedAt, 10)
	podAnnotations[MetadataContainer] = d.container.Name
	podLabels := make(map[string]string)
	podLabels[MetadataActive] = "true"
	podLabels[MetadataService] = resource.GetName()

	podTemplateSpec.
		WithAnnotations(podAnnotations).
		WithLabels(podLabels)

	return d.preparePodSpec(podTemplateSpec)
}

func (d *DebugComponent) preparePodSpec(podTemplateSpec *applyCoreV1.PodTemplateSpecApplyConfiguration) error {
	podSpec := applyCoreV1.PodSpec()


	if err := d.prepareContainer(podSpec); err != nil {
		return err
	}

	podTemplateSpec.WithSpec(podSpec)

	return nil
}

func (d *DebugComponent) prepareContainer(podSpec *applyCoreV1.PodSpecApplyConfiguration) error {
	container := applyCoreV1.Container().
		WithName(d.container.Name).
		WithCommand("sh", "-c", "tail -f /dev/null")

	if !d.isInitContainer {
	    nullProbe := d.getNullProbeApplyConfiguration()

	    container.
	        WithLivenessProbe(nullProbe).
            WithReadinessProbe(nullProbe).
            WithStartupProbe(nullProbe)
	}

	d.ContainerConfig.ApplyTo(container)

    if d.isInitContainer {
        podSpec.WithInitContainers(container)
    } else {
        podSpec.WithContainers(container)
    }

	return nil
}

func (d *DebugComponent) getNullProbeApplyConfiguration() *applyCoreV1.ProbeApplyConfiguration {
	return applyCoreV1.Probe().
		WithExec(applyCoreV1.ExecAction().WithCommand("true")).
		WithPeriodSeconds(5)
}

func (d *DebugComponent) getResourceSelector() (*apiMetaV1.LabelSelector, error) {
	switch d.resourceType {
	case Deployment:
		return d.deployment.Spec.Selector, nil
	case StatefulSet:
		return d.statefulSet.Spec.Selector, nil
	case DaemonSet:
		return d.daemonSet.Spec.Selector, nil
	default:
		return nil, d.resourceTypeNotSupportedError()
	}
}

func (d *DebugComponent) waitPodReady() error {
	d.StartSpinner(" Waiting for pod to be ready")
	defer d.StopSpinner()

	resource, err := d.getResource()
	if err != nil {
		return err
	}

	resourceSelector, err := d.getResourceSelector()
	if err != nil {
		return err
	}

	namespace := resource.GetNamespace()
	labelSelector := apiMetaV1.LabelSelector{MatchLabels: resourceSelector.MatchLabels}
	listOptions := apiMetaV1.ListOptions{
		LabelSelector: labels.Set(labelSelector.MatchLabels).String(),
	}

	startTimestamp := time.Now().Unix()
	for {
		time.Sleep(1 * time.Second)
		podList, err := d.kubernetesClient.ListPods(namespace, listOptions)
		if err != nil {
			return err
		}

		for _, pod := range podList.Items {
		    if d.isInitContainer {
		        if pod.DeletionTimestamp != nil || pod.Status.Phase != coreV1.PodPending {
                    continue
                }

                for _, containerStatus := range pod.Status.InitContainerStatuses {
                    if containerStatus.Name == d.container.Name && containerStatus.Started != nil && *containerStatus.Started {
                        return nil
                    }
                }
		    } else {
                if pod.DeletionTimestamp != nil || pod.Status.Phase != coreV1.PodRunning {
                    continue
                }

                for _, containerStatus := range pod.Status.ContainerStatuses {
                    if containerStatus.Name == d.container.Name && containerStatus.Ready {
                        return nil
                    }
                }
            }
		}

		nowTimestamp := time.Now().Unix()
		if nowTimestamp-startTimestamp >= d.waitTimeout {
			break
		}
	}

	// timeout reached
	return fmt.Errorf("pod not ready for debugging")
}

func (d *DebugComponent) getResourceContainers() ([]coreV1.Container, error) {
	switch d.resourceType {
	case Deployment:
		return k8sTools.GetDeploymentContainers(d.deployment), nil
	case StatefulSet:
		return k8sTools.GetStatefulSetContainers(d.statefulSet), nil
	case DaemonSet:
		return k8sTools.GetDaemonSetContainers(d.daemonSet), nil
	default:
		return []coreV1.Container{}, d.resourceTypeNotSupportedError()
	}
}

func (d *DebugComponent) getResourceContainer(containerName string) (*coreV1.Container, error) {
	switch d.resourceType {
	case Deployment:
		return k8sTools.GetDeploymentContainerByName(d.deployment, containerName)
	case StatefulSet:
		return k8sTools.GetStatefulSetContainerByName(d.statefulSet, containerName)
	case DaemonSet:
		return k8sTools.GetDaemonSetContainerByName(d.daemonSet, containerName)
	default:
		return nil, ErrNoResourceSelected
	}
}

func (d *DebugComponent) getResourceInitContainers() ([]coreV1.Container, error) {
	switch d.resourceType {
	case Deployment:
		return k8sTools.GetDeploymentInitContainers(d.deployment), nil
	case StatefulSet:
		return k8sTools.GetStatefulSetInitContainers(d.statefulSet), nil
	case DaemonSet:
		return k8sTools.GetDaemonSetInitContainers(d.daemonSet), nil
	default:
		return []coreV1.Container{}, d.resourceTypeNotSupportedError()
	}
}

func (d *DebugComponent) getResourceInitContainer(containerName string) (*coreV1.Container, error) {
	switch d.resourceType {
	case Deployment:
		return k8sTools.GetDeploymentInitContainerByName(d.deployment, containerName)
	case StatefulSet:
		return k8sTools.GetStatefulSetInitContainerByName(d.statefulSet, containerName)
	case DaemonSet:
		return k8sTools.GetDaemonSetInitContainerByName(d.daemonSet, containerName)
	default:
		return nil, ErrNoResourceSelected
	}
}

func (d *DebugComponent) getContainerIndex() (int, bool, error) {
	if d.container == nil {
		return -1, false, fmt.Errorf("%w: %s", ErrContainerNotFound, "<nil>")
	}

	containers, err := d.getResourceContainers()
	if err != nil {
		return -1, false, err
	}

	for i, container := range containers {
		if container.Name == d.container.Name {
			return i, false, nil
		}
	}

    initContainers, err := d.getResourceInitContainers()
    if err != nil {
        return -1, true, err
    }

    for i, initContainer := range initContainers {
        if initContainer.Name == d.container.Name {
            return i, true, nil
        }
    }

	return -1, false, fmt.Errorf("%w: %s", ErrContainerNotFound, d.container.Name)
}
