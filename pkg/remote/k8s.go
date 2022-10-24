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
	PVCNameFormat = "%s-remote-dev"

	SecretAuthorizedKeysKeyName = "authorized_keys"
	SecretAuthorizedKeysPath    = "ssh/authorized_keys"

	ContainerNameBinaries = "remote-dev-bin"
	ContainerNameWork     = "remote-dev-work"

	// ConfigSourceDir = "config"
)

func (r *RemoteDevelopment) prepareDeployment() error {
	r.StartSpinner(" Setup k8s pod for remote development")
	defer r.StopSpinner()

	strategy := appsV1.RecreateDeploymentStrategyType
	var replicas int32 = 1
	deploymentPatch := &patch.DeploymentPatchConfiguration{
		ObjectMetaApplyConfiguration: &applyMetaV1.ObjectMetaApplyConfiguration{},
		Spec: &patch.DeploymentSpecPatchConfiguration{
			Strategy: &patch.DeploymentStrategyPatchConfiguration{
				Type:          &strategy,
				RollingUpdate: nil,
			},
			Replicas: &replicas,
		},
	}

	currentManifestSnapshot, err := r.getCurrentManifestSnapshot()
	if err != nil {
		return err
	}

	annotations := make(map[string]string)
	annotations[MetadataStartedAt] = strconv.FormatInt(r.startedAt, 10)
	annotations[MetadataContainer] = r.container.Name
	_, ok := r.deployment.Annotations[MetadataRollback]
	if !ok {
		annotations[MetadataRollback] = string(currentManifestSnapshot)
	}
	labels := make(map[string]string)
	labels[MetadataActive] = "true"

	deploymentPatch.WithAnnotations(annotations).WithLabels(labels)

	r.preparePodTemplateSpec(deploymentPatch)

	data, err := json.Marshal(deploymentPatch)
	if err != nil {
		return err
	}

	return r.kubernetesClient.PatchDeployment(r.deployment.GetNamespace(), r.deployment.GetName(), data)
}

func (r *RemoteDevelopment) restoreDeployment() error {
	snapshot, ok := r.deployment.Annotations[MetadataRollback]
	if !ok {
		return fmt.Errorf("no rollback manifest available")
	}

	deployment := &appsV1.Deployment{}
	if err := json.Unmarshal([]byte(snapshot), deployment); err != nil {
		return err
	}

	_, err := r.kubernetesClient.UpdateDeployment(deployment.GetNamespace(), deployment)
	return err
}

func (r *RemoteDevelopment) getCurrentManifestSnapshot() (string, error) {
	// if snapshot, ok := r.Deployment.Annotations[MetadataKubeCTLLastAppliedConf]; ok {
	// 	return snapshot
	// }

	fullSnapshot, err := json.Marshal(r.deployment)
	if err != nil {
		return "", err
	}

	applyDeployment := &appsCoreV1.DeploymentApplyConfiguration{}
	if err := json.Unmarshal(fullSnapshot, applyDeployment); err != nil {
		return "", err
	}

	// strip unnecessary data
	applyDeployment.WithStatus(nil)
	applyDeployment.Generation = nil
	applyDeployment.UID = nil
	applyDeployment.ResourceVersion = nil
	annotations := make(map[string]string)
	for key, value := range applyDeployment.Annotations {
		if key == MetadataK8SRevision || key == MetadataKubeCTLLastAppliedConf {
			continue
		}

		annotations[key] = value
	}
	applyDeployment.Annotations = annotations

	snapshot, err := json.Marshal(applyDeployment)
	if err != nil {
		return "", err
	}

	return string(snapshot), nil
}

func (r *RemoteDevelopment) ensurePVC() error {
	labels := make(map[string]string)
	labels[MetadataActive] = "true"

	resourceLimits := coreV1.ResourceList{
		coreV1.ResourceStorage: resource.MustParse("2Gi"),
	}

	remoteDevPVC := applyCoreV1.PersistentVolumeClaim(r.getPVCName(), r.deployment.GetNamespace()).
		WithLabels(labels).
		WithSpec(applyCoreV1.PersistentVolumeClaimSpec().
			WithAccessModes(coreV1.ReadWriteOnce).
			WithResources(applyCoreV1.ResourceRequirements().
				WithRequests(resourceLimits)))

	return r.kubernetesClient.ApplyPVC(remoteDevPVC)
}

