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
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"
)

const (
	PortForwardMethod    = "POST"
	PortForwardInterface = "127.0.0.1"
	SSHRemotePort        = 2222

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

	SSHPortForwardOptions *PortForwardOptions
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
	newKubernetes.SSHPortForwardOptions = NewPortForwardOptions(PortForwardInterface, SSHRemotePort, 0)

	return newKubernetes, nil
}

func (k *KubernetesClient) GetKubeConfigNamespace() (string, error) {
	namespace, _, err := k.config.Namespace()
	return namespace, err
}

func (k *KubernetesClient) UpdateDeployment(namespace string, deployment *appsV1.Deployment) (*appsV1.Deployment, error) {
	return k.clientSet.AppsV1().Deployments(namespace).Update(context.TODO(), deployment, apiMetaV1.UpdateOptions{})
}

func (k *KubernetesClient) ListNamespaces() (*coreV1.NamespaceList, error) {
	return k.clientSet.CoreV1().Namespaces().List(context.TODO(), apiMetaV1.ListOptions{})
}

func (k *KubernetesClient) ListDeployments(namespace string) (*appsV1.DeploymentList, error) {
	return k.clientSet.AppsV1().Deployments(namespace).List(context.TODO(), apiMetaV1.ListOptions{})
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

// @todo better abstraction for portforward
func (k *KubernetesClient) PortForwardRemoteSSH(pod *coreV1.Pod) (*portforward.PortForwarder, error) {
	transport, upgrader, err := spdy.RoundTripperFor(k.restConfig)
	if err != nil {
		return nil, err
	}

	url := k.GetPortForwardSubresourceURL(pod)
	dialer := spdy.NewDialer(upgrader, &http.Client{Transport: transport}, PortForwardMethod, url)

	k.SSHPortForwardOptions.LocalPort, err = util.GetAvailableEphemeralPort(k.SSHPortForwardOptions.Interface)
	if err != nil {
		return nil, err
	}
	ports := []string{fmt.Sprintf(
		"%d:%d",
		k.SSHPortForwardOptions.LocalPort,
		k.SSHPortForwardOptions.RemotePort,
	)}

	forwarder, err := portforward.NewOnAddresses(
		dialer,
		[]string{k.SSHPortForwardOptions.Interface},
		ports,
		k.SSHPortForwardOptions.StopChannel,
		k.SSHPortForwardOptions.ReadyChannel,
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
