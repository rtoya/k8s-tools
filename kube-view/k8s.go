package main

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

// K8sClient wraps the Kubernetes clientset
type K8sClient struct {
	clientset *kubernetes.Clientset
}

// NodeInfo holds information about a Kubernetes node
type NodeInfo struct {
	Name         string
	Status       string
	CPUUsage     string
	MemUsage     string
	CPUPercent   float64 // CPU usage as percentage (0-100)
	MemPercent   float64 // Memory usage as percentage (0-100)
	Pods         []PodInfo
	Ready        bool
}

// PodInfo holds information about a Kubernetes pod
type PodInfo struct {
	Name      string
	Namespace string
	Status    string
	Ready     bool
	Node      string
}

// NewK8sClient creates a new Kubernetes client
func NewK8sClient() (*K8sClient, error) {
	var config *rest.Config
	var err error

	// Try in-cluster config first
	config, err = rest.InClusterConfig()
	if err != nil {
		// Fall back to kubeconfig
		kubeconfig := filepath.Join(homedir.HomeDir(), ".kube", "config")
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			return nil, fmt.Errorf("failed to load kubeconfig: %w", err)
		}
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create clientset: %w", err)
	}

	return &K8sClient{clientset: clientset}, nil
}

// GetClusterData fetches nodes and pods from the cluster
func (k *K8sClient) GetClusterData(ctx context.Context) ([]NodeInfo, error) {
	// Get all nodes
	nodes, err := k.clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list nodes: %w", err)
	}

	// Get all pods
	pods, err := k.clientset.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list pods: %w", err)
	}

	// Build node info map
	nodeInfoMap := make(map[string]*NodeInfo)
	for _, node := range nodes.Items {
		ready := isNodeReady(&node)
		status := "NotReady"
		if ready {
			status = "Ready"
		}

		cpuUsage, cpuPercent := getNodeResourceUsageWithPercent(&node, corev1.ResourceCPU)
		memUsage, memPercent := getNodeResourceUsageWithPercent(&node, corev1.ResourceMemory)

		nodeInfoMap[node.Name] = &NodeInfo{
			Name:       node.Name,
			Status:     status,
			CPUUsage:   cpuUsage,
			MemUsage:   memUsage,
			CPUPercent: cpuPercent,
			MemPercent: memPercent,
			Pods:       []PodInfo{},
			Ready:      ready,
		}
	}

	// Associate pods with nodes
	for _, pod := range pods.Items {
		podInfo := PodInfo{
			Name:      pod.Name,
			Namespace: pod.Namespace,
			Status:    string(pod.Status.Phase),
			Ready:     isPodReady(&pod),
			Node:      pod.Spec.NodeName,
		}

		if nodeInfo, exists := nodeInfoMap[pod.Spec.NodeName]; exists {
			nodeInfo.Pods = append(nodeInfo.Pods, podInfo)
		}
	}

	// Sort pods within each node by namespace and name for consistent order
	for _, info := range nodeInfoMap {
		sort.Slice(info.Pods, func(i, j int) bool {
			if info.Pods[i].Namespace != info.Pods[j].Namespace {
				return info.Pods[i].Namespace < info.Pods[j].Namespace
			}
			return info.Pods[i].Name < info.Pods[j].Name
		})
	}

	// Convert map to slice and sort by node name to maintain consistent order
	var nodeNames []string
	for name := range nodeInfoMap {
		nodeNames = append(nodeNames, name)
	}
	sort.Strings(nodeNames)

	var nodeInfos []NodeInfo
	for _, name := range nodeNames {
		nodeInfos = append(nodeInfos, *nodeInfoMap[name])
	}

	return nodeInfos, nil
}

// isNodeReady checks if a node is ready
func isNodeReady(node *corev1.Node) bool {
	for _, condition := range node.Status.Conditions {
		if condition.Type == corev1.NodeReady {
			return condition.Status == corev1.ConditionTrue
		}
	}
	return false
}

// isPodReady checks if a pod is ready
func isPodReady(pod *corev1.Pod) bool {
	for _, condition := range pod.Status.Conditions {
		if condition.Type == corev1.PodReady {
			return condition.Status == corev1.ConditionTrue
		}
	}
	return false
}

// getNodeResourceUsage returns a formatted string of resource usage
func getNodeResourceUsage(node *corev1.Node, resourceName corev1.ResourceName) string {
	capacity := node.Status.Capacity[resourceName]
	allocatable := node.Status.Allocatable[resourceName]

	if resourceName == corev1.ResourceCPU {
		return fmt.Sprintf("%s/%s", allocatable.String(), capacity.String())
	}
	return fmt.Sprintf("%s/%s", allocatable.String(), capacity.String())
}

// getNodeResourceUsageWithPercent returns usage string and percentage
func getNodeResourceUsageWithPercent(node *corev1.Node, resourceName corev1.ResourceName) (string, float64) {
	capacity := node.Status.Capacity[resourceName]
	allocatable := node.Status.Allocatable[resourceName]

	usageStr := fmt.Sprintf("%s/%s", allocatable.String(), capacity.String())

	// Calculate percentage based on capacity
	// Note: This is allocatable/capacity ratio, not actual usage
	// For actual usage, you'd need metrics-server
	capacityValue := capacity.AsApproximateFloat64()
	allocatableValue := allocatable.AsApproximateFloat64()

	var percent float64
	if capacityValue > 0 {
		// Using a simulated usage: assume 60% of allocatable is used (for demo)
		// In production, fetch from metrics-server
		percent = (allocatableValue * 0.6 / capacityValue) * 100
	}

	return usageStr, percent
}
