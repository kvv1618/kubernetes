## Hands-on progress till now:
- Setup KIND cluster for local Kubernetes development and testing.
- Forked Kubernetes repo and set up local development environment.
- Implemented basic Kdapt Scheduler plugin with scoring functionality. (returning static scores for testing)
- Genralised the plugin as per the Kubernetes scheduler plugin framework, and test it with static pod resource requests, using nodeAffinity to verify that the plugin is being invoked correctly during scheduling.
- Written a simple bin pack scoring algo based on Node resource utilization, combining CPU and memory utilization, giving more weight to CPU.
- Deployed pods with fixed pre-determined node requests with nodeAffinity to test the bin pack scoring algo. Static pods were deployed as per the expected bin packing strategy.
- Implemented an adaptive runtime-aware bin packing algorithm that uses smoothed runtime metrics from the metrics server, and dynamically adjusts the weight given to runtime metrics based on the mismatch between request-based and runtime-based utilization. The algorithm also uses projected utilization to make future-aware scheduling decisions, and applies penalties to nodes that are already under high runtime pressure.
- Deployed a burst of pods with fixed resource requests to test the adaptive runtime-aware bin packing algorithm. The results showed that the scheduler preferred packing on the most utilized node while avoiding overloading it, and then moved to the second best node, which is the expected behavior. When the burst was memory heavy, the scheduler was more conservative in packing on the most utilized node and preferred to spread the pods more evenly across nodes, which is also the expected behavior given that memory pressure is more dangerous than CPU pressure.
- Identified a problem with the use of smoothed runtime values in calculating resource utilization mismatch, which can lead to suboptimal scheduling decisions during fast bursts of pods or when there are sudden changes in node behavior. Handled this by introducing a new parameter `beta` to control the effectiveResourceUtilization used in the calculation of mismatch, and by adjusting the algorithm to bridge the gap between raw and smoothed metrics. This allows the scheduler to adapt more quickly to changes in node behavior while still benefiting from the stability of smoothed metrics.

### Testing methodology:
#### Static Pod testing:
##### NodeAffinity:
- Deployed a static pod deployment using kdapt scheduler.
- The plugin at this point of time had a bias towards a specific node ("scheduler-lab-worker") by returning a score of 100 for that node and 60 for all other nodes.

    ```go
    func (k *Kdapt) Score(
        ctx context.Context,
        state fwk.CycleState,
        pod *v1.Pod,
        nodeInfo fwk.NodeInfo,
    ) (int64, *fwk.Status) {

        // A simple controlled static load test scoring function
        if nodeInfo.Node().Name == "scheduler-lab-worker" {
            return fwk.MaxNodeScore, fwk.NewStatus(fwk.Success)
        }
        return 60, fwk.NewStatus(fwk.Success)
    }
    ```
- Observed that all pods were scheduled on the "scheduler-lab-worker" node, confirming that the plugin was being invoked and influencing scheduling decisions as expected.

  ```yaml
  apiVersion: apps/v1
  kind: Deployment
  metadata:
    name: static-pod-deployment
  spec:
    replicas: 10
    selector:
      matchLabels:
        app: static-pod
    template:
      metadata:
        labels:
          app: static-pod
      spec:
        schedulerName: kdapt-scheduler
        containers:
          - name: static-pod-container
            image: nginx:latest
            ports:
              - containerPort: 80
  ```
		
	```bash
		k8 get pods -o wide                                     
		NAME                                     READY   STATUS    RESTARTS   AGE   IP            NODE                   NOMINATED NODE   READINESS GATES
		static-pod-deployment-8659b49b9b-2gppv   1/1     Running   0          10m   10.244.2.19   scheduler-lab-worker   <none>           <none>
		static-pod-deployment-8659b49b9b-49xnd   1/1     Running   0          10m   10.244.2.13   scheduler-lab-worker   <none>           <none>
		static-pod-deployment-8659b49b9b-4wsts   1/1     Running   0          10m   10.244.2.17   scheduler-lab-worker   <none>           <none>
		static-pod-deployment-8659b49b9b-7xtgg   1/1     Running   0          10m   10.244.2.10   scheduler-lab-worker   <none>           <none>
		static-pod-deployment-8659b49b9b-mv9hv   1/1     Running   0          10m   10.244.2.15   scheduler-lab-worker   <none>           <none>
		static-pod-deployment-8659b49b9b-n4jb8   1/1     Running   0          10m   10.244.2.11   scheduler-lab-worker   <none>           <none>
		static-pod-deployment-8659b49b9b-qbs2v   1/1     Running   0          10m   10.244.2.14   scheduler-lab-worker   <none>           <none>
		static-pod-deployment-8659b49b9b-swfjq   1/1     Running   0          10m   10.244.2.16   scheduler-lab-worker   <none>           <none>
		static-pod-deployment-8659b49b9b-tncn2   1/1     Running   0          10m   10.244.2.12   scheduler-lab-worker   <none>           <none>
		static-pod-deployment-8659b49b9b-wjz8b   1/1     Running   0          10m   10.244.2.18   scheduler-lab-worker   <none>           <none>
	```

##### Simple Bin-Packing Algorithm:
- Deployed 3 pods on each node, with resource requests as follows:
    - _Note: Each node in the cluster would have 10000m CPU and 8Gi memory allocatable, as they are docker containers on my local machine, deployed as a KIND cluster._
    - scheduler-lab-worker: 1000m CPU, 1Gi memory -> leaving 9000m CPU and 7Gi memory on the node.
    - scheduler-lab-worker2: 3000m CPU, 1Gi memory -> leaving 7000m CPU and 7Gi memory on the node.
    - scheduler-lab-worker3: 4000m CPU, 3Gi memory -> leaving 6000m CPU and 5Gi memory on the node.
    ```yaml
    apiVersion: apps/v1
    kind: Deployment
    metadata:
      name: scheduler-lab-worker-resource-requests
    spec:
      replicas: 1
      selector:
        matchLabels:
          app: resource-requests
      template:
        metadata:
          labels:
            app: resource-requests
        spec:
          schedulerName: kdapt-scheduler
          containers:
            - name: scheduler-lab-worker-resource-requests-container
              image: nginx:latest
              resources:
                requests:
                  cpu: "1000m"
                  memory: "1Gi"
                limits:
                  cpu: "1000m"
                  memory: "1Gi"
              ports:
                - containerPort: 80
          nodeSelector:
            kubernetes.io/hostname: scheduler-lab-worker
    ---
    apiVersion: apps/v1
    kind: Deployment
    metadata:
      name: scheduler-lab-worker2-resource-requests
    spec:
      replicas: 1
      selector:
        matchLabels:
          app: resource-requests
      template:
        metadata:
          labels:
            app: resource-requests
        spec:
          schedulerName: kdapt-scheduler
          nodeSelector:
            kubernetes.io/hostname: scheduler-lab-worker2
          containers:
            - name: scheduler-lab-worker2-resource-requests-container
              image: nginx:latest
              resources:
                requests:
                  cpu: "3000m"
                  memory: "1Gi"
                limits:
                  cpu: "3000m"
                  memory: "1Gi"
              ports:
                - containerPort: 80
    ---
    apiVersion: apps/v1
    kind: Deployment
    metadata:
      name: scheduler-lab-worker3-resource-requests
    spec:
      replicas: 1
      selector:
        matchLabels:
          app: resource-requests
      template:
        metadata:
          labels:
            app: resource-requests
        spec:
          schedulerName: kdapt-scheduler
          nodeSelector:
            kubernetes.io/hostname: scheduler-lab-worker3
          containers:
            - name: scheduler-lab-worker3-resource-requests-container
              image: nginx:latest
              resources:
                requests:
                  cpu: "4000m"
                  memory: "3Gi"
                limits:
                  cpu: "4000m"
                  memory: "3Gi"
              ports:
                - containerPort: 80
    ```

