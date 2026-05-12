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
- Changed the penality calculation to be based on projected resource utilization instead of effective resource utilization, to make it future-aware and prevent overloading nodes with the placement of new pods. Also introduced a penality factor to scale the penality based on how close the projected utilization is to the threshold, to prevent the penality from exceeding the score.

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
- Example:
  - Pods deleted at t=0.1s
  - Metrics collector last ran at t=0s  →  raw=25m, smoothed=2225m -> rawUtil=0.0025, smoothedUtil=0.2225
  - beta = |0.0025 - 0.2225| / (0.2225 + 0.01) = 0.220 / 0.2325 = 0.946
  - effectiveUtil = (1-0.946)*0.2225 + 0.946*0.0025 = 0.013 -> This is close to the real-time utilization, which is what we want during such diverging scenarios.
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
- Multiple experiments have been conducted with these perticular manifests.
```go
# CPU well-matched: request ~= actual usage (stresses 2 cores)
apiVersion: apps/v1
kind: Deployment
metadata:
  name: deploy-cpu-matched
spec:
  replicas: 3
  selector:
    matchLabels:
      app: cpu-matched
  template:
    metadata:
      labels:
        app: cpu-matched
    spec:
      schedulerName: kdapt-scheduler
      containers:
        - name: cpu-matched
          image: polinux/stress
          command: ["stress"]
          args: ["--cpu", "2", "--timeout", "36000"]
          resources:
            requests:
              cpu: "2000m"
              memory: "256Mi"
            limits:
              cpu: "2000m"
              memory: "256Mi"
          # Actual usage: ~2000m CPU, ~10Mi mem
```
```go
# --- Over-provisioned CPU: requested a lot, uses almost nothing ---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: deploy-over-prov-cpu
spec:
  replicas: 3
  selector:
    matchLabels:
      app: over-prov-cpu
  template:
    metadata:
      labels:
        app: over-prov-cpu
    spec:
      schedulerName: kdapt-scheduler
      containers:
        - name: over-prov-cpu
          image: nginx:latest
          ports:
            - containerPort: 80
          resources:
            requests:
              cpu: "3000m"
              memory: "512Mi"
            limits:
              cpu: "3000m"
              memory: "512Mi"
          # Actual usage: ~100m CPU, ~50Mi mem (nginx idle)
```
```go
# CPU under-provisioned: scheduler sees 500m, pod actually burns ~2000m
# No CPU limit so it can burst; scheduler will place based on 500m request only
apiVersion: apps/v1
kind: Deployment
metadata:
  name: deploy-cpu-hungry
spec:
  replicas: 3
  selector:
    matchLabels:
      app: cpu-hungry
  template:
    metadata:
      labels:
        app: cpu-hungry
    spec:
      schedulerName: kdapt-scheduler
      containers:
        - name: cpu-hungry
          image: polinux/stress
          command: ["stress"]
          args: ["--cpu", "2", "--timeout", "36000"]
          resources:
            requests:
              cpu: "500m"
              memory: "256Mi"
            # No limits — pod bursts beyond its request
          # Actual usage: ~2000m CPU, ~10Mi mem
```
```go
# Over-provisioned Memory: large memory request, barely uses it
apiVersion: apps/v1
kind: Deployment
metadata:
  name: deploy-over-prov-mem
spec:
  replicas: 3
  selector:
    matchLabels:
      app: over-prov-mem
  template:
    metadata:
      labels:
        app: over-prov-mem
    spec:
      schedulerName: kdapt-scheduler
      containers:
        - name: over-prov-mem
          image: nginx:latest
          ports:
            - containerPort: 80
          resources:
            requests:
              cpu: "500m"
              memory: "2Gi"
            limits:
              cpu: "500m"
              memory: "2Gi"
          # Actual usage: ~100m CPU, ~200Mi mem (nginx idle)
```
```go
# Memory under-provisioned: scheduler sees 512Mi, pod actually allocates ~1.5Gi
# No memory limit so it can burst; useful to stress node memory beyond scheduled view
apiVersion: apps/v1
kind: Deployment
metadata:
  name: deploy-mem-hungry
spec:
  replicas: 3
  selector:
    matchLabels:
      app: mem-hungry
  template:
    metadata:
      labels:
        app: mem-hungry
    spec:
      schedulerName: kdapt-scheduler
      containers:
        - name: mem-hungry
          image: polinux/stress
          command: ["stress"]
          args: ["--vm", "1", "--vm-bytes", "1500M", "--vm-keep", "--timeout", "36000"]
          resources:
            requests:
              cpu: "200m"
              memory: "512Mi"
            # No memory limit — pod bursts to 1.5Gi past its 512Mi request
          # Actual usage: ~200m CPU, ~1.5Gi mem
```
```go
# Memory well-matched: request ~= actual usage (allocates 900Mi)
apiVersion: apps/v1
kind: Deployment
metadata:
  name: deploy-mem-matched
spec:
  replicas: 3
  selector:
    matchLabels:
      app: mem-matched
  template:
    metadata:
      labels:
        app: mem-matched
    spec:
      schedulerName: kdapt-scheduler
      containers:
        - name: mem-matched
          image: polinux/stress
          command: ["stress"]
          args: ["--vm", "1", "--vm-bytes", "900M", "--vm-keep", "--timeout", "36000"]
          resources:
            requests:
              cpu: "200m"
              memory: "1Gi"
            limits:
              cpu: "200m"
              memory: "1Gi"
          # Actual usage: ~200m CPU, ~900Mi mem
```

