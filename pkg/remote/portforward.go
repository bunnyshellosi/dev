package remote

import (
	"fmt"

	coreV1 "k8s.io/api/core/v1"
	apiMetaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

func (r *RemoteDevelopment) ensureRemoteSSHPortForward() error {
	r.StartSpinner(" Start Remote SSH Port Forward")
	defer r.spinner.Stop()

	remoteDevPod, err := r.getRemoteDevPod()
	if err != nil {
		return err
	}

	forwarder, err := r.kubernetesClient.PortForwardRemoteSSH(remoteDevPod)
	if err != nil {
		return err
	}
	r.remoteSSHForwarder = forwarder

	return nil
}

func (r *RemoteDevelopment) getRemoteDevPod() (*coreV1.Pod, error) {
	resource, err := r.GetResource()
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