- The plugin at this point of time was returning a score based on the resource utilization of the node, giving more weight to CPU utilization.
    ```go
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
    ```
- In this case, when 8 replica pods are deployed, each requesting 1000m CPU, they shuould be scheduled in this order;
    - Majority of the pods should go to scheduler-lab-worker3, as it would have the most utilisation score.
    - The remaining pods should go to scheduler-lab-worker2, as it would have the second most utilisation score.
    - All 8 pods should be scheduled on these two nodes only.

    ```bash
    k8 get pods -o wide
    NAME                                                       READY   STATUS    RESTARTS   AGE    IP            NODE                    NOMINATED NODE   READINESS GATES
    scheduler-lab-worker-resource-requests-588f767dc8-nrzvh    1/1     Running   0          4h6m   10.244.2.3    scheduler-lab-worker    <none>           <none>
    scheduler-lab-worker2-resource-requests-64655b574b-7qdwn   1/1     Running   0          4h6m   10.244.1.3    scheduler-lab-worker2   <none>           <none>
    scheduler-lab-worker3-resource-requests-78b84b8548-c4hhd   1/1     Running   0          4h6m   10.244.3.5    scheduler-lab-worker3   <none>           <none>
    static-pod-deployment-cpu-heavy-7f6fd9659-4nk29            1/1     Running   0          64s    10.244.3.15   scheduler-lab-worker3   <none>           <none>
    static-pod-deployment-cpu-heavy-7f6fd9659-8v8b2            1/1     Running   0          64s    10.244.3.13   scheduler-lab-worker3   <none>           <none>
    static-pod-deployment-cpu-heavy-7f6fd9659-hbgp7            1/1     Running   0          64s    10.244.1.7    scheduler-lab-worker2   <none>           <none>
    static-pod-deployment-cpu-heavy-7f6fd9659-lvdcw            1/1     Running   0          64s    10.244.3.11   scheduler-lab-worker3   <none>           <none>
    static-pod-deployment-cpu-heavy-7f6fd9659-n7kwz            1/1     Running   0          64s    10.244.1.6    scheduler-lab-worker2   <none>           <none>
    static-pod-deployment-cpu-heavy-7f6fd9659-n89tp            1/1     Running   0          64s    10.244.3.14   scheduler-lab-worker3   <none>           <none>
    static-pod-deployment-cpu-heavy-7f6fd9659-nfjvn            1/1     Running   0          64s    10.244.3.12   scheduler-lab-worker3   <none>           <none>
    static-pod-deployment-cpu-heavy-7f6fd9659-pkhwx            1/1     Running   0          64s    10.244.1.5    scheduler-lab-worker2   <none>           <none>
    ```
- The below output shows that the node `scheduler-lab-worker3` was fully utilised before any pods were scheduled on the second best node `scheduler-lab-worker2`, as per the algorithm.
  ```bash
  ProviderID:                   kind://docker/scheduler-lab/scheduler-lab-worker3
  Non-terminated Pods:          (8 in total)
    Namespace                   Name                                                        CPU Requests  CPU Limits  Memory Requests  Memory Limits  Age
    ---------                   ----                                                        ------------  ----------  ---------------  -------------  ---
    default                     scheduler-lab-worker3-resource-requests-78b84b8548-c4hhd    4 (40%)       4 (40%)     3Gi (39%)        3Gi (39%)      4h6m
    default                     static-pod-deployment-cpu-heavy-7f6fd9659-4nk29             1 (10%)       0 (0%)      0 (0%)           0 (0%)         100s
    default                     static-pod-deployment-cpu-heavy-7f6fd9659-8v8b2             1 (10%)       0 (0%)      0 (0%)           0 (0%)         100s
    default                     static-pod-deployment-cpu-heavy-7f6fd9659-lvdcw             1 (10%)       0 (0%)      0 (0%)           0 (0%)         100s
    default                     static-pod-deployment-cpu-heavy-7f6fd9659-n89tp             1 (10%)       0 (0%)      0 (0%)           0 (0%)         100s
    default                     static-pod-deployment-cpu-heavy-7f6fd9659-nfjvn             1 (10%)       0 (0%)      0 (0%)           0 (0%)         100s
    kube-system                 kindnet-v2xgr                                               100m (1%)     100m (1%)   50Mi (0%)        50Mi (0%)      6d23h
    kube-system                 kube-proxy-95dl8                                            0 (0%)        0 (0%)      0 (0%)           0 (0%)         6d23h
  ```
  - We can see that there is a pod requesting 40% of the node's CPU, which was deployed intentionally to create artificial resource crunch, and then 5 more pods each requesting 10% of the node's CPU, as per the static pod deployment test. Total of 91% CPU was utilized on this node, and then the remaining pods were scheduled on the second best node `scheduler-lab-worker2`.
  ```bash
  ProviderID:                   kind://docker/scheduler-lab/scheduler-lab-worker2
  Non-terminated Pods:          (7 in total)
    Namespace                   Name                                                        CPU Requests  CPU Limits  Memory Requests  Memory Limits  Age
    ---------                   ----                                                        ------------  ----------  ---------------  -------------  ---
    default                     scheduler-lab-worker2-resource-requests-64655b574b-7qdwn    3 (30%)       3 (30%)     1Gi (13%)        1Gi (13%)      4h6m
    default                     static-pod-deployment-cpu-heavy-7f6fd9659-hbgp7             1 (10%)       0 (0%)      0 (0%)           0 (0%)         100s
    default                     static-pod-deployment-cpu-heavy-7f6fd9659-n7kwz             1 (10%)       0 (0%)      0 (0%)           0 (0%)         100s
    default                     static-pod-deployment-cpu-heavy-7f6fd9659-pkhwx             1 (10%)       0 (0%)      0 (0%)           0 (0%)         100s
    kube-system                 kdapt-scheduler-6b8cc67c98-98w5z                            0 (0%)        0 (0%)      0 (0%)           0 (0%)         119s
    kube-system                 kindnet-ttmnw                                               100m (1%)     100m (1%)   50Mi (0%)        50Mi (0%)      6d23h
    kube-system                 kube-proxy-tmvx7                                            0 (0%)        0 (0%)      0 (0%)           0 (0%)         6d23h
  ```

