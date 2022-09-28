/*
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

package state

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"sync/atomic"

	"go.uber.org/multierr"
	v1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/clock"
	"sigs.k8s.io/controller-runtime/pkg/client"

	pod2 "github.com/bwagner5/kube-demo/pkg/utils/pod"
	podutils "github.com/bwagner5/kube-demo/pkg/utils/resources"
)

type observerFunc func()

// Cluster maintains cluster state that is often needed but expensive to compute.
type Cluster struct {
	kubeClient client.Client
	clock      clock.Clock

	// Node Status & Pod -> Node Binding
	mu          sync.RWMutex
	nodes       map[string]*Node                // node name -> node
	bindings    map[types.NamespacedName]string // pod namespaced named -> node name
	unboundPods map[types.NamespacedName]*v1.Pod

	updateObservers []observerFunc

	// consolidationState is a number indicating the state of the cluster with respect to consolidation.  If this number
	// hasn't changed, it indicates that the cluster hasn't changed in a state which would enable consolidation if
	// it previously couldn't occur.
	lastNodeDeletionTime int64
	lastNodeCreationTime int64
}

func NewCluster(clk clock.Clock, client client.Client) *Cluster {
	c := &Cluster{
		clock:       clk,
		kubeClient:  client,
		nodes:       map[string]*Node{},
		bindings:    map[types.NamespacedName]string{},
		unboundPods: map[types.NamespacedName]*v1.Pod{},
	}
	return c
}

// Node is a cached version of a node in the cluster that maintains state which is expensive to compute every time it's
// needed.  This currently contains node utilization across all the allocatable resources, but will soon be used to
// compute topology information.
// +k8s:deepcopy-gen=true
type Node struct {
	Node *v1.Node
	Pods map[types.NamespacedName]*v1.Pod
	// Capacity is the total resources on the node.
	Capacity v1.ResourceList
	// Allocatable is the total amount of resources on the node after os overhead.
	Allocatable v1.ResourceList
	// Available is allocatable minus anything allocated to pods.
	Available v1.ResourceList
	// Available is the total amount of resources that are available on the node.  This is the Allocatable minus the
	// resources requested by all pods bound to the node.
	// DaemonSetRequested is the total amount of resources that have been requested by daemon sets.  This allows users
	// of the Node to identify the remaining resources that we expect future daemonsets to consume.  This is already
	// included in the calculation for Available.
	DaemonSetRequested v1.ResourceList
	DaemonSetLimits    v1.ResourceList

	podRequests map[types.NamespacedName]v1.ResourceList
	podLimits   map[types.NamespacedName]v1.ResourceList

	// PodTotalRequests is the total resources on pods scheduled to this node
	PodTotalRequests v1.ResourceList
	// PodTotalLimits is the total resource limits scheduled to this node
	PodTotalLimits v1.ResourceList
}

// ForEachNode calls the supplied function once per node object that is being tracked. It is not safe to store the
// state.Node object, it should be only accessed from within the function provided to this method.
func (c *Cluster) ForEachNode(f func(n *Node) bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	var nodes []*Node
	for _, node := range c.nodes {
		nodes = append(nodes, node)
	}
	// sort nodes by creation time so we provide a consistent ordering
	sort.Slice(nodes, func(a, b int) bool {
		if nodes[a].Node.CreationTimestamp != nodes[b].Node.CreationTimestamp {
			return nodes[a].Node.CreationTimestamp.Time.Before(nodes[b].Node.CreationTimestamp.Time)
		}
		// sometimes we get nodes created in the same second, so sort again by node UID to provide a consistent ordering
		return nodes[a].Node.UID < nodes[b].Node.UID
	})

	for _, node := range nodes {
		if !f(node) {
			return
		}
	}
}

func (c *Cluster) ForEachUnboundPod(f func(*v1.Pod) bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	var pods []*v1.Pod
	for _, pod := range c.unboundPods {
		pods = append(pods, pod)
	}
	// sort nodes by creation time so we provide a consistent ordering
	sort.Slice(pods, func(a, b int) bool {
		if pods[a].CreationTimestamp != pods[b].CreationTimestamp {
			return pods[a].CreationTimestamp.Time.Before(pods[b].CreationTimestamp.Time)
		}
		// sometimes we get nodes created in the same second, so sort again by node UID to provide a consistent ordering
		return pods[a].UID < pods[b].UID
	})

	for _, pod := range pods {
		if !f(pod) {
			return
		}
	}
}

func (c *Cluster) AddOnChangeObserver(f observerFunc) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.updateObservers = append(c.updateObservers, f)
}

func (c *Cluster) notifyObservers() {
	wg := &sync.WaitGroup{}
	for _, observer := range c.updateObservers {
		wg.Add(1)
		go func(f observerFunc) {
			defer wg.Done()
			f()
		}(observer)
	}
	wg.Wait()
}

// newNode always returns a node, even if some portion of the update has failed
func (c *Cluster) newNode(ctx context.Context, node *v1.Node) (*Node, error) {
	n := &Node{
		Node:        node,
		Pods:        map[types.NamespacedName]*v1.Pod{},
		Capacity:    node.Status.Capacity,
		Allocatable: node.Status.Allocatable,
		Available:   v1.ResourceList{},
		podRequests: map[types.NamespacedName]v1.ResourceList{},
		podLimits:   map[types.NamespacedName]v1.ResourceList{},
	}
	if err := multierr.Combine(
		c.populateVolumeLimits(ctx, node, n),
		c.populateResourceRequests(ctx, node, n),
	); err != nil {
		return nil, err
	}
	return n, nil
}

func (c *Cluster) populateResourceRequests(ctx context.Context, node *v1.Node, n *Node) error {
	var pods v1.PodList
	if err := c.kubeClient.List(ctx, &pods, client.MatchingFields{"spec.nodeName": node.Name}); err != nil {
		return fmt.Errorf("listing pods, %w", err)
	}
	var requested []v1.ResourceList
	var limits []v1.ResourceList
	var daemonsetRequested []v1.ResourceList
	var daemonsetLimits []v1.ResourceList
	for i := range pods.Items {
		pod := &pods.Items[i]
		if pod2.IsTerminal(pod) {
			continue
		}
		requests := podutils.RequestsForPods(pod)
		podLimits := podutils.LimitsForPods(pod)
		podKey := client.ObjectKeyFromObject(pod)
		n.podRequests[podKey] = requests
		n.podLimits[podKey] = podLimits
		n.Pods[podKey] = pod
		c.bindings[podKey] = n.Node.Name
		if pod2.IsOwnedByDaemonSet(pod) {
			daemonsetRequested = append(daemonsetRequested, requests)
			daemonsetLimits = append(daemonsetLimits, podLimits)
		}
		requested = append(requested, requests)
		limits = append(limits, podLimits)
	}

	n.DaemonSetRequested = podutils.Merge(daemonsetRequested...)
	n.DaemonSetLimits = podutils.Merge(daemonsetLimits...)
	n.PodTotalRequests = podutils.Merge(requested...)
	n.PodTotalLimits = podutils.Merge(limits...)
	n.Available = podutils.Subtract(n.Allocatable, podutils.Merge(requested...))
	return nil
}

func (c *Cluster) populateVolumeLimits(ctx context.Context, node *v1.Node, _ *Node) error {
	var csiNode storagev1.CSINode
	if err := c.kubeClient.Get(ctx, client.ObjectKey{Name: node.Name}, &csiNode); err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("getting CSINode to determine volume limit for %s, %w", node.Name, err)
	}
	return nil
}

func (c *Cluster) deleteNode(nodeName string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.nodes, nodeName)
	c.notifyObservers()
}

// updateNode is called for every node reconciliation
func (c *Cluster) updateNode(ctx context.Context, node *v1.Node) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	n, err := c.newNode(ctx, node)
	if err != nil {
		// ensure that the out of date node is forgotten
		delete(c.nodes, node.Name)
		return err
	}

	c.nodes[node.Name] = n
	if node.DeletionTimestamp != nil {
		nodeDeletionTime := node.DeletionTimestamp.UnixMilli()
		if nodeDeletionTime > atomic.LoadInt64(&c.lastNodeDeletionTime) {
			atomic.StoreInt64(&c.lastNodeDeletionTime, nodeDeletionTime)
		}
	}
	nodeCreationTime := node.CreationTimestamp.UnixMilli()
	if nodeCreationTime > atomic.LoadInt64(&c.lastNodeCreationTime) {
		atomic.StoreInt64(&c.lastNodeCreationTime, nodeCreationTime)
	}
	c.notifyObservers()
	return nil
}

// deletePod is called when the pod has been deleted
func (c *Cluster) deletePod(podKey types.NamespacedName) {
	c.updateNodeUsageFromPodCompletion(podKey)
	c.notifyObservers()
}

func (c *Cluster) updateNodeUsageFromPodCompletion(podKey types.NamespacedName) {
	c.mu.Lock()
	defer c.mu.Unlock()

	nodeName, bindingKnown := c.bindings[podKey]
	if !bindingKnown {
		// we didn't think the pod was bound, so we weren't tracking it and don't need to do anything
		return
	}

	delete(c.bindings, podKey)
	n, ok := c.nodes[nodeName]
	if !ok {
		// we weren't tracking the node yet, so nothing to do
		return
	}
	// pod has been deleted so our available capacity increases by the resources that had been
	// requested by the pod
	n.Available = podutils.Merge(n.Available, n.podRequests[podKey])
	n.PodTotalRequests = podutils.Subtract(n.PodTotalRequests, n.podRequests[podKey])
	n.PodTotalLimits = podutils.Subtract(n.PodTotalLimits, n.podLimits[podKey])
	delete(n.podRequests, podKey)
	delete(n.podLimits, podKey)
	delete(n.Pods, podKey)

	// We can't easily track the changes to the DaemonsetRequested here as we no longer have the pod.  We could keep up
	// with this separately, but if a daemonset pod is being deleted, it usually means the node is going down.  In the
	// worst case we will resync to correct this.
}

// updatePod is called every time the pod is reconciled
func (c *Cluster) updatePod(ctx context.Context, pod *v1.Pod) error {
	var err error
	if pod2.IsTerminal(pod) {
		c.updateNodeUsageFromPodCompletion(client.ObjectKeyFromObject(pod))
	} else {
		err = c.updateNodeUsageFromPod(ctx, pod)
	}
	c.notifyObservers()
	return err
}

// updateNodeUsageFromPod is called every time a reconcile event occurs for the pod. If the pods binding has changed
// (unbound to bound), we need to update the resource requests on the node.
func (c *Cluster) updateNodeUsageFromPod(ctx context.Context, pod *v1.Pod) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	podKey := client.ObjectKeyFromObject(pod)
	// nothing to do if the pod isn't bound, checking early allows avoiding unnecessary locking
	if pod.Spec.NodeName == "" {
		c.unboundPods[podKey] = pod
		return nil
	}
	delete(c.unboundPods, podKey)

	oldNodeName, bindingKnown := c.bindings[podKey]
	if bindingKnown {
		if oldNodeName == pod.Spec.NodeName {
			// we are already tracking the pod binding, so nothing to update
			if err := c.ensureNodeCreated(ctx, oldNodeName); err != nil {
				return err
			}
			c.nodes[oldNodeName].Pods[podKey] = pod
			return nil
		}
		// the pod has switched nodes, this can occur if a pod name was re-used and it was deleted/re-created rapidly,
		// binding to a different node the second time
		n, ok := c.nodes[oldNodeName]
		if ok {
			// we were tracking the old node, so we need to reduce its capacity by the amount of the pod that has
			// left it
			delete(c.bindings, podKey)
			delete(n.Pods, podKey)
			n.Available = podutils.Merge(n.Available, n.podRequests[podKey])
			n.PodTotalRequests = podutils.Subtract(n.PodTotalRequests, n.podRequests[podKey])
			n.PodTotalLimits = podutils.Subtract(n.PodTotalLimits, n.podLimits[podKey])
			delete(n.podRequests, podKey)
			delete(n.podLimits, podKey)
		}
	}

	if err := c.ensureNodeCreated(ctx, pod.Spec.NodeName); err != nil {
		return err
	}
	n := c.nodes[pod.Spec.NodeName]

	// sum the newly bound pod's requests and limits into the existing node and record the binding
	podRequests := podutils.RequestsForPods(pod)
	podLimits := podutils.LimitsForPods(pod)
	// our available capacity goes down by the amount that the pod had requested
	n.Available = podutils.Subtract(n.Available, podRequests)
	n.PodTotalRequests = podutils.Merge(n.PodTotalRequests, podRequests)
	n.PodTotalLimits = podutils.Merge(n.PodTotalLimits, podLimits)
	// if it's a daemonset, we track what it has requested separately
	if pod2.IsOwnedByDaemonSet(pod) {
		n.DaemonSetRequested = podutils.Merge(n.DaemonSetRequested, podRequests)
		n.DaemonSetLimits = podutils.Merge(n.DaemonSetRequested, podLimits)
	}
	n.podRequests[podKey] = podRequests
	n.podLimits[podKey] = podLimits
	n.Pods[podKey] = pod
	c.bindings[podKey] = n.Node.Name
	return nil
}

func (c *Cluster) ensureNodeCreated(ctx context.Context, nodeName string) error {
	_, ok := c.nodes[nodeName]
	if !ok {
		var node v1.Node
		if err := c.kubeClient.Get(ctx, client.ObjectKey{Name: nodeName}, &node); err != nil && !errors.IsNotFound(err) {
			return fmt.Errorf("getting node, %w", err)
		}

		var err error
		// node didn't exist, but creating it will pick up this newly bound pod as well
		n, err := c.newNode(ctx, &node)
		if err != nil {
			// no need to delete c.nodes[node.Name] as it wasn't stored previously
			return err
		}
		c.nodes[node.Name] = n
	}
	return nil
}
