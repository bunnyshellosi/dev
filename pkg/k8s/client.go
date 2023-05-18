package k8s

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"

	"bunnyshell.com/dev/pkg/util"

	appsV1 "k8s.io/api/apps/v1"
	coreV1 "k8s.io/api/core/v1"
	apiMetaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	applyCoreV1 "k8s.io/client-go/applyconfigurations/core/v1"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"
)

const (
	PortForwardMethod = "POST"

	BunnyshellRemoteDevFieldManager = "bunnyshell-dev"
)

type PortForwardOptions struct {
	Interface string

	RemotePort int
	LocalPort  int

	StopChannel  chan struct{}
	ReadyChannel chan struct{}
}

func NewPortForwardOptions(iface string, remotePort, localPort int) *PortForwardOptions {
	return &PortForwardOptions{
		Interface: iface,

		RemotePort: remotePort,
		LocalPort:  localPort,

		StopChannel:  make(chan struct{}),
		ReadyChannel: make(chan struct{}, 1),
	}
}

type KubernetesClient struct {
	kubeConfigPath string

	config     clientcmd.ClientConfig
	restConfig *rest.Config
	clientSet  *kubernetes.Clientset
}

func NewKubernetesClient(kubeConfigPath string) (*KubernetesClient, error) {
	newKubernetes := new(KubernetesClient)

	kubeconfig, err := os.ReadFile(kubeConfigPath)
	if err != nil {
		return newKubernetes, err
	}

	config, err := clientcmd.NewClientConfigFromBytes(kubeconfig)
	if err != nil {
		return newKubernetes, err
	}

	restConfig, err := config.ClientConfig()
	if err != nil {
		return newKubernetes, err
	}

	clientset, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return newKubernetes, err
	}

	newKubernetes.kubeConfigPath = kubeConfigPath
	newKubernetes.config = config
	newKubernetes.restConfig = restConfig
	newKubernetes.clientSet = clientset

	return newKubernetes, nil
}

func (k *KubernetesClient) GetKubeConfigNamespace() (string, error) {
	namespace, _, err := k.config.Namespace()
	return namespace, err
}

func (k *KubernetesClient) UpdateDeployment(namespace string, deployment *appsV1.Deployment) (*appsV1.Deployment, error) {
	return k.clientSet.AppsV1().Deployments(namespace).Update(context.TODO(), deployment, apiMetaV1.UpdateOptions{})
}

func (k *KubernetesClient) UpdateStatefulSet(namespace string, statefulSet *appsV1.StatefulSet) (*appsV1.StatefulSet, error) {
	return k.clientSet.AppsV1().StatefulSets(namespace).Update(context.TODO(), statefulSet, apiMetaV1.UpdateOptions{})
}

func (k *KubernetesClient) UpdateDaemonSet(namespace string, daemonSet *appsV1.DaemonSet) (*appsV1.DaemonSet, error) {
	return k.clientSet.AppsV1().DaemonSets(namespace).Update(context.TODO(), daemonSet, apiMetaV1.UpdateOptions{})
}

func (k *KubernetesClient) ListNamespaces() (*coreV1.NamespaceList, error) {
	return k.clientSet.CoreV1().Namespaces().List(context.TODO(), apiMetaV1.ListOptions{})
}

func (k *KubernetesClient) ListDeployments(namespace string) (*appsV1.DeploymentList, error) {
	return k.clientSet.AppsV1().Deployments(namespace).List(context.TODO(), apiMetaV1.ListOptions{})
}

func (k *KubernetesClient) ListStatefulSets(namespace string) (*appsV1.StatefulSetList, error) {
	return k.clientSet.AppsV1().StatefulSets(namespace).List(context.TODO(), apiMetaV1.ListOptions{})
}

func (k *KubernetesClient) ListDaemonSets(namespace string) (*appsV1.DaemonSetList, error) {
	return k.clientSet.AppsV1().DaemonSets(namespace).List(context.TODO(), apiMetaV1.ListOptions{})
}

func (k *KubernetesClient) DeletePVC(namespace, name string) error {
	return k.clientSet.CoreV1().PersistentVolumeClaims(namespace).Delete(context.TODO(), name, apiMetaV1.DeleteOptions{})
}

func (k *KubernetesClient) DeleteSecret(namespace, name string) error {
	return k.clientSet.CoreV1().Secrets(namespace).Delete(context.TODO(), name, apiMetaV1.DeleteOptions{})
}

func (k *KubernetesClient) GetNamespace(name string) (*coreV1.Namespace, error) {
	return k.clientSet.CoreV1().Namespaces().Get(context.TODO(), name, apiMetaV1.GetOptions{})
}

func (k *KubernetesClient) GetDeployment(namespace, deploymentName string) (*appsV1.Deployment, error) {
	return k.clientSet.AppsV1().Deployments(namespace).Get(context.TODO(), deploymentName, apiMetaV1.GetOptions{})
}

func (k *KubernetesClient) GetStatefulSet(namespace, name string) (*appsV1.StatefulSet, error) {
	return k.clientSet.AppsV1().StatefulSets(namespace).Get(context.TODO(), name, apiMetaV1.GetOptions{})
}