### Bin-Packing powered with run-time metrics:
- This is a variant of the bin-packing algorithm that uses run-time metrics to make scheduling decisions.
- Algorithm:
  - Node runtime metrics are collected periodically from the metrics server, not on every pod scheduling event
  ```go
  go k.runMetricsCollector(ctx, metricsClient, 10*time.Second)
  ```
  - All pods scored within a same burst mostly see the same runtime snapshot, while only requested resources change between scheduling decisions
  - Tradeoff:
    - _Lower scheduling overhead at the cost of slightly stale runtime information during fast bursts_
    - During a burst of pods, collecting fresh metrics after every individual pod placement would increase scheduler overhead and likely add scheduling latency.
  - The scheduler combines:
    - requested resources already on the node
    - smoothed runtime CPU and memory usage
    - the incoming pod’s own request  
    - a penalty for nodes already under high runtime pressure
  - Role of Smoothed Runtime Metrics
    - A key design aspect of the scheduler is the use of smoothed runtime values, rather than raw instantaneous metrics.
    - Why smoothing is necessary
      - Raw metrics from the metrics server are often:
        - noisy
        - bursty
        - sensitive to short-lived spikes
      - Using them directly can lead to:
        - unstable scheduling decisions
        - oscillations (frequent switching between nodes)
    - _The scheduler does not overreact to temporary spikes and instead relies on stabilized runtime signals._
  - The central adaptive idea in this algorithm is based on the mismatch between request-based and runtime-based utilization.
  ```bash
  cpuMismatch=∣requestedCpuUtil−runTimeCpuUtil∣
  memMismatch=∣requestedMemUtil−runTimeMemUtil∣
  ```
  - This mismatch measures how much the scheduler’s static view differs from actual observed node behavior
  - Interpretation:
    - Low mismatch means requests and runtime usage agree reasonably well.
    - High mismatch means the request-based model is not accurately reflecting current behavior.
    - _This allows the scheduler to decide how much trust to place in runtime metrics._
  - The algorithm computes adaptive weights alphaCpu and alphaMem:
    - The constants 0.8 and 0.4 act as slope coefficients for the mismatch.
    - Higher slope coefficients mean more weight is given to the mismatch. (They control how fast α increases as mismatch increases)
  ```bash
  alphaCpu=clamp(0.3+cpuMismatch*0.8,0,1)
  alphaMem=clamp(0.1+memMismatch*0.4,0,1)
  ```
  - This allows the scheduler to dynamically adjust the weight given to runtime metrics based on how well they match the requested resources.
    - CPU starts with a higher runtime influence (0.3 base weight).
    - Memory starts with a lower runtime influence (0.1 base weight).
    - As mismatch increases, runtime metrics gain more importance.
    - CPU reacts more strongly than memory, because CPU is compressible.
    - Memory remains more conservative, because memory pressure is riskier.

  - The final score is computed as:
  ```bash
  cpuScore=(1−alphaCpu)⋅projectedCpuUtil+alphaCpu⋅runTimeCpuUtil
  memScore=(1−alphaMem)⋅projectedMemUtil+alphaMem⋅runTimeMemUtil
  ```
  ```bash
  finalScore=wCpu⋅cpuScore+wMem⋅memScore−penality
  ```
    - Penality is calculated as:
    ```bash
    penality=0.0
    if runTimeCpuUtil>0.8 {
      penality+=0.2 // less penalty for higher cpu utilization, because CPU is compressible
    }
    if runTimeMemUtil>0.8 {
      penality+=0.25
    }
    ```
    - Using projected utilization instead of current requested utilization is what makes the algorithm forward-looking, and not reactive to current utilization.
    - It makes scheduling decisions future-aware, and helps in evaluating the consequences of the placement.
    - Prevents greedy scheduling decisions that may lead to resource crunch, and over packing.
    - Helps in packing efficiently, backingoff from nodes that are already stressed, and improving load balancing thereafter.
      - Say Node A's current utilization is 0.6, and Node B's current utilization is 0.4.
      - If the scheduler places the incoming pod on Node A, the projected utilization of Node A will be 0.9.
      - If the scheduler places the incoming pod on Node B, the projected utilization of Node B will be 0.5.
      - Traditional bin-packing algorithm would place the pod on Node A, because it has higher utilization. But:
        - It is better to place the pod on Node B, because Node A's 0.9 utilization would be harmful for the cluster.
      - Projected utilization with penality covers such cases. Adding explicit penalties for already-stressed nodes, or nodes which would be stressed by the placement.
    - Memory penality is more than CPU, because memory pressure is more dangerous than CPU pressure

