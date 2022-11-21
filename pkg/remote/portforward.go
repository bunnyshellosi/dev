package remote

import (
	"fmt"

	"bunnyshell.com/dev/pkg/k8s"
	"bunnyshell.com/dev/pkg/ssh"

	coreV1 "k8s.io/api/core/v1"
	apiMetaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

const (
	SSHPortForwardInterface  = "127.0.0.1"
	SSHPortForwardRemotePort = 2222
)

func (r *RemoteDevelopment) ensureRemoteSSHPortForward() error {
	r.StartSpinner(" Start Remote SSH Port Forward")
	defer r.spinner.Stop()

	remoteDevPod, err := r.getRemoteDevPod()
	if err != nil {
		return err
	}

	r.sshPortForwardOptions = k8s.NewPortForwardOptions(SSHPortForwardInterface, SSHPortForwardRemotePort, 0)
	forwarder, err := r.kubernetesClient.PortForward(remoteDevPod, r.sshPortForwardOptions)
	if err != nil {
		return err
	}
	r.sshPortForwarder = forwarder

	return nil
}

func (r *RemoteDevelopment) startSSHTunnels() error {
	for i := range r.sshTunnels {
		serverEndpoint := ssh.NewEndpoint(r.sshPortForwardOptions.Interface, r.sshPortForwardOptions.LocalPort)
		auth, err := ssh.PrivateKeyFile(r.sshPrivateKeyPath)
		if err != nil {
			panic(err)
		}
		r.sshTunnels[i].WithSSHServerEndpoint(serverEndpoint).WithAuths(auth)
		if err := r.sshTunnels[i].Start(); err != nil {
			return err
		}
	}

	return nil
}

func (r *RemoteDevelopment) getRemoteDevPod() (*coreV1.Pod, error) {
	resource, err := r.getResource()
	if err != nil {
		return nil, err
	}

	resourceSelector, err := r.getResourceSelector()
	if err != nil {
		return nil, err
	}

	namespace := resource.GetNamespace()
	labelSelector := apiMetaV1.LabelSelector{MatchLabels: resourceSelector.MatchLabels}
	listOptions := apiMetaV1.ListOptions{
		LabelSelector: labels.Set(labelSelector.MatchLabels).String(),
	}

	podList, err := r.kubernetesClient.ListPods(namespace, listOptions)
	if err != nil {
		return nil, err
	}

	for _, pod := range podList.Items {
		if pod.DeletionTimestamp == nil && pod.Status.Phase == coreV1.PodRunning {
			return &pod, nil
		}
	}

	return nil, fmt.Errorf("pod not found for component %v", resource.GetName())
}
