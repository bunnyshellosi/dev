package remote

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"bunnyshell.com/dev/pkg/build"
	"bunnyshell.com/dev/pkg/k8s/patch"

	k8sTools "bunnyshell.com/dev/pkg/k8s/tools"
	appsV1 "k8s.io/api/apps/v1"
	coreV1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	apiMetaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	appsCoreV1 "k8s.io/client-go/applyconfigurations/apps/v1"
	applyCoreV1 "k8s.io/client-go/applyconfigurations/core/v1"
	applyMetaV1 "k8s.io/client-go/applyconfigurations/meta/v1"
)

const (
	MetadataPrefix    = "remote-dev.bunnyshell.com/"
	MetadataActive    = MetadataPrefix + "active"
	MetadataStartedAt = MetadataPrefix + "started-at"
	MetadataService   = MetadataPrefix + "service"
	MetadataContainer = MetadataPrefix + "container"
	MetadataRollback  = MetadataPrefix + "rollback-manifest"

	MetadataKubeCTLLastAppliedConf = "kubectl.kubernetes.io/last-applied-configuration"
	MetadataK8SRevision            = "deployment.kubernetes.io/revision"

	VolumeNameBinaries = "remote-dev-bin"
	VolumeNameConfig   = "remote-dev-config"
	VolumeNameWork     = "remote-dev-work"

	SecretName    = "remote-development"
	PVCNameFormat = "%s-%s-remote-dev"

	SecretAuthorizedKeysKeyName = "authorized_keys"
	SecretAuthorizedKeysPath    = "ssh/authorized_keys"

	ContainerNameBinaries         = "remote-dev-bin"
	ContainerNameWorkPermissions  = "remote-dev-work-permissions"
	ContainerNameWork             = "remote-dev-work"
	ContainerImageWorkPermissions = "alpine:3.17"

	// ConfigSourceDir = "config"
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

func (r *RemoteDevelopment) resourceTypeNotSupportedError() error {
	return fmt.Errorf("resource type \"%s\" not supported", r.resourceType)
}

func (r *RemoteDevelopment) getResourcePatch() (patch.Resource, error) {
	switch r.resourceType {
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
		return nil, r.resourceTypeNotSupportedError()
	}
}

func (r *RemoteDevelopment) prepareResource() error {
	r.StartSpinner(" Setup k8s pod for remote development")
	defer r.StopSpinner()

	currentManifestSnapshot, err := r.getCurrentManifestSnapshot()
	if err != nil {
		return err
	}

	resource, err := r.getResource()
	if err != nil {
		return err
	}

	resourcePatch, err := r.getResourcePatch()
	if err != nil {
		return err
	}

	annotations := make(map[string]string)
	annotations[MetadataStartedAt] = strconv.FormatInt(r.startedAt, 10)
	annotations[MetadataContainer] = r.container.Name
	_, ok := resource.GetAnnotations()[MetadataRollback]
	if !ok {
		annotations[MetadataRollback] = string(currentManifestSnapshot)
	}
	labels := make(map[string]string)
	labels[MetadataActive] = "true"

	resourcePatch.WithAnnotations(annotations).WithLabels(labels)

	podTemplateSpec := applyCoreV1.PodTemplateSpec()
	if err := r.preparePodTemplateSpec(podTemplateSpec); err != nil {
		return err
	}
	resourcePatch.WithSpecTemplate(podTemplateSpec)

	data, err := json.Marshal(resourcePatch)
	if err != nil {
		return err
	}

	switch r.resourceType {
	case Deployment:
		return r.kubernetesClient.PatchDeployment(resource.GetNamespace(), resource.GetName(), data)
	case StatefulSet:
		return r.kubernetesClient.PatchStatefulSet(resource.GetNamespace(), resource.GetName(), data)
	case DaemonSet:
		return r.kubernetesClient.PatchDaemonSet(resource.GetNamespace(), resource.GetName(), data)
	default:
		return r.resourceTypeNotSupportedError()
	}
}

func (r *RemoteDevelopment) restoreDeployment() error {
	resource, err := r.getResource()
	if err != nil {
		return err
	}

	snapshot, ok := resource.GetAnnotations()[MetadataRollback]
	if !ok {
		return fmt.Errorf("no rollback manifest available")
	}

	switch r.resourceType {
	case Deployment:
		deployment := &appsV1.Deployment{}
		if err := json.Unmarshal([]byte(snapshot), deployment); err != nil {
			return err
		}

		_, err = r.kubernetesClient.UpdateDeployment(deployment.GetNamespace(), deployment)
		return err
	case StatefulSet:
		statefulSet := &appsV1.StatefulSet{}
		if err := json.Unmarshal([]byte(snapshot), statefulSet); err != nil {
			return err
		}

		_, err = r.kubernetesClient.UpdateStatefulSet(statefulSet.GetNamespace(), statefulSet)
		return err
	case DaemonSet:
		daemonSet := &appsV1.DaemonSet{}
		if err := json.Unmarshal([]byte(snapshot), daemonSet); err != nil {
			return err
		}

		_, err = r.kubernetesClient.UpdateDaemonSet(daemonSet.GetNamespace(), daemonSet)
		return err
	default:
		return r.resourceTypeNotSupportedError()
	}
}

func (r *RemoteDevelopment) getResourceManifest() ([]byte, error) {
	resource, err := r.getResource()
	if err != nil {
		return nil, err
	}

	return json.Marshal(resource)
}

func (r *RemoteDevelopment) getCurrentManifestSnapshot() (string, error) {
	fullSnapshot, err := r.getResourceManifest()
	if err != nil {
		return "", err
	}

	var snapshot []byte
	switch r.resourceType {
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
		return "", r.resourceTypeNotSupportedError()
	}

	return string(snapshot), nil
}

func (r *RemoteDevelopment) ensurePVC() error {
	labels := make(map[string]string)
	labels[MetadataActive] = "true"

	resourceLimits := coreV1.ResourceList{
		coreV1.ResourceStorage: resource.MustParse("5Gi"),
	}

	resource, err := r.getResource()
	if err != nil {
		return err
	}

	pvcName, err := r.getPVCName()
	if err != nil {
		return err
	}

	remoteDevPVC := applyCoreV1.PersistentVolumeClaim(pvcName, resource.GetNamespace()).
		WithLabels(labels).
		WithSpec(applyCoreV1.PersistentVolumeClaimSpec().
			WithAccessModes(coreV1.ReadWriteOnce).
			WithResources(applyCoreV1.ResourceRequirements().
				WithRequests(resourceLimits)))

	return r.kubernetesClient.ApplyPVC(remoteDevPVC)
}

func (r *RemoteDevelopment) preparePodTemplateSpec(podTemplateSpec *applyCoreV1.PodTemplateSpecApplyConfiguration) error {
	resource, err := r.getResource()
	if err != nil {
		return err
	}

	podAnnotations := make(map[string]string)
	podAnnotations[MetadataStartedAt] = strconv.FormatInt(r.startedAt, 10)
	podAnnotations[MetadataContainer] = r.container.Name
	podLabels := make(map[string]string)
	podLabels[MetadataActive] = "true"
	podLabels[MetadataService] = resource.GetName()

	podTemplateSpec.
		WithAnnotations(podAnnotations).
		WithLabels(podLabels)

	return r.preparePodSpec(podTemplateSpec)
}

func (r *RemoteDevelopment) preparePodSpec(podTemplateSpec *applyCoreV1.PodTemplateSpecApplyConfiguration) error {
	podSpec := applyCoreV1.PodSpec()
	if err := r.prepareVolumes(podSpec); err != nil {
		return err
	}

	if err := r.prepareInitContainers(podSpec); err != nil {
		return err
	}

	if err := r.prepareContainer(podSpec); err != nil {
		return err
	}

	podTemplateSpec.WithSpec(podSpec)

	return nil
}

func (r *RemoteDevelopment) prepareVolumes(podSpec *applyCoreV1.PodSpecApplyConfiguration) error {
	volumes := []*applyCoreV1.VolumeApplyConfiguration{}

	binVolume := applyCoreV1.Volume().WithName(VolumeNameBinaries).WithEmptyDir(applyCoreV1.EmptyDirVolumeSource())
	volumes = append(volumes, binVolume)

	configVolume := applyCoreV1.Volume().
		WithName(VolumeNameConfig).
		WithSecret(applyCoreV1.SecretVolumeSource().
			WithSecretName(r.getSecretName()).
			WithItems(applyCoreV1.KeyToPath().
				WithKey(SecretAuthorizedKeysKeyName).
				WithPath(SecretAuthorizedKeysPath)))
	volumes = append(volumes, configVolume)

	pvcName, err := r.getPVCName()
	if err != nil {
		return err
	}
	workVolume := applyCoreV1.Volume().
		WithName(VolumeNameWork).
		WithPersistentVolumeClaim(applyCoreV1.PersistentVolumeClaimVolumeSource().
			WithClaimName(pvcName))
	volumes = append(volumes, workVolume)

	podSpec.WithVolumes(volumes...)

	return nil
}

func (r *RemoteDevelopment) prepareInitContainers(podSpec *applyCoreV1.PodSpecApplyConfiguration) error {
	pullPolicy := coreV1.PullIfNotPresent
	image := r.getSSHServerImage()
	if strings.Contains(image, ":latest") {
		pullPolicy = coreV1.PullAlways
	}

	binariesVolumeMountPath := "/remote-dev-bin"
	binariesInitContainer := applyCoreV1.Container().
		WithName(ContainerNameBinaries).
		WithCommand("sh", "-c", fmt.Sprintf("cp -p /usr/local/bin/* %s", binariesVolumeMountPath)).
		WithImage(image).
		WithImagePullPolicy(pullPolicy).
		WithVolumeMounts(applyCoreV1.VolumeMount().
			WithName(VolumeNameBinaries).
			WithMountPath(binariesVolumeMountPath))

	workVolumesMountPath := "/volumes"
	appSourceDir := r.getRemoteSyncPathHash()
	workVolumeAppSourceDir := fmt.Sprintf("%s/%s", workVolumesMountPath, appSourceDir)

	workPermissionsInitContainer := applyCoreV1.Container().
		WithName(ContainerNameWorkPermissions).
		WithCommand("sh", "-c", fmt.Sprintf(
			"mkdir -p -m 777 %s",
			workVolumeAppSourceDir,
		)).
		WithImage(ContainerImageWorkPermissions).
		WithImagePullPolicy(coreV1.PullIfNotPresent).
		WithVolumeMounts(applyCoreV1.VolumeMount().
			WithName(VolumeNameWork).
			WithMountPath(workVolumesMountPath))

	workInitContainer := applyCoreV1.Container().
		WithName(ContainerNameWork).
		WithCommand("sh", "-c", fmt.Sprintf(
			"[ \"$(ls -A %s)\" ] || (cp -Rp %s/. %s; exit 0)",
			workVolumeAppSourceDir,
			r.remoteSyncPath,
			workVolumeAppSourceDir,
		)).
		WithImage(r.container.Image).
		WithImagePullPolicy(coreV1.PullIfNotPresent).
		WithVolumeMounts(applyCoreV1.VolumeMount().
			WithName(VolumeNameWork).
			WithMountPath(workVolumesMountPath))

	podSpec.WithInitContainers(binariesInitContainer, workPermissionsInitContainer, workInitContainer)

	return nil
}

func (r *RemoteDevelopment) getSSHServerImage() string {
	return fmt.Sprintf("%s:%s", build.SSHServerImage, build.SSHServerVersion)
}

func (r *RemoteDevelopment) getRemoteSyncPathHash() string {
	hash := md5.Sum([]byte(r.remoteSyncPath))
	return hex.EncodeToString(hash[:])
}

func (r *RemoteDevelopment) prepareContainer(podSpec *applyCoreV1.PodSpecApplyConfiguration) error {
	basePath := "/opt/bunnyshell"
	binariesVolumeMountPath := basePath + "/bin"
	secretsVolumeMountPath := basePath + "/secret"
	appSourceDir := r.getRemoteSyncPathHash()
	// configVolumeMountPath := basePath + "/.config"

	volumeMounts := []*applyCoreV1.VolumeMountApplyConfiguration{
		applyCoreV1.VolumeMount().
			WithName(VolumeNameBinaries).
			WithMountPath(binariesVolumeMountPath),
		applyCoreV1.VolumeMount().
			WithName(VolumeNameConfig).
			WithMountPath(secretsVolumeMountPath),
		applyCoreV1.VolumeMount().
			WithName(VolumeNameWork).
			WithMountPath(r.remoteSyncPath).
			WithSubPath(appSourceDir),
		// applyCoreV1.VolumeMount().
		// 	WithName(VolumeNameWork).
		// 	WithMountPath(configVolumeMountPath).
		// 	WithSubPath(ConfigSourceDir),
	}

	nullProbe := r.getNullProbeApplyConfiguration()

	startCommand := binariesVolumeMountPath + "/start.sh"
	container := applyCoreV1.Container().
		WithName(r.container.Name).
		WithCommand(startCommand).
		WithLivenessProbe(nullProbe).
		WithReadinessProbe(nullProbe).
		WithStartupProbe(nullProbe).
		WithVolumeMounts(volumeMounts...)

	podSpec.WithContainers(container)

	return nil
}

func (r *RemoteDevelopment) getNullProbeApplyConfiguration() *applyCoreV1.ProbeApplyConfiguration {
	return applyCoreV1.Probe().
		WithExec(applyCoreV1.ExecAction().WithCommand("true")).
		WithPeriodSeconds(5)
}

func (r *RemoteDevelopment) getSecretName() string {
	return SecretName
}

func (r *RemoteDevelopment) getPVCName() (string, error) {
	resource, err := r.getResource()
	if err != nil {
		return "", err
	}

	return fmt.Sprintf(PVCNameFormat, r.resourceType, resource.GetName()), nil
}

func (r *RemoteDevelopment) ensureSecret() error {
	r.StartSpinner(" Setup k8s secret")
	defer r.spinner.Stop()

	sshPublicKeyData, err := os.ReadFile(r.sshPublicKeyPath)
	if err != nil {
		return err
	}

	resource, err := r.getResource()
	if err != nil {
		return err
	}

	namespace := resource.GetNamespace()

	labels := make(map[string]string)
	labels[MetadataActive] = "true"
	labels[MetadataService] = resource.GetName()

	secretData := make(map[string][]byte)
	secretData[SecretAuthorizedKeysKeyName] = sshPublicKeyData

	secret := applyCoreV1.Secret(r.getSecretName(), namespace).WithLabels(labels).WithData(secretData)
	return r.kubernetesClient.ApplySecret(secret)
}

func (r *RemoteDevelopment) deletePVC() error {
	resource, err := r.getResource()
	if err != nil {
		return err
	}

	pvcName, err := r.getPVCName()
	if err != nil {
		return err
	}
	return r.kubernetesClient.DeletePVC(resource.GetNamespace(), pvcName)
}

func (r *RemoteDevelopment) getResourceSelector() (*apiMetaV1.LabelSelector, error) {
	switch r.resourceType {
	case Deployment:
		return r.deployment.Spec.Selector, nil
	case StatefulSet:
		return r.statefulSet.Spec.Selector, nil
	case DaemonSet:
		return r.daemonSet.Spec.Selector, nil
	default:
		return nil, r.resourceTypeNotSupportedError()
	}
}

func (r *RemoteDevelopment) waitPodReady() error {
	r.StartSpinner(" Waiting for pod to be ready")
	defer r.StopSpinner()

	resource, err := r.getResource()
	if err != nil {
		return err
	}

	resourceSelector, err := r.getResourceSelector()
	if err != nil {
		return err
	}

	namespace := resource.GetNamespace()
	labelSelector := apiMetaV1.LabelSelector{MatchLabels: resourceSelector.MatchLabels}
	listOptions := apiMetaV1.ListOptions{
		LabelSelector: labels.Set(labelSelector.MatchLabels).String(),
	}

	timeout := int64(120)
	startTimestamp := time.Now().Unix()
	for {
		podList, err := r.kubernetesClient.ListPods(namespace, listOptions)
		if err != nil {
			return err
		}

		for _, pod := range podList.Items {
			if pod.DeletionTimestamp != nil || pod.Status.Phase != coreV1.PodRunning {
				continue
			}

			for _, containerStatus := range pod.Status.ContainerStatuses {
				if containerStatus.Name == r.container.Name && containerStatus.Ready {
					return nil
				}
			}
		}

		time.Sleep(1 * time.Second)
		nowTimestamp := time.Now().Unix()
		if nowTimestamp-startTimestamp >= timeout {
			break
		}
	}

	// timeout reached
	return fmt.Errorf("pod not ready")
}

func (r *RemoteDevelopment) getResourceContainers() ([]coreV1.Container, error) {
	switch r.resourceType {
	case Deployment:
		return k8sTools.GetDeploymentContainers(r.deployment), nil
	case StatefulSet:
		return k8sTools.GetStatefulSetContainers(r.statefulSet), nil
	case DaemonSet:
		return k8sTools.GetDaemonSetContainers(r.daemonSet), nil
	default:
		return []coreV1.Container{}, r.resourceTypeNotSupportedError()
	}
}

func (r *RemoteDevelopment) getResourceContainer(containerName string) (*coreV1.Container, error) {
	switch r.resourceType {
	case Deployment:
		return k8sTools.GetDeploymentContainerByName(r.deployment, containerName)
	case StatefulSet:
		return k8sTools.GetStatefulSetContainerByName(r.statefulSet, containerName)
	case DaemonSet:
		return k8sTools.GetDaemonSetContainerByName(r.daemonSet, containerName)
	default:
		return nil, ErrNoResourceSelected
	}
}