```go
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
	klog.Infof("Final score: %v", finalScore)
	return int64(finalScore * float64(fwk.MaxNodeScore)), fwk.NewStatus(fwk.Success)

}
```
- Design considerations:
  - Packing pods tightly:
    - frees other nodes
    - reduces fragmentation
    - improves cluster efficiency
  - Tunable parameters for better performance:
    - Interval for collecting runtime metrics (e.g., every 10 seconds)
    ```go
    go k.runMetricsCollector(ctx, metricsClient, 10*time.Second)
    ```
    - Smoothing factor for runtime metrics (e.g., exponential moving average = 0.3)
    ```go
    func (k *Kdapt) collectMetrics(
      ctx context.Context,
      metricsClient *metricsclient.Clientset,
    ) (map[string]NodeRuntimeMetrics, error) {
      ema := 0.3
    ...
    - Alpha base weight for projected utilization vs runtime utilization (e.g., 0.3 for CPU, 0.1 for memory)
    ```go
    alphaCpu := clamp(0.3+cpuMismatch*0.8, 0, 1) // 0.3 is the base weight, when there is no mismatch.
    alphaMem := clamp(0.1+memMismatch*0.4, 0, 1)
    ```
    - Penalty thresholds for high runtime utilization (e.g., 0.8 for CPU and memory)
    ```go
    penality := 0.0
    if runTimeCpuUtil > 0.8 {
      penality += 0.2 // less penalty for higher cpu utilization, because CPU is compressible
    }
    if runTimeMemUtil > 0.8 {
      penality += 0.25
    }
    ```
    - CPU and memory weights in the final score (e.g., 0.6 for CPU, 0.4 for memory)
    ```go
    wCpu := 0.6
    wMem := 0.4
    ```
  - _This aligns with real systems like Borg and Kubernetes bin-packing strategies_
  - Safety penalties:
    - Prevents overloading nodes
    - Protects against resource crunch
    - Avoids greedy scheduling decisions
    - Provides safety guarantees

- In short, it is a hybrid between:
  - safe request-based bin-packing
  - adaptive runtime-sensitive placement

###### Refer to `Experiments/cpu-heavy-burst-scheduling.md` for the experiment details and results of this algorithm when the burst is CPU heavy. The results show that the scheduler prefers packing on the most utilized node, while avoiding overloading it, and then moves to the second best node, which is the expected behavior.
###### Refer to `Experiments/memory-heavy-burst-scheduling.md` for the experiment details and results of this algorithm when the burst is memory heavy. The results show that the scheduler is more conservative in packing on the most utilized node, and prefers to spread the pods more evenly across nodes, which is also the expected behavior given that memory pressure is more dangerous than CPU pressure.

#### Refinements:
- There was a problem with smoothed runtime values. They are used in calculating mismatch too, which is not ideal, because they are slow to react to changes in runtime behavior. This can lead to suboptimal scheduling decisions during fast bursts of pods, or when there are sudden changes in node behavior.
- Example: 
  - When a pod terminates, the raw CPU usage drops from 2000m to 28m, but the smoothed value takes time to catch up, leading the scheduler to think that the node is still busy for a while, which is wrong. Similarly, when there is a spike in CPU usage, the raw value jumps from 28m to 2000m, but the smoothed value takes time to catch up, leading the scheduler to underestimate the node's busy state and potentially schedule more pods onto an already overloaded node, which is also wrong.
  - This can be solved by reducing the interval for collecting runtime metrics, or by using a more responsive smoothing algorithm that can react faster to changes in runtime behavior. But by reducing the interval for collecting runtime metrics, we would be increasing the scheduling overhead, which is not ideal.
- This behaviour is not seen in the above experiments because the actual resources are not being utilised by the simple nginx pod. To see this behaviour, we would need to deploy a more realistic CPU or memory hungry pod that actually utilises the requested resources, and then observe the scheduling decisions during the burst, and how they are affected by the smoothed values.
- A new parameter `beta` is introduced to control the effectiveResourceUtilization which is being used in calculation of mismatch between requested and actual resource usage.
- Score function is effectively changed.
```go
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
```
- The final score and penality is now based on effectiveCpuUtil and effectiveMemUtil instead of smoothed values, which makes the scheduler more responsive to changes in runtime behavior, while still benefiting from the stability of smoothed metrics when they are accurate. This should lead to better scheduling decisions during fast bursts of pods, or when there are sudden changes in node behavior.
```go
cpuScore := (1-alphaCpu)*projectedCpuUtil+alphaCpu*effectiveCpuUtil
memScore := (1-alphaMem)*projectedMemUtil+alphaMem*effectiveMemUtil
...
if effectiveCpuUtil > 0.8 {
  penality += 0.2 // less penalty for higher cpu utilization, because CPU is compressible
}
if effectiveMemUtil > 0.8 {
  penality += 0.25
}
```

## Next Steps:
- Staged arrivals

| Deployment | Requests (CPU / Mem) | Actual Usage | Pattern |
|---|---|---|---|
| over-prov-cpu | 3000m / 512Mi | ~100m / ~50Mi | Over-provisioned CPU |
| over-prov-mem | 500m / 2Gi | ~100m / ~200Mi | Over-provisioned Memory |
| cpu-matched | 2000m / 256Mi | ~2000m / ~10Mi | CPU well-matched |
| mem-matched | 200m / 1Gi | ~200m / ~900Mi | Memory well-matched |
| cpu-hungry | 500m / 256Mi | ~2000m / ~10Mi | Under-provisioned CPU (bursts past request) |
| mem-hungry | 200m / 512Mi | ~200m / ~1.5Gi | Under-provisioned Memory (bursts past request) |

- Read Borg design paper to understand the inspiration behind Kubernetes scheduling.
- Validate against kube-scheduler 
  - pod placement distribution by node
  - scheduling latency per pod
  - how tightly each scheduler packs before spilling to the next node
- Explore more complex scheduling scenarios with dynamic resource requests.

To Document:
- Staged arrivals output:
  -  `bash deploy.sh gradual 15`
```bash 
I0506 22:41:33.019445       1 kdapt.go:71] nodeMetrics {name=scheduler-lab-control-plane, cpuMilli=139.00, memoryBytes=857223168.00, smoothedCpuMilli=138.59, smoothedMemoryBytes=857120176.01}
I0506 22:41:33.019488       1 kdapt.go:71] nodeMetrics {name=scheduler-lab-worker, cpuMilli=30.00, memoryBytes=274472960.00, smoothedCpuMilli=29.80, smoothedMemoryBytes=274738129.31}
I0506 22:41:33.019494       1 kdapt.go:71] nodeMetrics {name=scheduler-lab-worker2, cpuMilli=23.00, memoryBytes=244834304.00, smoothedCpuMilli=26.22, smoothedMemoryBytes=241479145.51}
I0506 22:41:33.019497       1 kdapt.go:71] nodeMetrics {name=scheduler-lab-worker3, cpuMilli=18.00, memoryBytes=259493888.00, smoothedCpuMilli=22.93, smoothedMemoryBytes=261324736.91}
I0506 22:41:33.019500       1 kdapt.go:80] -----------------------------------------------------------------
I0506 22:41:43.017898       1 kdapt.go:71] nodeMetrics {name=scheduler-lab-control-plane, cpuMilli=139.00, memoryBytes=857223168.00, smoothedCpuMilli=138.71, smoothedMemoryBytes=857151073.61}
I0506 22:41:43.017950       1 kdapt.go:71] nodeMetrics {name=scheduler-lab-worker, cpuMilli=30.00, memoryBytes=274472960.00, smoothedCpuMilli=29.86, smoothedMemoryBytes=274658578.51}
I0506 22:41:43.017954       1 kdapt.go:71] nodeMetrics {name=scheduler-lab-worker2, cpuMilli=23.00, memoryBytes=244834304.00, smoothedCpuMilli=25.26, smoothedMemoryBytes=242485693.06}
I0506 22:41:43.017957       1 kdapt.go:71] nodeMetrics {name=scheduler-lab-worker3, cpuMilli=18.00, memoryBytes=259493888.00, smoothedCpuMilli=21.45, smoothedMemoryBytes=260775482.24}
I0506 22:41:43.017960       1 kdapt.go:80] -----------------------------------------------------------------
I0506 22:41:43.146642       1 kdapt.go:241] pod=deploy-cpu-matched-67dc8cb568-6hv97 node=scheduler-lab-worker2 | cpu: req=0.01 rt=0.00 mismatch=0.01 proj=0.21 alpha=0.31 | mem: req=0.01 rt=0.03 mismatch=0.02 proj=0.04 alpha=0.11 | penalty=0.00 final=0.1031
I0506 22:41:43.146699       1 kdapt.go:241] pod=deploy-cpu-matched-67dc8cb568-6hv97 node=scheduler-lab-worker | cpu: req=0.02 rt=0.00 mismatch=0.02 proj=0.22 alpha=0.31 | mem: req=0.03 rt=0.03 mismatch=0.00 proj=0.06 alpha=0.10 | penalty=0.00 final=0.1157
I0506 22:41:43.146764       1 kdapt.go:241] pod=deploy-cpu-matched-67dc8cb568-6hv97 node=scheduler-lab-worker3 | cpu: req=0.01 rt=0.00 mismatch=0.01 proj=0.21 alpha=0.31 | mem: req=0.01 rt=0.03 mismatch=0.03 proj=0.04 alpha=0.11 | penalty=0.00 final=0.1031
I0506 22:41:43.150111       1 kdapt.go:241] pod=deploy-cpu-matched-67dc8cb568-mf57p node=scheduler-lab-worker2 | cpu: req=0.01 rt=0.00 mismatch=0.01 proj=0.21 alpha=0.31 | mem: req=0.01 rt=0.03 mismatch=0.02 proj=0.04 alpha=0.11 | penalty=0.00 final=0.1031
I0506 22:41:43.150135       1 kdapt.go:241] pod=deploy-cpu-matched-67dc8cb568-mf57p node=scheduler-lab-worker3 | cpu: req=0.01 rt=0.00 mismatch=0.01 proj=0.21 alpha=0.31 | mem: req=0.01 rt=0.03 mismatch=0.03 proj=0.04 alpha=0.11 | penalty=0.00 final=0.1031
I0506 22:41:43.150155       1 kdapt.go:241] pod=deploy-cpu-matched-67dc8cb568-mf57p node=scheduler-lab-worker | cpu: req=0.22 rt=0.00 mismatch=0.22 proj=0.42 alpha=0.47 | mem: req=0.06 rt=0.03 mismatch=0.03 proj=0.10 alpha=0.11 | penalty=0.00 final=0.1695
I0506 22:41:43.150432       1 kdapt.go:241] pod=deploy-cpu-matched-67dc8cb568-27b2t node=scheduler-lab-worker | cpu: req=0.42 rt=0.00 mismatch=0.42 proj=0.62 alpha=0.63 | mem: req=0.10 rt=0.03 mismatch=0.06 proj=0.13 alpha=0.13 | penalty=0.00 final=0.1845
I0506 22:41:43.150445       1 kdapt.go:241] pod=deploy-cpu-matched-67dc8cb568-27b2t node=scheduler-lab-worker2 | cpu: req=0.01 rt=0.00 mismatch=0.01 proj=0.21 alpha=0.31 | mem: req=0.01 rt=0.03 mismatch=0.02 proj=0.04 alpha=0.11 | penalty=0.00 final=0.1031
I0506 22:41:43.150450       1 kdapt.go:241] pod=deploy-cpu-matched-67dc8cb568-27b2t node=scheduler-lab-worker3 | cpu: req=0.01 rt=0.00 mismatch=0.01 proj=0.21 alpha=0.31 | mem: req=0.01 rt=0.03 mismatch=0.03 proj=0.04 alpha=0.11 | penalty=0.00 final=0.1031
I0506 22:41:53.017947       1 kdapt.go:71] nodeMetrics {name=scheduler-lab-control-plane, cpuMilli=135.00, memoryBytes=859275264.00, smoothedCpuMilli=137.60, smoothedMemoryBytes=857788330.73}
I0506 22:41:53.018002       1 kdapt.go:71] nodeMetrics {name=scheduler-lab-worker, cpuMilli=307.00, memoryBytes=306081792.00, smoothedCpuMilli=113.00, smoothedMemoryBytes=284085542.56}
I0506 22:41:53.018006       1 kdapt.go:71] nodeMetrics {name=scheduler-lab-worker2, cpuMilli=24.00, memoryBytes=245166080.00, smoothedCpuMilli=24.88, smoothedMemoryBytes=243289809.14}
I0506 22:41:53.018009       1 kdapt.go:71] nodeMetrics {name=scheduler-lab-worker3, cpuMilli=17.00, memoryBytes=258981888.00, smoothedCpuMilli=20.12, smoothedMemoryBytes=260237403.97}
I0506 22:41:53.018012       1 kdapt.go:80] -----------------------------------------------------------------
I0506 22:41:58.317115       1 kdapt.go:241] pod=deploy-mem-matched-5c49fb5d9d-5d2pp node=scheduler-lab-worker | cpu: req=0.62 rt=0.02 mismatch=0.60 proj=0.64 alpha=0.78 | mem: req=0.13 rt=0.03 mismatch=0.10 proj=0.26 alpha=0.14 | penalty=0.00 final=0.1883
I0506 22:41:58.317161       1 kdapt.go:241] pod=deploy-mem-matched-5c49fb5d9d-5d2pp node=scheduler-lab-worker2 | cpu: req=0.01 rt=0.00 mismatch=0.01 proj=0.03 alpha=0.31 | mem: req=0.01 rt=0.03 mismatch=0.02 proj=0.14 alpha=0.11 | penalty=0.00 final=0.0631
I0506 22:41:58.317173       1 kdapt.go:241] pod=deploy-mem-matched-5c49fb5d9d-5d2pp node=scheduler-lab-worker3 | cpu: req=0.01 rt=0.00 mismatch=0.01 proj=0.03 alpha=0.31 | mem: req=0.01 rt=0.03 mismatch=0.03 proj=0.14 alpha=0.11 | penalty=0.00 final=0.0630
I0506 22:41:58.324799       1 kdapt.go:241] pod=deploy-mem-matched-5c49fb5d9d-f7h26 node=scheduler-lab-worker | cpu: req=0.64 rt=0.02 mismatch=0.62 proj=0.66 alpha=0.79 | mem: req=0.26 rt=0.03 mismatch=0.23 proj=0.39 alpha=0.19 | penalty=0.00 final=0.2224
I0506 22:41:58.324825       1 kdapt.go:241] pod=deploy-mem-matched-5c49fb5d9d-f7h26 node=scheduler-lab-worker3 | cpu: req=0.01 rt=0.00 mismatch=0.01 proj=0.03 alpha=0.31 | mem: req=0.01 rt=0.03 mismatch=0.03 proj=0.14 alpha=0.11 | penalty=0.00 final=0.0630
I0506 22:41:58.324804       1 kdapt.go:241] pod=deploy-mem-matched-5c49fb5d9d-f7h26 node=scheduler-lab-worker2 | cpu: req=0.01 rt=0.00 mismatch=0.01 proj=0.03 alpha=0.31 | mem: req=0.01 rt=0.03 mismatch=0.02 proj=0.14 alpha=0.11 | penalty=0.00 final=0.0631
I0506 22:41:58.325103       1 kdapt.go:241] pod=deploy-mem-matched-5c49fb5d9d-twpnq node=scheduler-lab-worker | cpu: req=0.66 rt=0.02 mismatch=0.64 proj=0.68 alpha=0.81 | mem: req=0.39 rt=0.03 mismatch=0.36 proj=0.52 alpha=0.24 | penalty=0.00 final=0.2508
I0506 22:41:58.325111       1 kdapt.go:241] pod=deploy-mem-matched-5c49fb5d9d-twpnq node=scheduler-lab-worker2 | cpu: req=0.01 rt=0.00 mismatch=0.01 proj=0.03 alpha=0.31 | mem: req=0.01 rt=0.03 mismatch=0.02 proj=0.14 alpha=0.11 | penalty=0.00 final=0.0631
I0506 22:41:58.325115       1 kdapt.go:241] pod=deploy-mem-matched-5c49fb5d9d-twpnq node=scheduler-lab-worker3 | cpu: req=0.01 rt=0.00 mismatch=0.01 proj=0.03 alpha=0.31 | mem: req=0.01 rt=0.03 mismatch=0.03 proj=0.14 alpha=0.11 | penalty=0.00 final=0.0630
I0506 22:42:03.048279       1 kdapt.go:71] nodeMetrics {name=scheduler-lab-control-plane, cpuMilli=135.00, memoryBytes=859275264.00, smoothedCpuMilli=136.82, smoothedMemoryBytes=858234410.71}
I0506 22:42:03.048393       1 kdapt.go:71] nodeMetrics {name=scheduler-lab-worker, cpuMilli=307.00, memoryBytes=306081792.00, smoothedCpuMilli=171.20, smoothedMemoryBytes=290684417.39}
I0506 22:42:03.048401       1 kdapt.go:71] nodeMetrics {name=scheduler-lab-worker2, cpuMilli=24.00, memoryBytes=245166080.00, smoothedCpuMilli=24.62, smoothedMemoryBytes=243852690.40}
I0506 22:42:03.048408       1 kdapt.go:71] nodeMetrics {name=scheduler-lab-worker3, cpuMilli=17.00, memoryBytes=258981888.00, smoothedCpuMilli=19.18, smoothedMemoryBytes=259860749.18}
I0506 22:42:03.048413       1 kdapt.go:80] -----------------------------------------------------------------
I0506 22:42:13.027053       1 kdapt.go:71] nodeMetrics {name=scheduler-lab-control-plane, cpuMilli=116.00, memoryBytes=856031232.00, smoothedCpuMilli=130.57, smoothedMemoryBytes=857573457.10}
I0506 22:42:13.027199       1 kdapt.go:71] nodeMetrics {name=scheduler-lab-worker, cpuMilli=6013.00, memoryBytes=306257920.00, smoothedCpuMilli=1923.74, smoothedMemoryBytes=295356468.17}
I0506 22:42:13.027204       1 kdapt.go:71] nodeMetrics {name=scheduler-lab-worker2, cpuMilli=18.00, memoryBytes=245014528.00, smoothedCpuMilli=22.63, smoothedMemoryBytes=244201241.68}
I0506 22:42:13.027207       1 kdapt.go:71] nodeMetrics {name=scheduler-lab-worker3, cpuMilli=18.00, memoryBytes=259325952.00, smoothedCpuMilli=18.83, smoothedMemoryBytes=259700310.02}
I0506 22:42:13.027210       1 kdapt.go:80] -----------------------------------------------------------------
I0506 22:42:13.589178       1 kdapt.go:241] pod=deploy-over-prov-cpu-5b74dd784d-22jqz node=scheduler-lab-worker3 | cpu: req=0.01 rt=0.00 mismatch=0.01 proj=0.31 alpha=0.31 | mem: req=0.01 rt=0.03 mismatch=0.03 proj=0.07 alpha=0.11 | penalty=0.00 final=0.1563
I0506 22:42:13.589240       1 kdapt.go:241] pod=deploy-over-prov-cpu-5b74dd784d-22jqz node=scheduler-lab-worker | cpu: req=0.68 rt=0.47 mismatch=0.21 proj=0.98 alpha=0.47 | mem: req=0.52 rt=0.04 mismatch=0.49 proj=0.59 alpha=0.29 | penalty=0.00 final=0.6150
I0506 22:42:13.589659       1 kdapt.go:241] pod=deploy-over-prov-cpu-5b74dd784d-22jqz node=scheduler-lab-worker2 | cpu: req=0.01 rt=0.00 mismatch=0.01 proj=0.31 alpha=0.31 | mem: req=0.01 rt=0.03 mismatch=0.02 proj=0.07 alpha=0.11 | penalty=0.00 final=0.1563
I0506 22:42:13.609838       1 kdapt.go:241] pod=deploy-over-prov-cpu-5b74dd784d-xzngv node=scheduler-lab-worker2 | cpu: req=0.01 rt=0.00 mismatch=0.01 proj=0.31 alpha=0.31 | mem: req=0.01 rt=0.03 mismatch=0.02 proj=0.07 alpha=0.11 | penalty=0.00 final=0.1563
I0506 22:42:13.609898       1 kdapt.go:241] pod=deploy-over-prov-cpu-5b74dd784d-xzngv node=scheduler-lab-worker3 | cpu: req=0.01 rt=0.00 mismatch=0.01 proj=0.31 alpha=0.31 | mem: req=0.01 rt=0.03 mismatch=0.03 proj=0.07 alpha=0.11 | penalty=0.00 final=0.1563
I0506 22:42:13.611023       1 kdapt.go:241] pod=deploy-over-prov-cpu-5b74dd784d-7nvjc node=scheduler-lab-worker2 | cpu: req=0.31 rt=0.00 mismatch=0.31 proj=0.61 alpha=0.55 | mem: req=0.07 rt=0.03 mismatch=0.04 proj=0.14 alpha=0.12 | penalty=0.00 final=0.2166
I0506 22:42:13.611038       1 kdapt.go:241] pod=deploy-over-prov-cpu-5b74dd784d-7nvjc node=scheduler-lab-worker3 | cpu: req=0.01 rt=0.00 mismatch=0.01 proj=0.31 alpha=0.31 | mem: req=0.01 rt=0.03 mismatch=0.03 proj=0.07 alpha=0.11 | penalty=0.00 final=0.1563
I0506 22:42:23.050235       1 kdapt.go:71] nodeMetrics {name=scheduler-lab-control-plane, cpuMilli=248.00, memoryBytes=858140672.00, smoothedCpuMilli=165.80, smoothedMemoryBytes=857743621.57}
I0506 22:42:23.050479       1 kdapt.go:71] nodeMetrics {name=scheduler-lab-worker, cpuMilli=6530.00, memoryBytes=3203854336.00, smoothedCpuMilli=3305.62, smoothedMemoryBytes=1167905828.52}
I0506 22:42:23.050484       1 kdapt.go:71] nodeMetrics {name=scheduler-lab-worker2, cpuMilli=102.00, memoryBytes=281964544.00, smoothedCpuMilli=46.44, smoothedMemoryBytes=255530232.38}
I0506 22:42:23.050487       1 kdapt.go:71] nodeMetrics {name=scheduler-lab-worker3, cpuMilli=24.00, memoryBytes=259616768.00, smoothedCpuMilli=20.38, smoothedMemoryBytes=259675247.42}
I0506 22:42:23.050491       1 kdapt.go:80] -----------------------------------------------------------------
I0506 22:42:28.820732       1 kdapt.go:241] pod=deploy-over-prov-mem-69ffc6cdd4-h9q57 node=scheduler-lab-worker2 | cpu: req=0.61 rt=0.01 mismatch=0.60 proj=0.66 alpha=0.78 | mem: req=0.14 rt=0.03 mismatch=0.11 proj=0.40 alpha=0.14 | penalty=0.00 final=0.2284
I0506 22:42:28.820874       1 kdapt.go:241] pod=deploy-over-prov-mem-69ffc6cdd4-h9q57 node=scheduler-lab-worker3 | cpu: req=0.01 rt=0.00 mismatch=0.01 proj=0.06 alpha=0.31 | mem: req=0.01 rt=0.03 mismatch=0.03 proj=0.27 alpha=0.11 | penalty=0.00 final=0.1220
I0506 22:42:28.824871       1 kdapt.go:241] pod=deploy-over-prov-mem-69ffc6cdd4-86jbl node=scheduler-lab-worker2 | cpu: req=0.66 rt=0.01 mismatch=0.65 proj=0.71 alpha=0.82 | mem: req=0.40 rt=0.03 mismatch=0.37 proj=0.66 alpha=0.25 | penalty=0.00 final=0.2815
I0506 22:42:28.825168       1 kdapt.go:241] pod=deploy-over-prov-mem-69ffc6cdd4-86jbl node=scheduler-lab-worker3 | cpu: req=0.01 rt=0.00 mismatch=0.01 proj=0.06 alpha=0.31 | mem: req=0.01 rt=0.03 mismatch=0.03 proj=0.27 alpha=0.11 | penalty=0.00 final=0.1220
I0506 22:42:28.828913       1 kdapt.go:241] pod=deploy-over-prov-mem-69ffc6cdd4-896kd node=scheduler-lab-worker2 | cpu: req=0.71 rt=0.01 mismatch=0.70 proj=0.76 alpha=0.86 | mem: req=0.66 rt=0.03 mismatch=0.63 proj=0.92 alpha=0.35 | penalty=0.00 final=0.3103
I0506 22:42:28.829157       1 kdapt.go:241] pod=deploy-over-prov-mem-69ffc6cdd4-896kd node=scheduler-lab-worker3 | cpu: req=0.01 rt=0.00 mismatch=0.01 proj=0.06 alpha=0.31 | mem: req=0.01 rt=0.03 mismatch=0.03 proj=0.27 alpha=0.11 | penalty=0.00 final=0.1220
I0506 22:42:33.060038       1 kdapt.go:71] nodeMetrics {name=scheduler-lab-control-plane, cpuMilli=248.00, memoryBytes=858140672.00, smoothedCpuMilli=190.46, smoothedMemoryBytes=857862736.70}
I0506 22:42:33.060167       1 kdapt.go:71] nodeMetrics {name=scheduler-lab-worker, cpuMilli=6530.00, memoryBytes=3203854336.00, smoothedCpuMilli=4272.93, smoothedMemoryBytes=1778690380.77}
I0506 22:42:33.060174       1 kdapt.go:71] nodeMetrics {name=scheduler-lab-worker2, cpuMilli=102.00, memoryBytes=281964544.00, smoothedCpuMilli=63.11, smoothedMemoryBytes=263460525.86}
I0506 22:42:33.060176       1 kdapt.go:71] nodeMetrics {name=scheduler-lab-worker3, cpuMilli=24.00, memoryBytes=259616768.00, smoothedCpuMilli=21.47, smoothedMemoryBytes=259657703.59}
I0506 22:42:33.060828       1 kdapt.go:80] -----------------------------------------------------------------
I0506 22:42:43.040235       1 kdapt.go:71] nodeMetrics {name=scheduler-lab-control-plane, cpuMilli=216.00, memoryBytes=857182208.00, smoothedCpuMilli=198.12, smoothedMemoryBytes=857658578.09}
I0506 22:42:43.040356       1 kdapt.go:71] nodeMetrics {name=scheduler-lab-worker, cpuMilli=6638.00, memoryBytes=3204665344.00, smoothedCpuMilli=4982.45, smoothedMemoryBytes=2206482869.74}
I0506 22:42:43.040363       1 kdapt.go:71] nodeMetrics {name=scheduler-lab-worker2, cpuMilli=50.00, memoryBytes=282664960.00, smoothedCpuMilli=59.18, smoothedMemoryBytes=269221856.10}
I0506 22:42:43.040368       1 kdapt.go:71] nodeMetrics {name=scheduler-lab-worker3, cpuMilli=33.00, memoryBytes=259522560.00, smoothedCpuMilli=24.93, smoothedMemoryBytes=259617160.51}
I0506 22:42:43.040371       1 kdapt.go:80] -----------------------------------------------------------------
I0506 22:42:44.052608       1 kdapt.go:241] pod=deploy-cpu-hungry-5464bdc95b-b2df7 node=scheduler-lab-worker3 | cpu: req=0.01 rt=0.00 mismatch=0.01 proj=0.06 alpha=0.31 | mem: req=0.01 rt=0.03 mismatch=0.03 proj=0.04 alpha=0.11 | penalty=0.00 final=0.0408
I0506 22:42:44.050971       1 kdapt.go:241] pod=deploy-cpu-hungry-5464bdc95b-b2df7 node=scheduler-lab-worker2 | cpu: req=0.76 rt=0.01 mismatch=0.75 proj=0.81 alpha=0.90 | mem: req=0.92 rt=0.03 mismatch=0.89 proj=0.95 alpha=0.46 | penalty=0.00 final=0.2639
I0506 22:42:44.064235       1 kdapt.go:241] pod=deploy-cpu-hungry-5464bdc95b-dd94w node=scheduler-lab-worker2 | cpu: req=0.81 rt=0.01 mismatch=0.80 proj=0.86 alpha=0.94 | mem: req=0.95 rt=0.03 mismatch=0.92 proj=0.99 alpha=0.47 | penalty=0.00 final=0.2484
I0506 22:42:44.064933       1 kdapt.go:241] pod=deploy-cpu-hungry-5464bdc95b-dd94w node=scheduler-lab-worker3 | cpu: req=0.01 rt=0.00 mismatch=0.01 proj=0.06 alpha=0.31 | mem: req=0.01 rt=0.03 mismatch=0.03 proj=0.04 alpha=0.11 | penalty=0.00 final=0.0408
I0506 22:42:53.146950       1 kdapt.go:71] nodeMetrics {name=scheduler-lab-control-plane, cpuMilli=208.00, memoryBytes=858710016.00, smoothedCpuMilli=201.09, smoothedMemoryBytes=857974009.46}
I0506 22:42:53.148695       1 kdapt.go:71] nodeMetrics {name=scheduler-lab-worker, cpuMilli=6580.00, memoryBytes=3208290304.00, smoothedCpuMilli=5461.72, smoothedMemoryBytes=2507025100.02}
I0506 22:42:53.148734       1 kdapt.go:71] nodeMetrics {name=scheduler-lab-worker2, cpuMilli=226.00, memoryBytes=350953472.00, smoothedCpuMilli=109.22, smoothedMemoryBytes=293741340.87}
I0506 22:42:53.149221       1 kdapt.go:71] nodeMetrics {name=scheduler-lab-worker3, cpuMilli=86.00, memoryBytes=269578240.00, smoothedCpuMilli=43.25, smoothedMemoryBytes=262605484.36}
I0506 22:42:53.149234       1 kdapt.go:80] -----------------------------------------------------------------
I0506 22:42:59.658850       1 kdapt.go:241] pod=deploy-mem-hungry-657fb577df-g8znt node=scheduler-lab-worker | cpu: req=0.98 rt=0.57 mismatch=0.41 proj=1.00 alpha=0.63 | mem: req=0.59 rt=0.32 mismatch=0.26 proj=0.65 alpha=0.21 | penalty=0.00 final=0.6691
I0506 22:42:59.659371       1 kdapt.go:241] pod=deploy-mem-hungry-657fb577df-g8znt node=scheduler-lab-worker3 | cpu: req=0.06 rt=0.01 mismatch=0.05 proj=0.08 alpha=0.34 | mem: req=0.04 rt=0.03 mismatch=0.01 proj=0.10 alpha=0.10 | penalty=0.00 final=0.0716
I0506 22:43:03.628881       1 kdapt.go:71] nodeMetrics {name=scheduler-lab-control-plane, cpuMilli=216.00, memoryBytes=859525120.00, smoothedCpuMilli=205.56, smoothedMemoryBytes=858439342.62}
I0506 22:43:03.629059       1 kdapt.go:71] nodeMetrics {name=scheduler-lab-worker, cpuMilli=4246.00, memoryBytes=3208986624.00, smoothedCpuMilli=5097.00, smoothedMemoryBytes=2717613557.21}
I0506 22:43:03.629074       1 kdapt.go:71] nodeMetrics {name=scheduler-lab-worker2, cpuMilli=3530.00, memoryBytes=354086912.00, smoothedCpuMilli=1135.46, smoothedMemoryBytes=311845012.21}
I0506 22:43:03.629078       1 kdapt.go:71] nodeMetrics {name=scheduler-lab-worker3, cpuMilli=1991.00, memoryBytes=269447168.00, smoothedCpuMilli=627.57, smoothedMemoryBytes=264657989.45}
I0506 22:43:03.629080       1 kdapt.go:80] -----------------------------------------------------------------
I0506 22:43:14.747551       1 kdapt.go:71] nodeMetrics {name=scheduler-lab-control-plane, cpuMilli=216.00, memoryBytes=859525120.00, smoothedCpuMilli=208.69, smoothedMemoryBytes=858765075.84}
I0506 22:43:14.750521       1 kdapt.go:71] nodeMetrics {name=scheduler-lab-worker, cpuMilli=4246.00, memoryBytes=3208986624.00, smoothedCpuMilli=4841.70, smoothedMemoryBytes=2865025477.25}
I0506 22:43:14.750529       1 kdapt.go:71] nodeMetrics {name=scheduler-lab-worker2, cpuMilli=3530.00, memoryBytes=354086912.00, smoothedCpuMilli=1853.82, smoothedMemoryBytes=324517582.15}
I0506 22:43:14.750535       1 kdapt.go:71] nodeMetrics {name=scheduler-lab-worker3, cpuMilli=1991.00, memoryBytes=269447168.00, smoothedCpuMilli=1036.60, smoothedMemoryBytes=266094743.02}
I0506 22:43:14.750540       1 kdapt.go:80] -----------------------------------------------------------------
```
```bash
[18:41:43] Applying: cpu.yaml
deployment.apps/deploy-cpu-matched created
[18:41:43] Waiting 15s before next arrival...
[18:41:58] --- Cluster snapshot ---
NAME                          CPU(cores)   CPU(%)   MEMORY(bytes)   MEMORY(%)   
scheduler-lab-control-plane   135m         1%       819Mi           10%         
scheduler-lab-worker          307m         3%       291Mi           3%          
scheduler-lab-worker2         24m          0%       233Mi           2%          
scheduler-lab-worker3         17m          0%       246Mi           3%          

