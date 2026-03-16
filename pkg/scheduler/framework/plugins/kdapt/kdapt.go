package kdapt

import (
	"context"
	"sync"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	fwk "k8s.io/kube-scheduler/framework"
	metricsclient "k8s.io/metrics/pkg/client/clientset/versioned"
	"time"
)

const Name = "KdaptScheduler"

type NodeRuntimeMetrics struct {
	CPUMilli    int64
	MemoryBytes int64
}

type Kdapt struct {
	handle fwk.Handle
	// map is not thread safe, ever for multiple read and single write goroutines.
	nodeMetrics map[string]NodeRuntimeMetrics
	// for thread safety
	mutexLock sync.RWMutex
}

var _ fwk.ScorePlugin = &Kdapt{}

func (k *Kdapt) Name() string {
	return Name
}

func New(
	ctx context.Context,
	obj runtime.Object,
	handle fwk.Handle,
) (fwk.Plugin, error) {
	k := &Kdapt{
		handle:      handle,
		nodeMetrics: make(map[string]NodeRuntimeMetrics),
	}
	metricsClient, err := metricsclient.NewForConfig(handle.KubeConfig())
	if err != nil {
		return nil, err
	}

	go k.runMetricsCollector(ctx, metricsClient, 10*time.Second)

	return k, nil
}

func (k *Kdapt) runMetricsCollector(
	ctx context.Context,
	metricsClient *metricsclient.Clientset,
	interval time.Duration,
) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			next, err := k.collectMetrics(ctx, metricsClient)
			if err != nil {
				continue
			}
			k.mutexLock.Lock()
			k.nodeMetrics = next
			k.mutexLock.Unlock()
		}
	}
}

func (k *Kdapt) collectMetrics(
	ctx context.Context,
	metricsClient *metricsclient.Clientset,
) (map[string]NodeRuntimeMetrics, error) {
	list, err := metricsClient.MetricsV1beta1().NodeMetricses().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	next := make(map[string]NodeRuntimeMetrics, len(list.Items))
	for _, node := range list.Items {
		next[node.Name] = NodeRuntimeMetrics{
			CPUMilli:    node.Usage.Cpu().MilliValue(),
			MemoryBytes: node.Usage.Memory().Value(),
		}
	}
	return next, nil
}

func (k *Kdapt) Score(
	ctx context.Context,
	state fwk.CycleState,
	pod *v1.Pod,
	nodeInfo fwk.NodeInfo,
) (int64, *fwk.Status) {
	// A simple bin pack scoring plugin that scores nodes based on their resource utilization
	allocatable := nodeInfo.GetAllocatable()
	used := nodeInfo.GetRequested()

	cpuUtilization := float64(used.GetMilliCPU()) / float64(allocatable.GetMilliCPU())
	memUtilization := float64(used.GetMemory()) / float64(allocatable.GetMemory())

	// Simple scoring function that combines CPU and memory utilization, giving more weight to CPU
	score := cpuUtilization*0.7 + memUtilization*0.3

	return int64(score * float64(fwk.MaxNodeScore)), fwk.NewStatus(fwk.Success)
}

func (k *Kdapt) NormalizeScore(
	ctx context.Context,
	state fwk.CycleState,
	pod *v1.Pod,
	scores fwk.NodeScoreList,
) *fwk.Status {
	var maxScore int64 = 0
	for i := range scores {
		if scores[i].Score > maxScore {
			maxScore = scores[i].Score
		}
	}

	for i := range scores {
		if maxScore > 0 {
			scores[i].Score = scores[i].Score * fwk.MaxNodeScore / maxScore
		}
	}
	return fwk.NewStatus(fwk.Success)
}

func (k *Kdapt) ScoreExtensions() fwk.ScoreExtensions {
	return k
}