###### Refer to `Experiments/staged-gradual-15.md` for the experiment details and results of the above manifests when they are deployed with a gradual staged arrival pattern.
- Take Aways:
  - Beta blending caught the spike. 
    - At wave 3, worker raw=6012m but smoothed=2176m.
    - Beta≈0.63 → effective=0.46 → rt logged as 0.46
    - Without beta, effective would have been 0.22, and worker would have looked far less loaded than it was.
  - _Algorithm still had a problem with introducing penality._
    - At wave 6, worker had reqCPU=0.98, and scheduling the perticular pod would take it to projectedCpu=1.00.
    - This should be avoided by the scheduler algorithm. 
    - This is because the penality is based on effectiveResourceUtil, which is a blend of smoothed and raw metrics.
    - It has to be based on projectedResourceUtil, which is the future view of the node's resource utilization after placing the pod, and not on the current view of the node's resource utilization.
- Penality can also be a continious function instead of a step function, which would provide a smoother penalty curve as the projected utilization approaches the threshold, rather than a hard cutoff. This would allow for more nuanced scheduling decisions, and prevent abrupt changes in scheduling behavior when the projected utilization crosses the threshold.
- To prevent penality from exceeding the score, we can introduce a penality factor that scales the penality based on how close the projected utilization is to the threshold.
- The penalty now means: "reduce this node's score by up to 100% depending on how overloaded it would become", rather than a fixed value subtracted blindly.
```go
	wCpu := 0.6
	wMem := 0.4
	weightedScore := wCpu*cpuScore + wMem*memScore

	// Penality for projected utilization above a threshold, to avoid scheduling on nodes that are likely to become overloaded.
	cpuPenality := 0.0
	if projectedCpuUtil > 0.8 {
		cpuPenality = (projectedCpuUtil - 0.8) / 0.2 // Linear penality from 0 to 1 as projectedCpuUtil goes from 0.8 to 1.0
	}
	memPenality := 0.0
	if projectedMemUtil > 0.8 {
		memPenality = (projectedMemUtil - 0.8) / 0.2 // Linear penality from 0 to 1 as projectedMemUtil goes from 0.8 to 1.0
		// CPU is slastic vs memory is conservative.
	}

	penalityFactor := clamp((1-wCpu)*cpuPenality+(1-wMem)*memPenality, 0, 1) // Overall penality factor based on CPU and memory penality, weighted by their importance in the score.
	// penality is higher for memory because memory pressure can lead to OOM kills, which is more disruptive than CPU contention in many cases.

	finalScore := clamp(weightedScore*(1-penalityFactor), 0, 1)
```

## Next Steps:
- RL/ML can be used to adjust the thresholds and weights dynamically.
- Read Borg design paper to understand the inspiration behind Kubernetes scheduling.
- Validate against kube-scheduler 
  - pod placement distribution by node
  - scheduling latency per pod
  - how tightly each scheduler packs before spilling to the next node
- Explore more complex scheduling scenarios with dynamic resource requests.
