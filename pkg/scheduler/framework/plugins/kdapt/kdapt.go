package kdapt

import (
	"context"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	fwk "k8s.io/kube-scheduler/framework"
)

const Name = "KdaptScheduler"

type Kdapt struct {
	handle fwk.Handle
}

var _ fwk.ScorePlugin = &Kdapt{}

func (k *Kdapt) Name() string {
	return Name
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
	score := int64((cpuUtilization * 0.7) + (memUtilization*0.3)*float64(fwk.MaxNodeScore))

	return score, fwk.NewStatus(fwk.Success)
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

func New(
	ctx context.Context,
	obj runtime.Object,
	handle fwk.Handle,
) (fwk.Plugin, error) {
	return &Kdapt{
		handle: handle,
	}, nil
}
