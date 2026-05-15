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
			"nodeUsageMetrics {name=%s, cpuMilli=%.2f, memoryBytes=%.2f, smoothedCpuMilli=%.2f, smoothedMemoryBytes=%.2f}",
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
	ema := 0.4 // Exponential Moving Average, considering 40% new runtime values and 60% old/hostoric values.
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
	const epsilonUtil = 0.0001
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

	runTimeCpuUtil := clamp(rt.CPUMilli/allocCpu, 0, 1)
	runTimeMemUtil := clamp(rt.MemoryBytes/allocMem, 0, 1)
	smoothedCpuUtil := clamp(rt.SmoothedCPUMilli/allocCpu, 0, 1)
	smoothedMemUtil := clamp(rt.SmoothedMemoryBytes/allocMem, 0, 1)

	// beta — how much has reality diverged from the trend
	// When raw ≈ smoothed: EMA is tracking reality well → trust smoothed (beta ≈ 0)
	// When raw >> smoothed: spike in progress, EMA hasn't caught up → trust raw (beta → 1)
	// When raw << smoothed: recovery after termination, EMA still high → trust raw (beta → 1)
	betaCpu := clamp(
		math.Abs(runTimeCpuUtil-smoothedCpuUtil)/(math.Max(runTimeCpuUtil, smoothedCpuUtil)+epsilonUtil),
		0, 1,
	)

	betaMem := clamp(
		math.Abs(runTimeMemUtil-smoothedMemUtil)/(math.Max(runTimeMemUtil, smoothedMemUtil)+epsilonUtil),
		0, 1,
	)

	// beta=0 (stable):    effective = smoothed        (EMA is tracking fine, use its stability)
	// beta=1 (diverging): effective = raw             (EMA is stale, use real-time signal)
	effectiveCpuUtil := (1-betaCpu)*smoothedCpuUtil + betaCpu*runTimeCpuUtil
	effectiveMemUtil := (1-betaMem)*smoothedMemUtil + betaMem*runTimeMemUtil

	cpuMismatch := clamp(math.Abs(requestedCpuUtil-effectiveCpuUtil), 0, 1)
	memMismatch := clamp(math.Abs(requestedMemUtil-effectiveMemUtil), 0, 1)

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
	cpuScore := (1-alphaCpu)*projectedCpuUtil + alphaCpu*effectiveCpuUtil
	memScore := (1-alphaMem)*projectedMemUtil + alphaMem*effectiveMemUtil
	// Convex combination of projected and effective utilization keeps
	// the score between 0 and 1, while allowing it to reflect both the current state and the projected impact of placing the pod on the node.

	wCpu := 0.6
	wMem := 0.4
	// Design choise to keep wCpu + wMem = 1, so that all scores doesn't exceed 1, which is improtant
	// for this algorithm to work as expected.

	cpuPenalityFactor := 0.0
	memPenalityFactor := 0.0
	if projectedCpuUtil > 0.8 {
		cpuPenalityFactor = (projectedCpuUtil - 0.8) / 0.2 // Linear penality from 0 to 1 as projectedCpuUtil goes from 0.8 to 1.0
	}
	if projectedMemUtil > 0.75 && projectedMemUtil < 0.9 {
		// Memory is more of a hard constraint, so we start applying penality earlier at 70% projected utilization.
		memPenalityFactor = (projectedMemUtil - 0.75) / 0.15 // Linear penality from 0 to 1 as projectedMemUtil goes from 0.75 to 0.9
	}

	finalScore := 0.0
	if projectedMemUtil >= 0.9 { //Critical memory threshold
		memPenality := (projectedMemUtil - 0.9) / 0.1 // Linear penality from 0 to 1 as projectedMemUtil goes from 0.9 to 1.0
		finalScore = 0.5 * clamp(1.0-memPenality, 0, 1)
		//Memory safe bin-packing, which never  exceeds 0.5 score if memory is projected to be above 90% utilization, regardless of CPU score.
	} else {
		penalisedScore := wCpu*cpuScore*(1-cpuPenalityFactor) + wMem*memScore*(1-memPenalityFactor)
		finalScore = 0.5 + 0.5*clamp(penalisedScore, 0, 1)
	}

	klog.Infof(
		"pod=%s, node=%s, requestedCpuUtil=%.2f, requestedMemUtil=%.2f,"+
			"runTimeCpuUtil=%.2f, runTimeMemUtil=%.2f, smoothedCpuUtil=%.2f,"+
			"smoothedMemUtil=%.2f, betaCpu=%.2f, betaMem=%.2f, effectiveCpuUtil=%.2f,"+
			"effectiveMemUtil=%.2f, cpuMismatch=%.2f, memMismatch=%.2f,"+
			"projectedCpuUtil=%.2f, projectedMemUtil=%.2f, alphaCpu=%.2f,"+
			"alphaMem=%.2f, cpuScore=%.2f, memScore=%.2f, finalScore=%.4f,"+
			"cpuPenalityFactor=%.2f, memPenalityFactor=%.2f",
		pod.Name,
		nodeInfo.Node().Name,
		requestedCpuUtil,
		requestedMemUtil,
		runTimeCpuUtil,
		runTimeMemUtil,
		smoothedCpuUtil,
		smoothedMemUtil,
		betaCpu,
		betaMem,
		effectiveCpuUtil,
		effectiveMemUtil,
		cpuMismatch,
		memMismatch,
		projectedCpuUtil,
		projectedMemUtil,
		alphaCpu,
		alphaMem,
		cpuScore,
		memScore,
		finalScore,
		cpuPenalityFactor,
		memPenalityFactor,
	)

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
