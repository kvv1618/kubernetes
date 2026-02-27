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

	// A simple controlled static load test scoring function that assigns a score of 100 to the node named "scheduler-lab-worker" and a score of 60 to all other nodes.
	if nodeInfo.Node().Name == "scheduler-lab-worker" {
		return fwk.MaxNodeScore, fwk.NewStatus(fwk.Success)
	}
	return 60, fwk.NewStatus(fwk.Success)
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
