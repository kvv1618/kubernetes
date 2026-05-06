package kdapt

import (
	"context"
	"k8s.io/klog/v2"
	"maps"
	"math"
	"sort"
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
	CPUMilli    float64
	MemoryBytes float64

	SmoothedCPUMilli    float64
	SmoothedMemoryBytes float64
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

func (k *Kdapt) logMetrics() {
	nodeNames := make([]string, 0, len(k.nodeMetrics))
	for nodeName := range k.nodeMetrics {
		nodeNames = append(nodeNames, nodeName)
	}
	sort.Strings(nodeNames)

	for _, nodeName := range nodeNames {
		metrics := k.nodeMetrics[nodeName]
		klog.Infof(
			"nodeMetrics {name=%s, cpuMilli=%.2f, memoryBytes=%.2f, smoothedCpuMilli=%.2f, smoothedMemoryBytes=%.2f}",
			nodeName,
			metrics.CPUMilli,
			metrics.MemoryBytes,
			metrics.SmoothedCPUMilli,
			metrics.SmoothedMemoryBytes,
		)
	}
	klog.Infof("-----------------------------------------------------------------")
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
				klog.Errorf("error collecting metrics: %v", err)
				continue
			}
			k.mutexLock.Lock()
			k.nodeMetrics = next
			k.mutexLock.Unlock()
		}
		k.logMetrics()
	}
}

func clamp(x, min, max float64) float64 {
	if x < min {
		return min
	}
	if x > max {
		return max
	}
	return x
}

func (k *Kdapt) collectMetrics(
	ctx context.Context,
	metricsClient *metricsclient.Clientset,
) (map[string]NodeRuntimeMetrics, error) {
	ema := 0.3 // Exponential Moving Average, considering 30% new runtime values and 70% old/hostoric values.
	list, err := metricsClient.MetricsV1beta1().NodeMetricses().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	k.mutexLock.RLock()
	prevMetrics := make(map[string]NodeRuntimeMetrics, len(k.nodeMetrics))
	maps.Copy(prevMetrics, k.nodeMetrics)
	k.mutexLock.RUnlock()

	next := make(map[string]NodeRuntimeMetrics, len(list.Items))
	for _, node := range list.Items {
		currCpu := float64(node.Usage.Cpu().MilliValue())
		currMem := float64(node.Usage.Memory().Value())
		prevNodeMetrics, exists := prevMetrics[node.Name]
		if !exists {
			prevNodeMetrics = NodeRuntimeMetrics{
				SmoothedCPUMilli:    currCpu,
				SmoothedMemoryBytes: currMem,
			}
		}
		SmoothedCPUMilli := ema*currCpu + (1-ema)*prevNodeMetrics.SmoothedCPUMilli
		SmoothedMemoryBytes := ema*currMem + (1-ema)*prevNodeMetrics.SmoothedMemoryBytes
		next[node.Name] = NodeRuntimeMetrics{
			CPUMilli:            currCpu,
			MemoryBytes:         currMem,
			SmoothedCPUMilli:    SmoothedCPUMilli,
			SmoothedMemoryBytes: SmoothedMemoryBytes,
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
	allocatable := nodeInfo.GetAllocatable()
	requested := nodeInfo.GetRequested()

	allocCpu := float64(allocatable.GetMilliCPU())
	allocMem := float64(allocatable.GetMemory())

	requestedCpuOnNode := float64(requested.GetMilliCPU())
	requestedMemOnNode := float64(requested.GetMemory())

	requestedCpuUtil := clamp(requestedCpuOnNode/allocCpu, 0, 1)
	requestedMemUtil := clamp(requestedMemOnNode/allocMem, 0, 1)

	k.mutexLock.RLock()
	rt, ok := k.nodeMetrics[nodeInfo.Node().Name]
	k.mutexLock.RUnlock()
	if !ok {
		klog.Errorf("node metrics not found for node: %s", nodeInfo.Node().Name)
		binPackScore := requestedCpuUtil*0.7 + requestedMemUtil*0.3
		klog.Infof("binPackScore: %v", binPackScore)
		return int64(binPackScore * float64(fwk.MaxNodeScore)), fwk.NewStatus(fwk.Success)
	}

	runTimeCpuUtil := clamp(rt.SmoothedCPUMilli/allocCpu, 0, 1)
	runTimeMemUtil := clamp(rt.SmoothedMemoryBytes/allocMem, 0, 1)

	cpuMismatch := clamp(math.Abs(requestedCpuUtil-runTimeCpuUtil), 0, 1)
	memMismatch := clamp(math.Abs(requestedMemUtil-runTimeMemUtil), 0, 1)

	requestedCpuByPod, requestedMemByPod := int64(0), int64(0)
	for _, container := range pod.Spec.Containers {
		requestedCpuByPod += container.Resources.Requests.Cpu().MilliValue()
		requestedMemByPod += container.Resources.Requests.Memory().Value()
	}
	projectedCpu := requestedCpuOnNode + float64(requestedCpuByPod)
	projectedMem := requestedMemOnNode + float64(requestedMemByPod)
	projectedCpuUtil := clamp(projectedCpu/allocCpu, 0, 1)
	projectedMemUtil := clamp(projectedMem/allocMem, 0, 1)

	// Missmatch is used to calculate the weight of the runtime utilization.
	// aplha is the weight of the runtime metrics.
	alphaCpu := clamp(0.3+cpuMismatch*0.8, 0, 1) // 0.3 is the base weight, when there is no mismatch.
	alphaMem := clamp(0.1+memMismatch*0.4, 0, 1)
	// Score using projectedUtil vs runtimeUtil - estimate placement quality
	cpuScore := (1-alphaCpu)*projectedCpuUtil + alphaCpu*runTimeCpuUtil
	memScore := (1-alphaMem)*projectedMemUtil + alphaMem*runTimeMemUtil

	wCpu := 0.6
	wMem := 0.4
	penality := 0.0
	if runTimeCpuUtil > 0.8 {
		penality += 0.2 // less penalty for higher cpu utilization, because CPU is compressible
	}
	if runTimeMemUtil > 0.8 {
		penality += 0.25
	}

	finalScore := wCpu*cpuScore + wMem*memScore - penality

	klog.Infof(
		"pod=%s node=%s reqCPU=%.2f rtCPU=%.2f mismatchCPU=%.2f projectedCPU=%.2f final=%.2f",
		pod.Name,
		nodeInfo.Node().Name,
		requestedCpuUtil,
		runTimeCpuUtil,
		cpuMismatch,
		projectedCpuUtil,
		finalScore,
	)
	klog.Infof(
		"pod=%s node=%s reqMem=%.2f rtMem=%.2f mismatchMem=%.2f projectedMem=%.2f final=%.2f",
		pod.Name,
		nodeInfo.Node().Name,
		requestedMemUtil,
		runTimeMemUtil,
		memMismatch,
		projectedMemUtil,
		finalScore,
	)
	klog.Infof("Final score: %v", finalScore)
	return int64(finalScore * float64(fwk.MaxNodeScore)), fwk.NewStatus(fwk.Success)

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