func (r *RemoteDevelopment) preparePodTemplateSpec(deploymentPatch *patch.DeploymentPatchConfiguration) {
	podAnnotations := make(map[string]string)
	podAnnotations[MetadataStartedAt] = strconv.FormatInt(r.startedAt, 10)
	podAnnotations[MetadataContainer] = r.container.Name
	podLabels := make(map[string]string)
	podLabels[MetadataActive] = "true"
	podLabels[MetadataService] = r.deployment.GetName()

	deploymentPatch.Spec.Template = applyCoreV1.PodTemplateSpec().
		WithAnnotations(podAnnotations).
		WithLabels(podLabels)

	r.preparePodSpec(deploymentPatch)
}

func (r *RemoteDevelopment) preparePodSpec(deploymentPatch *patch.DeploymentPatchConfiguration) {
	deploymentPatch.Spec.Template.WithSpec(applyCoreV1.PodSpec())
	r.prepareVolumes(deploymentPatch)
	r.prepareInitContainers(deploymentPatch)
	r.prepareContainer(deploymentPatch)
}

func (r *RemoteDevelopment) prepareVolumes(deploymentPatch *patch.DeploymentPatchConfiguration) {
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

	workVolume := applyCoreV1.Volume().
		WithName(VolumeNameWork).
		WithPersistentVolumeClaim(applyCoreV1.PersistentVolumeClaimVolumeSource().
			WithClaimName(r.getPVCName()))
	volumes = append(volumes, workVolume)

	deploymentPatch.Spec.Template.Spec.WithVolumes(volumes...)
}

func (r *RemoteDevelopment) prepareInitContainers(deploymentPatch *patch.DeploymentPatchConfiguration) {
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

	appSourceDir := r.getRemoteSyncPathHash()
	workVolumeMountPath := "/volumes/" + appSourceDir
	workInitContainer := applyCoreV1.Container().
		WithName(ContainerNameWork).
		WithCommand("sh", "-c", fmt.Sprintf(
			"[ \"$(ls -A %s)\" ] || (cp -Rp %s/. %s; exit 0)",
			workVolumeMountPath,
			r.remoteSyncPath,
			workVolumeMountPath,
		)).
		WithImage(r.container.Image).
		WithImagePullPolicy(coreV1.PullIfNotPresent).
		WithVolumeMounts(applyCoreV1.VolumeMount().
			WithName(VolumeNameWork).
			WithMountPath(workVolumeMountPath).
			WithSubPath(appSourceDir))

	deploymentPatch.Spec.Template.Spec.WithInitContainers(binariesInitContainer, workInitContainer)
}

func (r *RemoteDevelopment) getSSHServerImage() string {
	return fmt.Sprintf("%s:%s", build.SSHServerImage, build.SSHServerVersion)
}

func (r *RemoteDevelopment) getRemoteSyncPathHash() string {
	hash := md5.Sum([]byte(r.remoteSyncPath))
	return hex.EncodeToString(hash[:])
}

func (r *RemoteDevelopment) prepareContainer(deploymentPatch *patch.DeploymentPatchConfiguration) {
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

	startCommand := binariesVolumeMountPath + "/start.sh"
	container := applyCoreV1.Container().
		WithName(r.container.Name).
		WithCommand(startCommand).
		WithVolumeMounts(volumeMounts...)

	deploymentPatch.Spec.Template.Spec.WithContainers(container)
}

func (r *RemoteDevelopment) getSecretName() string {
	return SecretName
}

func (r *RemoteDevelopment) getPVCName() string {
	return fmt.Sprintf(PVCNameFormat, r.deployment.GetName())
}

func (r *RemoteDevelopment) ensureSecret() error {
	r.StartSpinner(" Setup k8s secret")
	defer r.spinner.Stop()

	sshPublicKeyData, err := os.ReadFile(r.sshPublicKeyPath)
	if err != nil {
		return err
	}

	namespace := r.deployment.GetNamespace()

	labels := make(map[string]string)
	labels[MetadataActive] = "true"
	labels[MetadataService] = r.deployment.GetName()

	secretData := make(map[string][]byte)
	secretData[SecretAuthorizedKeysKeyName] = sshPublicKeyData

	secret := applyCoreV1.Secret(r.getSecretName(), namespace).WithLabels(labels).WithData(secretData)
	return r.kubernetesClient.ApplySecret(secret)
}

func (r *RemoteDevelopment) deletePVC() error {
	return r.kubernetesClient.DeletePVC(r.deployment.GetNamespace(), r.getPVCName())
}

func (r *RemoteDevelopment) deleteSecret() error {
	return r.kubernetesClient.DeleteSecret(r.deployment.GetNamespace(), r.getSecretName())
}

func (r *RemoteDevelopment) waitPodReady() error {
	r.StartSpinner(" Waiting for pod to be ready")
	defer r.StopSpinner()

	namespace := r.deployment.GetNamespace()
	labelSelector := apiMetaV1.LabelSelector{MatchLabels: r.deployment.Spec.Selector.MatchLabels}
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
