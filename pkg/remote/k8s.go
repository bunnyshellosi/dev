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

	"bunnyshell.com/dev/pkg/k8s/patch"

	appsV1 "k8s.io/api/apps/v1"
	coreV1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	apiMetaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/watch"
	applyCoreV1 "k8s.io/client-go/applyconfigurations/core/v1"
)

const (
	MetadataPrefix    = "remote-dev.bunnyshell.com/"
	MetadataEnabled   = MetadataPrefix + "enabled"
	MetadataStartedAt = MetadataPrefix + "started-at"
	MetadataComponent = MetadataPrefix + "component"
	MetadataContainer = MetadataPrefix + "container"
	MetadataRollback  = MetadataPrefix + "rollback-manifest"

	VolumeNameBinaries = "remote-dev-bin"
	VolumeNameConfig   = "remote-dev-config"
	VolumeNameWork     = "remote-dev-work"

	SecretNameFormat = "%s-remote-dev"
	PVCNameFormat    = "%s-remote-dev"

	SecretAuthorizedKeysKeyName = "authorized_keys"
	SecretAuthorizedKeysPath    = "ssh/authorized_keys"

	ContainerNameBinaries = "remote-dev-bin"
	ContainerNameWork     = "remote-dev-work"

	// @todo inject this at build time maybe :?
	BinariesImage = "public.ecr.aws/x0p9x6p7/bunnyshell/remote-binaries:latest"

	// ConfigSourceDir = "config"
)

func (r *RemoteDevelopment) prepareDeployment() error {
	r.StartSpinner(" Setup k8s pod for remote development")
	defer r.StopSpinner()

	strategy := appsV1.RecreateDeploymentStrategyType
	var replicas int32 = 1
	deploymentPatch := &patch.DeploymentPatchConfiguration{
		Spec: &patch.DeploymentSpecPatchConfiguration{
			Strategy: &patch.DeploymentStrategyPatchConfiguration{
				Type:          &strategy,
				RollingUpdate: nil,
			},
			Replicas: &replicas,
		},
	}

	r.preparePodTemplateSpec(deploymentPatch)

	data, err := json.Marshal(deploymentPatch)
	if err != nil {
		return err
	}

	if err := r.ensureSecret(); err != nil {
		return err
	}

	if err := r.ensurePVC(); err != nil {
		return err
	}

	return r.KubernetesClient.PatchDeployment(r.Deployment.Namespace, r.Deployment.Name, data)
}

func (r *RemoteDevelopment) ensurePVC() error {
	labels := make(map[string]string)
	labels[MetadataContainer] = r.Container.Name
	resourceLimits := coreV1.ResourceList{
		coreV1.ResourceStorage: resource.MustParse("2Gi"),
	}
	remoteDevPVC := applyCoreV1.PersistentVolumeClaim(r.getPVCName(), r.Deployment.Namespace).
		WithLabels(labels).
		WithSpec(applyCoreV1.PersistentVolumeClaimSpec().
			WithAccessModes(coreV1.ReadWriteOnce).
			WithResources(applyCoreV1.ResourceRequirements().
				WithRequests(resourceLimits)))

	return r.KubernetesClient.ApplyPVC(remoteDevPVC)
}

func (r *RemoteDevelopment) preparePodTemplateSpec(deploymentPatch *patch.DeploymentPatchConfiguration) {
	podLabels := make(map[string]string)
	podLabels[MetadataEnabled] = "true"
	podLabels[MetadataComponent] = r.Deployment.GetName()

	podAnnotations := make(map[string]string)
	podAnnotations[MetadataStartedAt] = strconv.FormatInt(time.Now().Unix(), 10)
	podAnnotations[MetadataContainer] = r.Container.Name

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
	if strings.Contains(BinariesImage, ":latest") {
		pullPolicy = coreV1.PullAlways
	}

	binariesVolumeMountPath := "/remote-dev-bin"
	binariesInitContainer := applyCoreV1.Container().
		WithName(ContainerNameBinaries).
		WithCommand("sh", "-c", fmt.Sprintf("cp -p /usr/local/bin/* %s", binariesVolumeMountPath)).
		WithImage(BinariesImage).
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
			r.RemoteSyncPath,
			workVolumeMountPath,
		)).
		WithImage(r.Container.Image).
		WithImagePullPolicy(coreV1.PullIfNotPresent).
		WithVolumeMounts(applyCoreV1.VolumeMount().
			WithName(VolumeNameWork).
			WithMountPath(workVolumeMountPath).
			WithSubPath(appSourceDir))

	deploymentPatch.Spec.Template.Spec.WithInitContainers(binariesInitContainer, workInitContainer)
}

