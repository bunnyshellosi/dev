package remote

import (
	"fmt"

	coreV1 "k8s.io/api/core/v1"
	apiMetaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

func (r *RemoteDevelopment) ensureRemoteSSHPortForward() error {
	r.StartSpinner(" Start Remote SSH Port Forward")
	defer r.Spinner.Stop()

	remoteDevPod, err := r.getRemoteDevPod()
	if err != nil {
		return err
	}

	forwarder, err := r.KubernetesClient.PortForwardRemoteSSH(remoteDevPod)
	if err != nil {
		return err
	}
	r.RemoteSSHForwarder = forwarder

	return nil
}

func (r *RemoteDevelopment) getRemoteDevPod() (*coreV1.Pod, error) {
	namespace := r.Deployment.GetNamespace()
	labelSelector := apiMetaV1.LabelSelector{MatchLabels: r.Deployment.Spec.Selector.MatchLabels}
	listOptions := apiMetaV1.ListOptions{
		LabelSelector: labels.Set(labelSelector.MatchLabels).String(),
	}

	podList, err := r.KubernetesClient.ListPods(namespace, listOptions)
	if err != nil {
		return nil, err
	}

	for _, pod := range podList.Items {
		if pod.DeletionTimestamp == nil && pod.Status.Phase == coreV1.PodRunning {
			return &pod, nil
		}
	}

	return nil, fmt.Errorf("pod not found for component %v", r.Deployment.GetName())
}