func (k *KubernetesClient) GetDaemonSet(namespace, name string) (*appsV1.DaemonSet, error) {
	return k.clientSet.AppsV1().DaemonSets(namespace).Get(context.TODO(), name, apiMetaV1.GetOptions{})
}

func (k *KubernetesClient) ApplySecret(secret *applyCoreV1.SecretApplyConfiguration) error {
	applyOptions := apiMetaV1.ApplyOptions{
		FieldManager: BunnyshellRemoteDevFieldManager,
	}
	_, err := k.clientSet.CoreV1().Secrets(*secret.Namespace).Apply(context.TODO(), secret, applyOptions)
	return err
}

func (k *KubernetesClient) ApplyPVC(pvc *applyCoreV1.PersistentVolumeClaimApplyConfiguration) error {
	applyOptions := apiMetaV1.ApplyOptions{
		FieldManager: BunnyshellRemoteDevFieldManager,
	}
	_, err := k.clientSet.CoreV1().PersistentVolumeClaims(*pvc.Namespace).Apply(context.TODO(), pvc, applyOptions)
	return err
}

func (k *KubernetesClient) PatchDeployment(namespace, name string, data []byte) error {
	_, err := k.clientSet.AppsV1().Deployments(namespace).Patch(context.TODO(), name, types.StrategicMergePatchType, data, apiMetaV1.PatchOptions{})
	return err
}

func (k *KubernetesClient) PatchStatefulSet(namespace, name string, data []byte) error {
	_, err := k.clientSet.AppsV1().StatefulSets(namespace).Patch(context.TODO(), name, types.StrategicMergePatchType, data, apiMetaV1.PatchOptions{})
	return err
}

func (k *KubernetesClient) PatchDaemonSet(namespace, name string, data []byte) error {
	_, err := k.clientSet.AppsV1().DaemonSets(namespace).Patch(context.TODO(), name, types.StrategicMergePatchType, data, apiMetaV1.PatchOptions{})
	return err
}

func (k *KubernetesClient) BatchPatchDeployment(namespace, name string, data []byte) error {
	_, err := k.clientSet.AppsV1().Deployments(namespace).Patch(context.TODO(), name, types.JSONPatchType, data, apiMetaV1.PatchOptions{})

	return err
}

func (k *KubernetesClient) BatchPatchStatefulSet(namespace, name string, data []byte) error {
	_, err := k.clientSet.AppsV1().StatefulSets(namespace).Patch(context.TODO(), name, types.JSONPatchType, data, apiMetaV1.PatchOptions{})
	return err
}

func (k *KubernetesClient) BatchPatchDaemonSet(namespace, name string, data []byte) error {
	_, err := k.clientSet.AppsV1().DaemonSets(namespace).Patch(context.TODO(), name, types.JSONPatchType, data, apiMetaV1.PatchOptions{})
	return err
}

func (k *KubernetesClient) ListPods(namespace string, listOptions apiMetaV1.ListOptions) (*coreV1.PodList, error) {
	return k.clientSet.CoreV1().Pods(namespace).List(context.TODO(), listOptions)
}

func (k *KubernetesClient) GetPortForwardSubresourceURL(pod *coreV1.Pod) *url.URL {
	return k.clientSet.CoreV1().RESTClient().Post().
		Resource("pods").
		Namespace(pod.Namespace).
		Name(pod.Name).
		SubResource("portforward").URL()
}

func (k *KubernetesClient) PortForward(pod *coreV1.Pod, portForwardOptions *PortForwardOptions) (*portforward.PortForwarder, error) {
	transport, upgrader, err := spdy.RoundTripperFor(k.restConfig)
	if err != nil {
		return nil, err
	}

	url := k.GetPortForwardSubresourceURL(pod)
	dialer := spdy.NewDialer(upgrader, &http.Client{Transport: transport}, PortForwardMethod, url)

	if portForwardOptions.LocalPort == 0 {
		portForwardOptions.LocalPort, err = util.GetAvailableEphemeralPort(portForwardOptions.Interface)
		if err != nil {
			return nil, err
		}
	}
	ports := []string{fmt.Sprintf(
		"%d:%d",
		portForwardOptions.LocalPort,
		portForwardOptions.RemotePort,
	)}

	forwarder, err := portforward.NewOnAddresses(
		dialer,
		[]string{portForwardOptions.Interface},
		ports,
		portForwardOptions.StopChannel,
		portForwardOptions.ReadyChannel,
		io.Discard,
		io.Discard,
	)
	if err != nil {
		return nil, err
	}

	errChan := make(chan error, 1)
	go func() {
		errChan <- forwarder.ForwardPorts()
		close(errChan)
	}()

	select {
	case <-forwarder.Ready:
	case err := <-errChan:
		return nil, err
	}

	return forwarder, nil
}

func (k *KubernetesClient) WatchPods(namespace string, listOptions apiMetaV1.ListOptions) (watch.Interface, error) {
	return k.clientSet.CoreV1().Pods(namespace).Watch(context.TODO(), listOptions)
}
