/*
Copyright The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package pod

import (
	"fmt"
	"strconv"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"

	kueuealpha "sigs.k8s.io/kueue/apis/kueue/v1alpha1"
	"sigs.k8s.io/kueue/pkg/constants"
	controllerconsts "sigs.k8s.io/kueue/pkg/controller/constants"
	podconstants "sigs.k8s.io/kueue/pkg/controller/jobs/pod/constants"
	"sigs.k8s.io/kueue/pkg/util/testing"
)

// PodWrapper wraps a Pod.
type PodWrapper struct {
	corev1.Pod
}

// MakePod creates a wrapper for a pod with a single container.
func MakePod(name, ns string) *PodWrapper {
	return &PodWrapper{corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   ns,
			Annotations: make(map[string]string, 1),
		},
		Spec: corev1.PodSpec{
			RestartPolicy: corev1.RestartPolicyNever,
			Containers: []corev1.Container{
				{
					Name:      "c",
					Image:     "pause",
					Resources: corev1.ResourceRequirements{Requests: corev1.ResourceList{}, Limits: corev1.ResourceList{}},
				},
			},
			SchedulingGates: make([]corev1.PodSchedulingGate, 0),
		},
	}}
}

// Obj returns the inner Pod.
func (p *PodWrapper) Obj() *corev1.Pod {
	return &p.Pod
}

// MakeGroup returns multiple pods that form a pod group, based on the original wrapper.
func (p *PodWrapper) MakeGroup(count int) []*corev1.Pod {
	var pods []*corev1.Pod
	for i := range count {
		pod := p.Clone().Group(p.Pod.Name).GroupTotalCount(strconv.Itoa(count))
		pod.Pod.Name += fmt.Sprintf("-%d", i)
		pods = append(pods, pod.Obj())
	}
	return pods
}

func (p *PodWrapper) MakePodGroupWrappers(count int) []*PodWrapper {
	var pods []*PodWrapper
	for i := range count {
		pod := p.Clone().Group(p.Pod.Name).GroupTotalCount(strconv.Itoa(count))
		pod.Pod.Name += fmt.Sprintf("-%d", i)
		pods = append(pods, pod)
	}
	return pods
}

// MakeIndexedGroup returns multiple indexed pods that form a pod group, based on the original wrapper.
func (p *PodWrapper) MakeIndexedGroup(count int) []*corev1.Pod {
	var pods []*corev1.Pod
	for i := range count {
		pod := p.Clone().
			Group(p.Pod.Name).
			GroupTotalCount(strconv.Itoa(count)).
			GroupIndex(strconv.Itoa(i))
		pod.Pod.Name += fmt.Sprintf("-%d", i)
		pods = append(pods, pod.Obj())
	}
	return pods
}

// Clone returns deep copy of the Pod.
func (p *PodWrapper) Clone() *PodWrapper {
	return &PodWrapper{Pod: *p.DeepCopy()}
}

// Queue updates the queue name of the Pod
func (p *PodWrapper) Queue(q string) *PodWrapper {
	return p.Label(controllerconsts.QueueLabel, q)
}

func (p *PodWrapper) PrebuiltWorkload(name string) *PodWrapper {
	return p.Label(controllerconsts.PrebuiltWorkloadLabel, name)
}

// PriorityClass updates the priority class name of the Pod
func (p *PodWrapper) PriorityClass(pc string) *PodWrapper {
	p.Spec.PriorityClassName = pc
	return p
}

// Name updated the name of the pod
func (p *PodWrapper) Name(n string) *PodWrapper {
	p.ObjectMeta.Name = n
	return p
}

// Namespace updates the namespace of the Pod.
func (p *PodWrapper) Namespace(n string) *PodWrapper {
	p.ObjectMeta.Namespace = n
	return p
}

// Group updates the pod.GroupNameLabel of the Pod
func (p *PodWrapper) Group(g string) *PodWrapper {
	return p.Label(podconstants.GroupNameLabel, g)
}

// GroupTotalCount updates the pod.GroupTotalCountAnnotation of the Pod
func (p *PodWrapper) GroupTotalCount(gtc string) *PodWrapper {
	return p.Annotation(podconstants.GroupTotalCountAnnotation, gtc)
}

// GroupIndex updates the pod.GroupIndexLabel of the Pod
func (p *PodWrapper) GroupIndex(index string) *PodWrapper {
	return p.Label(kueuealpha.PodGroupPodIndexLabel, index)
}

// Label sets the label of the Pod
func (p *PodWrapper) Label(k, v string) *PodWrapper {
	if p.Labels == nil {
		p.Labels = make(map[string]string)
	}
	p.Labels[k] = v
	return p
}

func (p *PodWrapper) Annotation(key, content string) *PodWrapper {
	p.Annotations[key] = content
	return p
}

func (p *PodWrapper) ManagedByKueueLabel() *PodWrapper {
	return p.Label(constants.ManagedByKueueLabelKey, constants.ManagedByKueueLabelValue)
}

func (p *PodWrapper) PodGroupServingAnnotation() *PodWrapper {
	return p.Annotation(podconstants.GroupServingAnnotationKey, podconstants.GroupServingAnnotationValue)
}

// RoleHash updates the pod.RoleHashAnnotation of the pod
func (p *PodWrapper) RoleHash(h string) *PodWrapper {
	return p.Annotation(podconstants.RoleHashAnnotation, h)
}

// KueueSchedulingGate adds kueue scheduling gate to the Pod
func (p *PodWrapper) KueueSchedulingGate() *PodWrapper {
	return p.Gate(podconstants.SchedulingGateName)
}

// TopologySchedulingGate adds kueue scheduling gate to the Pod
func (p *PodWrapper) TopologySchedulingGate() *PodWrapper {
	return p.Gate(kueuealpha.TopologySchedulingGate)
}

// Gate adds kueue scheduling gate to the Pod by the gate name
func (p *PodWrapper) Gate(gateNames ...string) *PodWrapper {
	for _, gate := range gateNames {
		p.Spec.SchedulingGates = append(p.Spec.SchedulingGates, corev1.PodSchedulingGate{
			Name: gate,
		})
	}
	return p
}

// Finalizer adds a finalizer to the Pod
func (p *PodWrapper) Finalizer(f string) *PodWrapper {
	if p.Finalizers == nil {
		p.Finalizers = make([]string, 0)
	}
	p.Finalizers = append(p.Finalizers, f)
	return p
}

// KueueFinalizer adds kueue finalizer to the Pod
func (p *PodWrapper) KueueFinalizer() *PodWrapper {
	return p.Finalizer(constants.ManagedByKueueLabelKey)
}

// NodeSelector adds a node selector to the Pod.
func (p *PodWrapper) NodeSelector(k, v string) *PodWrapper {
	if p.Spec.NodeSelector == nil {
		p.Spec.NodeSelector = make(map[string]string, 1)
	}

	p.Spec.NodeSelector[k] = v
	return p
}

// NodeName sets a node name to the Pod.
func (p *PodWrapper) NodeName(name string) *PodWrapper {
	p.Spec.NodeName = name
	return p
}

// Request adds a resource request to the default container.
func (p *PodWrapper) Request(r corev1.ResourceName, v string) *PodWrapper {
	p.Spec.Containers[0].Resources.Requests[r] = resource.MustParse(v)
	return p
}

// RequestAndLimit adds a resource request and limit to the default container.
func (p *PodWrapper) RequestAndLimit(r corev1.ResourceName, v string) *PodWrapper {
	return p.Request(r, v).Limit(r, v)
}

func (p *PodWrapper) ServiceAccountName(serviceAccountName string) *PodWrapper {
	p.Spec.ServiceAccountName = serviceAccountName
	return p
}

func (p *PodWrapper) Image(image string, args []string) *PodWrapper {
	p.Spec.Containers[0].Image = image
	p.Spec.Containers[0].Args = args
	return p
}

// Limit adds a resource limit to the default container.
func (p *PodWrapper) Limit(r corev1.ResourceName, v string) *PodWrapper {
	p.Spec.Containers[0].Resources.Limits[r] = resource.MustParse(v)
	return p
}

// OwnerReference adds a ownerReference to the default container.
func (p *PodWrapper) OwnerReference(ownerName string, ownerGVK schema.GroupVersionKind) *PodWrapper {
	testing.AppendOwnerReference(&p.Pod, ownerGVK, ownerName, ownerName, ptr.To(true), ptr.To(true))
	return p
}

// UID updates the uid of the Pod.
func (p *PodWrapper) UID(uid string) *PodWrapper {
	p.ObjectMeta.UID = types.UID(uid)
	return p
}

// StatusConditions updates status conditions of the Pod.
func (p *PodWrapper) StatusConditions(conditions ...corev1.PodCondition) *PodWrapper {
	p.Status.Conditions = conditions
	return p
}

// StatusPhase updates status phase of the Pod.
func (p *PodWrapper) StatusPhase(ph corev1.PodPhase) *PodWrapper {
	p.Status.Phase = ph
	return p
}

// StatusMessage updates status message of the Pod.
func (p *PodWrapper) StatusMessage(msg string) *PodWrapper {
	p.Status.Message = msg
	return p
}

// CreationTimestamp sets a creation timestamp for the pod object
func (p *PodWrapper) CreationTimestamp(t time.Time) *PodWrapper {
	timestamp := metav1.NewTime(t).Rfc3339Copy()
	p.Pod.CreationTimestamp = timestamp
	return p
}

// DeletionTimestamp sets a creation timestamp for the pod object
func (p *PodWrapper) DeletionTimestamp(t time.Time) *PodWrapper {
	timestamp := metav1.NewTime(t).Rfc3339Copy()
	p.Pod.DeletionTimestamp = &timestamp
	return p
}

// Delete sets a deletion timestamp for the pod object
func (p *PodWrapper) Delete() *PodWrapper {
	t := metav1.NewTime(time.Now())
	p.Pod.DeletionTimestamp = &t
	return p
}

// Volume adds a new volume for the pod object
func (p *PodWrapper) Volume(v corev1.Volume) *PodWrapper {
	p.Spec.Volumes = append(p.Spec.Volumes, v)
	return p
}

// TerminationGracePeriod sets terminationGracePeriodSeconds for the pod object
func (p *PodWrapper) TerminationGracePeriod(seconds int64) *PodWrapper {
	p.Spec.TerminationGracePeriodSeconds = &seconds
	return p
}