[18:41:58] Applying: mem.yaml
deployment.apps/deploy-mem-matched created
[18:41:58] Waiting 15s before next arrival...
[18:42:13] --- Cluster snapshot ---
NAME                          CPU(cores)   CPU(%)   MEMORY(bytes)   MEMORY(%)   
scheduler-lab-control-plane   116m         1%       816Mi           10%         
scheduler-lab-worker          6013m        60%      292Mi           3%          
scheduler-lab-worker2         18m          0%       233Mi           2%          
scheduler-lab-worker3         18m          0%       247Mi           3%          

[18:42:13] Applying: heavy-cpu-request.yaml
deployment.apps/deploy-over-prov-cpu created
[18:42:13] Waiting 15s before next arrival...
[18:42:28] --- Cluster snapshot ---
NAME                          CPU(cores)   CPU(%)   MEMORY(bytes)   MEMORY(%)   
scheduler-lab-control-plane   248m         2%       818Mi           10%         
scheduler-lab-worker          6530m        65%      3055Mi          38%         
scheduler-lab-worker2         102m         1%       268Mi           3%          
scheduler-lab-worker3         24m          0%       247Mi           3%          

[18:42:28] Applying: heavy-mem-request.yaml
deployment.apps/deploy-over-prov-mem created
[18:42:28] Waiting 15s before next arrival...
[18:42:43] --- Cluster snapshot ---
NAME                          CPU(cores)   CPU(%)   MEMORY(bytes)   MEMORY(%)   
scheduler-lab-control-plane   216m         2%       817Mi           10%         
scheduler-lab-worker          6638m        66%      3056Mi          38%         
scheduler-lab-worker2         50m          0%       269Mi           3%          
scheduler-lab-worker3         33m          0%       247Mi           3%          