func (r *RemoteDevelopment) getRemoteSyncPathHash() string {
	hash := md5.Sum([]byte(r.RemoteSyncPath))
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
			WithMountPath(r.RemoteSyncPath).
			WithSubPath(appSourceDir),
		// applyCoreV1.VolumeMount().
		// 	WithName(VolumeNameWork).
		// 	WithMountPath(configVolumeMountPath).
		// 	WithSubPath(ConfigSourceDir),
	}

	startCommand := binariesVolumeMountPath + "/start.sh"
	container := applyCoreV1.Container().
		WithName(r.Container.Name).
		WithCommand(startCommand).
		WithVolumeMounts(volumeMounts...)

	deploymentPatch.Spec.Template.Spec.WithContainers(container)
}

func (r *RemoteDevelopment) getSecretName() string {
	return fmt.Sprintf(SecretNameFormat, r.Deployment.Name)
}

func (r *RemoteDevelopment) getPVCName() string {
	return fmt.Sprintf(PVCNameFormat, r.Container.Name)
}

func (r *RemoteDevelopment) ensureSecret() error {
	r.StartSpinner(" Setup k8s secret")
	defer r.Spinner.Stop()

	sshPublicKeyData, err := os.ReadFile(r.SSHPublicKeyPath)
	if err != nil {
		return err
	}

	namespace := r.Deployment.Namespace

	labels := make(map[string]string)
	labels[MetadataEnabled] = "true"
	labels[MetadataComponent] = r.Deployment.GetName()

	secretData := make(map[string][]byte)
	secretData[SecretAuthorizedKeysKeyName] = sshPublicKeyData

	secret := applyCoreV1.Secret(r.getSecretName(), namespace).WithLabels(labels).WithData(secretData)
	return r.KubernetesClient.ApplySecret(secret)
}

func (r *RemoteDevelopment) deleteRemoteDevPVC() error {
	return r.KubernetesClient.DeletePVC(r.getPVCName())
}

func (r *RemoteDevelopment) waitPodReady() error {
	r.StartSpinner(" Waiting for pod to be ready for remote development")
	defer r.StopSpinner()

	namespace := r.Deployment.GetNamespace()
	labelSelector := apiMetaV1.LabelSelector{MatchLabels: r.Deployment.Spec.Selector.MatchLabels}
	timeout := int64(120)
	listOptions := apiMetaV1.ListOptions{
		LabelSelector:  labels.Set(labelSelector.MatchLabels).String(),
		TimeoutSeconds: &timeout,
	}

	podList, err := r.KubernetesClient.ListPods(namespace, listOptions)
	if err != nil {
		return err
	}
	allRunning := len(podList.Items) > 0
	for _, pod := range podList.Items {
		if pod.DeletionTimestamp != nil || pod.Status.Phase != coreV1.PodRunning {
			allRunning = false
			break
		}
	}

	if allRunning {
		return nil
	}

	watcher, err := r.KubernetesClient.WatchPods(namespace, listOptions)
	if err != nil {
		return err
	}

	defer watcher.Stop()
	for event := range watcher.ResultChan() {
		pod := event.Object.(*coreV1.Pod)
		// ignore terminating pod
		if pod.DeletionTimestamp != nil {
			continue
		}

		if event.Type == watch.Added {
			continue
		}

		for _, containerStatus := range pod.Status.ContainerStatuses {
			if containerStatus.Name == r.Container.Name && containerStatus.Ready {
				return nil
			}
		}
	}

	// timeout reached
	return fmt.Errorf("failed to start remote development")
}