[18:42:43] Applying: heavy-cpu-util.yaml
deployment.apps/deploy-cpu-hungry created
[18:42:44] Waiting 15s before next arrival...
[18:42:59] --- Cluster snapshot ---
NAME                          CPU(cores)   CPU(%)   MEMORY(bytes)   MEMORY(%)   
scheduler-lab-control-plane   208m         2%       818Mi           10%         
scheduler-lab-worker          6580m        65%      3059Mi          39%         
scheduler-lab-worker2         226m         2%       334Mi           4%          
scheduler-lab-worker3         86m          0%       257Mi           3%          

[18:42:59] Applying: heavy-mem-util.yaml
deployment.apps/deploy-mem-hungry created
[18:42:59] Waiting 15s before next arrival...
[18:43:14] --- Cluster snapshot ---
NAME                          CPU(cores)   CPU(%)   MEMORY(bytes)   MEMORY(%)   
scheduler-lab-control-plane   216m         2%       819Mi           10%         
scheduler-lab-worker          4246m        42%      3060Mi          39%         
scheduler-lab-worker2         3530m        35%      337Mi           4%          
scheduler-lab-worker3         1991m        19%      256Mi           3%          

[18:43:14] All workloads applied. Monitoring for 60s to observe scheduling decisions...
[18:44:14] Final state:
[18:44:14] --- Cluster snapshot ---
NAME                          CPU(cores)   CPU(%)   MEMORY(bytes)   MEMORY(%)   
scheduler-lab-control-plane   395m         3%       603Mi           7%          
scheduler-lab-worker          3757m        37%      4479Mi          57%         
scheduler-lab-worker2         2878m        28%      242Mi           3%          
scheduler-lab-worker3         2539m        25%      1680Mi          21% 
```
