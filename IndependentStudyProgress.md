## Hands-on progress till now:
- Setup KIND cluster for local Kubernetes development and testing.
- Forked Kubernetes repo and set up local development environment.
- Implemented basic Kdapt Scheduler plugin with scoring functionality. (returning static scores for testing)
- Genralised the plugin as per the Kubernetes scheduler plugin framework, and test it with static pod resource requests, using nodeAffinity to verify that the plugin is being invoked correctly during scheduling.
- Written a simple bin pack scoring algo based on Node resource utilization, combining CPU and memory utilization, giving more weight to CPU.
- Deployed pods with fixed pre-determined node requests with nodeAffinity to test the bin pack scoring algo. Static pods were deployed as per the expected bin packing strategy.

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

##### Bin-Packing powered with run-time metrics:
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
  - _This aligns with real systems like Borg and Kubernetes bin-packing strategies_
  - Safety penalties:
    - Prevents overloading nodes
    - Protects against resource crunch
    - Avoids greedy scheduling decisions
    - Provides safety guarantees

- In short, it is a hybrid between:
  - safe request-based bin-packing
  - adaptive runtime-sensitive placement

###### Experiment: CPU heavy burst scheduling with runtime-aware bin-packing
  - The workload is a burst of 8 pods, each requesting 1000m CPU, using kdapt-scheduler
  ```bash
  apiVersion: apps/v1
  kind: Deployment
  metadata:
    name: static-pod-deployment-cpu-heavy
  spec:
    replicas: 8
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
          - name: static-pod-container-cpu-heavy
            image: nginx:latest
            ports:
              - containerPort: 80
            resources:
              requests:
                cpu: "1000m"
  ```
  - Before burst:
    - All nodes are underutilized, and the cluster is balanced.
    - Each node has approximately 35 milli CPU utilization and 703205376.00 bytes (703.20 MB) of memory utilization.
  - During burst:
    - For each incoming pod, the scheduler evaluated all feasible worker nodes and logged:
      - current request-based CPU utilization on the node (reqCPU),
      - runtime CPU utilization from smoothed metrics (rtCPU),
      - mismatch between the two (mismatchCPU),
      - projected CPU utilization after placement (projectedCPU),
      - and the resulting final score.
    - pod=static-pod-deployment-cpu-heavy-7f6fd9659-gtsfl has scored:
      - node=scheduler-lab-worker
        - reqCPU=0.02
        - projectedCPU=0.12
        - finalScore=0.06596783636060116
      - node=scheduler-lab-worker2
        - reqCPU=0.01
        - projectedCPU=0.11
        - finalScore=0.053271573068199234
      - node=scheduler-lab-worker3
        - reqCPU=0.01
        - projectedCPU=0.11
        - finalScore=0.054177428323767325
      - Hence the pod will now be placed on node=scheduler-lab-worker
    - After that first placement, the same runtime snapshot was still being used for scoring the second pod.
    - pod=static-pod-deployment-cpu-heavy-7f6fd9659-nv7fn has scored:
      - node=scheduler-lab-worker
        - reqCPU=0.12
        - projectedCPU=0.22
        - finalScore=0.09675110036060115
      - node=scheduler-lab-worker2
        - reqCPU=0.01
        - projectedCPU=0.11
        - finalScore=0.053271573068199234
      - node=scheduler-lab-worker3
        - reqCPU=0.01
        - projectedCPU=0.11
        - finalScore=0.054177428323767325
      - Hence the pod will now be placed on node=scheduler-lab-worker
    ```bash
    I0405 20:03:04.709106       1 kdapt.go:80] -----------------------------------------------------------------
    I0405 20:03:14.697039       1 kdapt.go:71] nodeMetrics {name=scheduler-lab-control-plane, cpuMilli=166.00, memoryBytes=1452298240.00, smoothedCpuMilli=147.80, smoothedMemoryBytes=1450443161.60}
    I0405 20:03:14.697182       1 kdapt.go:71] nodeMetrics {name=scheduler-lab-worker, cpuMilli=36.00, memoryBytes=783876096.00, smoothedCpuMilli=29.70, smoothedMemoryBytes=783864627.20}
    I0405 20:03:14.697189       1 kdapt.go:71] nodeMetrics {name=scheduler-lab-worker2, cpuMilli=35.00, memoryBytes=703205376.00, smoothedCpuMilli=30.80, smoothedMemoryBytes=724666368.00}
    I0405 20:03:14.697193       1 kdapt.go:71] nodeMetrics {name=scheduler-lab-worker3, cpuMilli=38.00, memoryBytes=741777408.00, smoothedCpuMilli=73.70, smoothedMemoryBytes=740871372.80}
    I0405 20:03:14.697197       1 kdapt.go:80] -----------------------------------------------------------------
    I0405 20:03:24.697820       1 kdapt.go:71] nodeMetrics {name=scheduler-lab-control-plane, cpuMilli=166.00, memoryBytes=1452298240.00, smoothedCpuMilli=153.26, smoothedMemoryBytes=1450999685.12}
    I0405 20:03:24.697854       1 kdapt.go:71] nodeMetrics {name=scheduler-lab-worker, cpuMilli=36.00, memoryBytes=783876096.00, smoothedCpuMilli=31.59, smoothedMemoryBytes=783868067.84}
    I0405 20:03:24.697859       1 kdapt.go:71] nodeMetrics {name=scheduler-lab-worker2, cpuMilli=35.00, memoryBytes=703205376.00, smoothedCpuMilli=32.06, smoothedMemoryBytes=718228070.40}
    I0405 20:03:24.697862       1 kdapt.go:71] nodeMetrics {name=scheduler-lab-worker3, cpuMilli=38.00, memoryBytes=741777408.00, smoothedCpuMilli=62.99, smoothedMemoryBytes=741143183.36}
    I0405 20:03:24.697866       1 kdapt.go:80] -----------------------------------------------------------------
    I0405 20:03:31.685631       1 kdapt.go:219] pod=static-pod-deployment-cpu-heavy-7f6fd9659-gtsfl node=scheduler-lab-worker2 reqCPU=0.01 rtCPU=0.00 mismatchCPU=0.01 projectedCPU=0.11 final=0.05
    I0405 20:03:31.685654       1 kdapt.go:229] Final score: 0.053271573068199234
    I0405 20:03:31.685968       1 kdapt.go:219] pod=static-pod-deployment-cpu-heavy-7f6fd9659-gtsfl node=scheduler-lab-worker reqCPU=0.02 rtCPU=0.00 mismatchCPU=0.02 projectedCPU=0.12 final=0.07
    I0405 20:03:31.686008       1 kdapt.go:229] Final score: 0.06596783636060116
    I0405 20:03:31.686032       1 kdapt.go:219] pod=static-pod-deployment-cpu-heavy-7f6fd9659-gtsfl node=scheduler-lab-worker3 reqCPU=0.01 rtCPU=0.01 mismatchCPU=0.00 projectedCPU=0.11 final=0.05
    I0405 20:03:31.686092       1 kdapt.go:229] Final score: 0.054177428323767325
    I0405 20:03:31.687649       1 kdapt.go:219] pod=static-pod-deployment-cpu-heavy-7f6fd9659-nv7fn node=scheduler-lab-worker reqCPU=0.12 rtCPU=0.00 mismatchCPU=0.12 projectedCPU=0.22 final=0.10
    I0405 20:03:31.687664       1 kdapt.go:229] Final score: 0.09675110036060115
    I0405 20:03:31.687670       1 kdapt.go:219] pod=static-pod-deployment-cpu-heavy-7f6fd9659-nv7fn node=scheduler-lab-worker2 reqCPU=0.01 rtCPU=0.00 mismatchCPU=0.01 projectedCPU=0.11 final=0.05
    I0405 20:03:31.687673       1 kdapt.go:229] Final score: 0.053271573068199234
    I0405 20:03:31.687676       1 kdapt.go:219] pod=static-pod-deployment-cpu-heavy-7f6fd9659-nv7fn node=scheduler-lab-worker3 reqCPU=0.01 rtCPU=0.01 mismatchCPU=0.00 projectedCPU=0.11 final=0.05
    I0405 20:03:31.687678       1 kdapt.go:229] Final score: 0.054177428323767325
    I0405 20:03:31.690796       1 kdapt.go:219] pod=static-pod-deployment-cpu-heavy-7f6fd9659-w5rl4 node=scheduler-lab-worker reqCPU=0.22 rtCPU=0.00 mismatchCPU=0.22 projectedCPU=0.32 final=0.12
    I0405 20:03:31.690833       1 kdapt.go:229] Final score: 0.11793436436060116
    I0405 20:03:31.690841       1 kdapt.go:219] pod=static-pod-deployment-cpu-heavy-7f6fd9659-w5rl4 node=scheduler-lab-worker2 reqCPU=0.01 rtCPU=0.00 mismatchCPU=0.01 projectedCPU=0.11 final=0.05
    I0405 20:03:31.690846       1 kdapt.go:229] Final score: 0.053271573068199234
    I0405 20:03:31.690849       1 kdapt.go:219] pod=static-pod-deployment-cpu-heavy-7f6fd9659-w5rl4 node=scheduler-lab-worker3 reqCPU=0.01 rtCPU=0.01 mismatchCPU=0.00 projectedCPU=0.11 final=0.05
    I0405 20:03:31.690871       1 kdapt.go:229] Final score: 0.054177428323767325
    I0405 20:03:31.694940       1 kdapt.go:219] pod=static-pod-deployment-cpu-heavy-7f6fd9659-6snw7 node=scheduler-lab-worker reqCPU=0.32 rtCPU=0.00 mismatchCPU=0.32 projectedCPU=0.42 final=0.13
    I0405 20:03:31.694969       1 kdapt.go:229] Final score: 0.12951762836060116
    I0405 20:03:31.694987       1 kdapt.go:219] pod=static-pod-deployment-cpu-heavy-7f6fd9659-6snw7 node=scheduler-lab-worker3 reqCPU=0.01 rtCPU=0.01 mismatchCPU=0.00 projectedCPU=0.11 final=0.05
    I0405 20:03:31.694952       1 kdapt.go:219] pod=static-pod-deployment-cpu-heavy-7f6fd9659-6snw7 node=scheduler-lab-worker2 reqCPU=0.01 rtCPU=0.00 mismatchCPU=0.01 projectedCPU=0.11 final=0.05
    I0405 20:03:31.695001       1 kdapt.go:229] Final score: 0.054177428323767325
    I0405 20:03:31.695006       1 kdapt.go:229] Final score: 0.053271573068199234
    I0405 20:03:31.696087       1 kdapt.go:219] pod=static-pod-deployment-cpu-heavy-7f6fd9659-pfbnk node=scheduler-lab-worker reqCPU=0.42 rtCPU=0.00 mismatchCPU=0.42 projectedCPU=0.52 final=0.13
    I0405 20:03:31.696100       1 kdapt.go:229] Final score: 0.13150089236060117
    I0405 20:03:31.696104       1 kdapt.go:219] pod=static-pod-deployment-cpu-heavy-7f6fd9659-pfbnk node=scheduler-lab-worker2 reqCPU=0.01 rtCPU=0.00 mismatchCPU=0.01 projectedCPU=0.11 final=0.05
    I0405 20:03:31.696108       1 kdapt.go:229] Final score: 0.053271573068199234
    I0405 20:03:31.696117       1 kdapt.go:219] pod=static-pod-deployment-cpu-heavy-7f6fd9659-pfbnk node=scheduler-lab-worker3 reqCPU=0.01 rtCPU=0.01 mismatchCPU=0.00 projectedCPU=0.11 final=0.05
    I0405 20:03:31.696133       1 kdapt.go:229] Final score: 0.054177428323767325
    I0405 20:03:31.696694       1 kdapt.go:219] pod=static-pod-deployment-cpu-heavy-7f6fd9659-r4c6x node=scheduler-lab-worker reqCPU=0.52 rtCPU=0.00 mismatchCPU=0.52 projectedCPU=0.62 final=0.12
    I0405 20:03:31.696753       1 kdapt.go:229] Final score: 0.12388415636060116
    I0405 20:03:31.696761       1 kdapt.go:219] pod=static-pod-deployment-cpu-heavy-7f6fd9659-r4c6x node=scheduler-lab-worker2 reqCPU=0.01 rtCPU=0.00 mismatchCPU=0.01 projectedCPU=0.11 final=0.05
    I0405 20:03:31.696767       1 kdapt.go:229] Final score: 0.053271573068199234
    I0405 20:03:31.696772       1 kdapt.go:219] pod=static-pod-deployment-cpu-heavy-7f6fd9659-r4c6x node=scheduler-lab-worker3 reqCPU=0.01 rtCPU=0.01 mismatchCPU=0.00 projectedCPU=0.11 final=0.05
    I0405 20:03:31.696777       1 kdapt.go:229] Final score: 0.054177428323767325
    I0405 20:03:31.697427       1 kdapt.go:219] pod=static-pod-deployment-cpu-heavy-7f6fd9659-gkhql node=scheduler-lab-worker reqCPU=0.62 rtCPU=0.00 mismatchCPU=0.62 projectedCPU=0.72 final=0.11
    I0405 20:03:31.697458       1 kdapt.go:229] Final score: 0.10666742036060116
    I0405 20:03:31.697470       1 kdapt.go:219] pod=static-pod-deployment-cpu-heavy-7f6fd9659-gkhql node=scheduler-lab-worker2 reqCPU=0.01 rtCPU=0.00 mismatchCPU=0.01 projectedCPU=0.11 final=0.05
    I0405 20:03:31.697479       1 kdapt.go:229] Final score: 0.053271573068199234
    I0405 20:03:31.697489       1 kdapt.go:219] pod=static-pod-deployment-cpu-heavy-7f6fd9659-gkhql node=scheduler-lab-worker3 reqCPU=0.01 rtCPU=0.01 mismatchCPU=0.00 projectedCPU=0.11 final=0.05
    I0405 20:03:31.697498       1 kdapt.go:229] Final score: 0.054177428323767325
    I0405 20:03:31.704181       1 kdapt.go:219] pod=static-pod-deployment-cpu-heavy-7f6fd9659-tlg45 node=scheduler-lab-worker reqCPU=0.72 rtCPU=0.00 mismatchCPU=0.72 projectedCPU=0.82 final=0.08
    I0405 20:03:31.704223       1 kdapt.go:229] Final score: 0.07985068436060119
    I0405 20:03:31.704223       1 kdapt.go:219] pod=static-pod-deployment-cpu-heavy-7f6fd9659-tlg45 node=scheduler-lab-worker2 reqCPU=0.01 rtCPU=0.00 mismatchCPU=0.01 projectedCPU=0.11 final=0.05
    I0405 20:03:31.704233       1 kdapt.go:219] pod=static-pod-deployment-cpu-heavy-7f6fd9659-tlg45 node=scheduler-lab-worker3 reqCPU=0.01 rtCPU=0.01 mismatchCPU=0.00 projectedCPU=0.11 final=0.05
    I0405 20:03:31.704236       1 kdapt.go:229] Final score: 0.053271573068199234
    I0405 20:03:31.704248       1 kdapt.go:229] Final score: 0.054177428323767325
    I0405 20:03:34.694968       1 kdapt.go:71] nodeMetrics {name=scheduler-lab-control-plane, cpuMilli=145.00, memoryBytes=1451958272.00, smoothedCpuMilli=150.78, smoothedMemoryBytes=1451287261.18}
    I0405 20:03:34.695012       1 kdapt.go:71] nodeMetrics {name=scheduler-lab-worker, cpuMilli=28.00, memoryBytes=783613952.00, smoothedCpuMilli=30.51, smoothedMemoryBytes=783791833.09}
    I0405 20:03:34.695018       1 kdapt.go:71] nodeMetrics {name=scheduler-lab-worker2, cpuMilli=19.00, memoryBytes=702443520.00, smoothedCpuMilli=28.14, smoothedMemoryBytes=713492705.28}
    I0405 20:03:34.695021       1 kdapt.go:71] nodeMetrics {name=scheduler-lab-worker3, cpuMilli=26.00, memoryBytes=742375424.00, smoothedCpuMilli=51.89, smoothedMemoryBytes=741512855.55}
    I0405 20:03:34.695023       1 kdapt.go:80] -----------------------------------------------------------------
    I0405 20:03:44.699764       1 kdapt.go:71] nodeMetrics {name=scheduler-lab-control-plane, cpuMilli=160.00, memoryBytes=1451147264.00, smoothedCpuMilli=153.55, smoothedMemoryBytes=1451245262.03}
    I0405 20:03:44.699798       1 kdapt.go:71] nodeMetrics {name=scheduler-lab-worker, cpuMilli=391.00, memoryBytes=903962624.00, smoothedCpuMilli=138.66, smoothedMemoryBytes=819843070.36}
    I0405 20:03:44.699803       1 kdapt.go:71] nodeMetrics {name=scheduler-lab-worker2, cpuMilli=19.00, memoryBytes=702193664.00, smoothedCpuMilli=25.40, smoothedMemoryBytes=710102992.90}
    I0405 20:03:44.699807       1 kdapt.go:71] nodeMetrics {name=scheduler-lab-worker3, cpuMilli=29.00, memoryBytes=742871040.00, smoothedCpuMilli=45.03, smoothedMemoryBytes=741920310.89}
    I0405 20:03:44.699810       1 kdapt.go:80] -----------------------------------------------------------------
    ```
    - It is to be observed that for the second pod, node=scheduler-lab-worker's requested CPU utilization is 0.12, which was taken into consideration while score the pod before using projectedCPU metric. 
    - This consistency between the previous step's projectedCPU and the next step's requestedCPU metrics proves that the projectedCPU metric is trust worthy to impose a penality when the node whould be stressed by the placement.
    - It can be observed that for each new scoring cycle, requestedCPU on node=scheduler-lab-worker is gradually increasing as all previous pods are scheduled on this node. (0.12 → 0.22 → 0.32 → 0.42 → 0.52 → ...)
    - All pods in the burst are scheduled on node=scheduler-lab-worker, because the node's runtime utilization never exceeds 0.82 and as per bin-packing this node would be the best fit for the pods based on requested recourses too.
```bash
k8 get pods -o wide                                  
NAME                                              READY   STATUS    RESTARTS   AGE   IP            NODE                   NOMINATED NODE   READINESS GATES
static-pod-deployment-cpu-heavy-7f6fd9659-6snw7   1/1     Running   0          56s   10.244.2.34   scheduler-lab-worker   <none>           <none>
static-pod-deployment-cpu-heavy-7f6fd9659-gkhql   1/1     Running   0          56s   10.244.2.37   scheduler-lab-worker   <none>           <none>
static-pod-deployment-cpu-heavy-7f6fd9659-gtsfl   1/1     Running   0          56s   10.244.2.31   scheduler-lab-worker   <none>           <none>
static-pod-deployment-cpu-heavy-7f6fd9659-nv7fn   1/1     Running   0          56s   10.244.2.33   scheduler-lab-worker   <none>           <none>
static-pod-deployment-cpu-heavy-7f6fd9659-pfbnk   1/1     Running   0          56s   10.244.2.35   scheduler-lab-worker   <none>           <none>
static-pod-deployment-cpu-heavy-7f6fd9659-r4c6x   1/1     Running   0          56s   10.244.2.36   scheduler-lab-worker   <none>           <none>
static-pod-deployment-cpu-heavy-7f6fd9659-tlg45   1/1     Running   0          56s   10.244.2.38   scheduler-lab-worker   <none>           <none>
static-pod-deployment-cpu-heavy-7f6fd9659-w5rl4   1/1     Running   0          56s   10.244.2.32   scheduler-lab-worker   <none>           <none>
```
```bash
 k8 top nodes
NAME                          CPU(cores)   CPU(%)   MEMORY(bytes)   MEMORY(%)
scheduler-lab-control-plane   160m         1%       1163Mi          14%
scheduler-lab-worker          142m         1%       825Mi           10%
scheduler-lab-worker2         30m          0%       601Mi           7%
scheduler-lab-worker3         40m          0%       644Mi           8%
```
  - The score for other nodes along with their requestedCPU utilization remained the same throughtout the experiment - (0.053~0.054) - because no pod was scheduled on them at any point of time.
  - Below is the output when the burst is scheduled using default kube-scheduler with default bin-packing strategy:
```bash
k8 get pods -o wide
NAME                                              READY   STATUS    RESTARTS   AGE    IP            NODE                    NOMINATED NODE   READINESS GATES
static-pod-deployment-cpu-heavy-768b5fbdd-7c9w5   1/1     Running   0          118s   10.244.1.3    scheduler-lab-worker2   <none>           <none>
static-pod-deployment-cpu-heavy-768b5fbdd-bn9qq   1/1     Running   0          118s   10.244.3.3    scheduler-lab-worker3   <none>           <none>
static-pod-deployment-cpu-heavy-768b5fbdd-dscms   1/1     Running   0          118s   10.244.1.4    scheduler-lab-worker2   <none>           <none>
static-pod-deployment-cpu-heavy-768b5fbdd-dxrr2   1/1     Running   0          118s   10.244.2.13   scheduler-lab-worker    <none>           <none>
static-pod-deployment-cpu-heavy-768b5fbdd-gq6f6   1/1     Running   0          118s   10.244.1.2    scheduler-lab-worker2   <none>           <none>
static-pod-deployment-cpu-heavy-768b5fbdd-rh7q7   1/1     Running   0          118s   10.244.2.12   scheduler-lab-worker    <none>           <none>
static-pod-deployment-cpu-heavy-768b5fbdd-xgn6m   1/1     Running   0          118s   10.244.2.11   scheduler-lab-worker    <none>           <none>
static-pod-deployment-cpu-heavy-768b5fbdd-z69jz   1/1     Running   0          118s   10.244.3.4    scheduler-lab-worker3   <none>           <none>
```
```bash
k8 top nodes
NAME                          CPU(cores)   CPU(%)   MEMORY(bytes)   MEMORY(%)
scheduler-lab-control-plane   143m         1%       1160Mi          14%
scheduler-lab-worker          39m          0%       751Mi           9%
scheduler-lab-worker2         36m          0%       650Mi           8%
scheduler-lab-worker3         30m          0%       673Mi           8%
```
  - We can also observe that the utilization of worker node `scheduler-lab-worker` has increased while keeping the other nodes as is using kdapt-scheduler, while the default kube-scheduler has distributed the pods across all worker nodes. Using Kdapt-Scheduler can lead to better resource garbage collection and efficiency.
  - Take aways:
    - The scheduler successfully used a hybrid score combining requests, runtime signals, mismatch weighting, and projected utilization.
    - It behaved deterministically and consistently across all eight scheduling decisions.
    - Runtime metrics stayed almost constant, while request-based projected utilization changed only on a worker node as expected. (As the pods were not compute heavy)
    - Worker node=scheduler-lab-worker began with the highest score, and the score increased as the requestedCPU increased (0.06 → 0.09 → 0.11 → 0.13 → 0.13 → ...) within marginal utility.
    - Then the score decreased as node fills up, but not enough to overturn its lead
    - We now have a adaptive runtime-aware bin-packing scheduler that intentionally prefers consolidation (packing) while applying safety penalties to prevent overload.

###### Experiment: Memory heavy burst scheduling with runtime-aware bin-packing
  - The workload is a burst of 8 pods, each requesting 1Gi memory, using kdapt-scheduler
  - Since memory is treated more conservatively in the scoring algorithm, it is expected that the scheduler will prefer spreading the pods across nodes rather than packing them on a single node, to avoid memory pressure and potential OOM kills.
```bash
apiVersion: apps/v1
kind: Deployment
metadata:
  name: static-pod-deployment-memory-heavy
spec:
  replicas: 8
  selector:
    matchLabels:
      app: static-pod-memory-heavy
  template:
    metadata:
      labels:
        app: static-pod-memory-heavy
    spec:
      schedulerName: kdapt-scheduler
      containers:
        - name: static-pod-container-memory-heavy
          image: nginx:latest
          ports:
            - containerPort: 80
          resources:
            requests:
              memory: "1Gi"
```
  - Before burst:
    - All nodes are underutilized, and the cluster is balanced.
    - Each node has approximately 30 milli CPU utilization and 703205376.00 bytes (703.20 MB) of memory utilization.
  - During burst:
```bash
I0506 15:39:17.704253       1 kdapt.go:71] nodeMetrics {name=scheduler-lab-control-plane, cpuMilli=139.00, memoryBytes=1218187264.00, smoothedCpuMilli=134.48, smoothedMemoryBytes=1218706660.15}
I0506 15:39:17.704299       1 kdapt.go:71] nodeMetrics {name=scheduler-lab-worker, cpuMilli=27.00, memoryBytes=742604800.00, smoothedCpuMilli=64.93, smoothedMemoryBytes=776661665.38}
I0506 15:39:17.704305       1 kdapt.go:71] nodeMetrics {name=scheduler-lab-worker2, cpuMilli=24.00, memoryBytes=644603904.00, smoothedCpuMilli=38.98, smoothedMemoryBytes=649462210.97}
I0506 15:39:17.704308       1 kdapt.go:71] nodeMetrics {name=scheduler-lab-worker3, cpuMilli=25.00, memoryBytes=703262720.00, smoothedCpuMilli=153.65, smoothedMemoryBytes=707540840.45}
I0506 15:39:17.704312       1 kdapt.go:80] -----------------------------------------------------------------
I0506 15:39:27.704321       1 kdapt.go:71] nodeMetrics {name=scheduler-lab-control-plane, cpuMilli=110.00, memoryBytes=1219473408.00, smoothedCpuMilli=127.14, smoothedMemoryBytes=1218936684.50}
I0506 15:39:27.704423       1 kdapt.go:71] nodeMetrics {name=scheduler-lab-worker, cpuMilli=52.00, memoryBytes=742907904.00, smoothedCpuMilli=61.05, smoothedMemoryBytes=766535536.97}
I0506 15:39:27.704433       1 kdapt.go:71] nodeMetrics {name=scheduler-lab-worker2, cpuMilli=19.00, memoryBytes=644882432.00, smoothedCpuMilli=32.99, smoothedMemoryBytes=648088277.28}
I0506 15:39:27.704437       1 kdapt.go:71] nodeMetrics {name=scheduler-lab-worker3, cpuMilli=25.00, memoryBytes=703401984.00, smoothedCpuMilli=115.06, smoothedMemoryBytes=706299183.51}
I0506 15:39:27.704440       1 kdapt.go:80] -----------------------------------------------------------------
I0506 15:39:29.150815       1 kdapt.go:219] pod=static-pod-deployment-mem-heavy-7cbc98d99b-bpkch node=scheduler-lab-worker2 reqCPU=0.01 rtCPU=0.00 mismatchCPU=0.01 projectedCPU=0.01 final=0.06
I0506 15:39:29.150859       1 kdapt.go:229] pod=static-pod-deployment-mem-heavy-7cbc98d99b-bpkch node=scheduler-lab-worker2 reqMem=0.01 rtMem=0.08 mismatchMem=0.07 projectedMem=0.14 final=0.06
I0506 15:39:29.150863       1 kdapt.go:239] Final score: 0.05658515814615161
I0506 15:39:29.150868       1 kdapt.go:219] pod=static-pod-deployment-mem-heavy-7cbc98d99b-bpkch node=scheduler-lab-worker3 reqCPU=0.01 rtCPU=0.01 mismatchCPU=0.00 projectedCPU=0.01 final=0.06
I0506 15:39:29.150873       1 kdapt.go:229] pod=static-pod-deployment-mem-heavy-7cbc98d99b-bpkch node=scheduler-lab-worker3 reqMem=0.01 rtMem=0.09 mismatchMem=0.08 projectedMem=0.14 final=0.06
I0506 15:39:29.150875       1 kdapt.go:239] Final score: 0.05839262136829619
I0506 15:39:29.150877       1 kdapt.go:219] pod=static-pod-deployment-mem-heavy-7cbc98d99b-bpkch node=scheduler-lab-worker reqCPU=0.02 rtCPU=0.01 mismatchCPU=0.01 projectedCPU=0.02 final=0.07
I0506 15:39:29.150880       1 kdapt.go:229] pod=static-pod-deployment-mem-heavy-7cbc98d99b-bpkch node=scheduler-lab-worker reqMem=0.03 rtMem=0.09 mismatchMem=0.06 projectedMem=0.16 final=0.07
I0506 15:39:29.150884       1 kdapt.go:239] Final score: 0.07097680685531889
I0506 15:39:29.154428       1 kdapt.go:219] pod=static-pod-deployment-mem-heavy-7cbc98d99b-68wqd node=scheduler-lab-worker2 reqCPU=0.01 rtCPU=0.00 mismatchCPU=0.01 projectedCPU=0.01 final=0.06
I0506 15:39:29.154459       1 kdapt.go:229] pod=static-pod-deployment-mem-heavy-7cbc98d99b-68wqd node=scheduler-lab-worker2 reqMem=0.01 rtMem=0.08 mismatchMem=0.07 projectedMem=0.14 final=0.06
I0506 15:39:29.154465       1 kdapt.go:239] Final score: 0.05658515814615161
I0506 15:39:29.154470       1 kdapt.go:219] pod=static-pod-deployment-mem-heavy-7cbc98d99b-68wqd node=scheduler-lab-worker3 reqCPU=0.01 rtCPU=0.01 mismatchCPU=0.00 projectedCPU=0.01 final=0.06
I0506 15:39:29.154473       1 kdapt.go:229] pod=static-pod-deployment-mem-heavy-7cbc98d99b-68wqd node=scheduler-lab-worker3 reqMem=0.01 rtMem=0.09 mismatchMem=0.08 projectedMem=0.14 final=0.06
I0506 15:39:29.154476       1 kdapt.go:239] Final score: 0.05839262136829619
I0506 15:39:29.154478       1 kdapt.go:219] pod=static-pod-deployment-mem-heavy-7cbc98d99b-68wqd node=scheduler-lab-worker reqCPU=0.02 rtCPU=0.01 mismatchCPU=0.01 projectedCPU=0.02 final=0.12
I0506 15:39:29.154481       1 kdapt.go:229] pod=static-pod-deployment-mem-heavy-7cbc98d99b-68wqd node=scheduler-lab-worker reqMem=0.16 rtMem=0.09 mismatchMem=0.07 projectedMem=0.29 final=0.12
I0506 15:39:29.154483       1 kdapt.go:239] Final score: 0.116477308840957
I0506 15:39:29.154787       1 kdapt.go:219] pod=static-pod-deployment-mem-heavy-7cbc98d99b-zllzp node=scheduler-lab-worker2 reqCPU=0.01 rtCPU=0.00 mismatchCPU=0.01 projectedCPU=0.01 final=0.06
I0506 15:39:29.154798       1 kdapt.go:229] pod=static-pod-deployment-mem-heavy-7cbc98d99b-zllzp node=scheduler-lab-worker2 reqMem=0.01 rtMem=0.08 mismatchMem=0.07 projectedMem=0.14 final=0.06
I0506 15:39:29.154801       1 kdapt.go:239] Final score: 0.05658515814615161
I0506 15:39:29.154804       1 kdapt.go:219] pod=static-pod-deployment-mem-heavy-7cbc98d99b-zllzp node=scheduler-lab-worker3 reqCPU=0.01 rtCPU=0.01 mismatchCPU=0.00 projectedCPU=0.01 final=0.06
I0506 15:39:29.154807       1 kdapt.go:229] pod=static-pod-deployment-mem-heavy-7cbc98d99b-zllzp node=scheduler-lab-worker3 reqMem=0.01 rtMem=0.09 mismatchMem=0.08 projectedMem=0.14 final=0.06
I0506 15:39:29.154809       1 kdapt.go:239] Final score: 0.05839262136829619
I0506 15:39:29.154811       1 kdapt.go:219] pod=static-pod-deployment-mem-heavy-7cbc98d99b-zllzp node=scheduler-lab-worker reqCPU=0.02 rtCPU=0.01 mismatchCPU=0.01 projectedCPU=0.02 final=0.16
I0506 15:39:29.154814       1 kdapt.go:229] pod=static-pod-deployment-mem-heavy-7cbc98d99b-zllzp node=scheduler-lab-worker reqMem=0.29 rtMem=0.09 mismatchMem=0.20 projectedMem=0.42 final=0.16
I0506 15:39:29.154817       1 kdapt.go:239] Final score: 0.15515433229941455
I0506 15:39:29.161105       1 kdapt.go:219] pod=static-pod-deployment-mem-heavy-7cbc98d99b-5lrhs node=scheduler-lab-worker2 reqCPU=0.01 rtCPU=0.00 mismatchCPU=0.01 projectedCPU=0.01 final=0.06
I0506 15:39:29.161127       1 kdapt.go:229] pod=static-pod-deployment-mem-heavy-7cbc98d99b-5lrhs node=scheduler-lab-worker2 reqMem=0.01 rtMem=0.08 mismatchMem=0.07 projectedMem=0.14 final=0.06
I0506 15:39:29.161132       1 kdapt.go:239] Final score: 0.05658515814615161
I0506 15:39:29.161137       1 kdapt.go:219] pod=static-pod-deployment-mem-heavy-7cbc98d99b-5lrhs node=scheduler-lab-worker reqCPU=0.02 rtCPU=0.01 mismatchCPU=0.01 projectedCPU=0.02 final=0.19
I0506 15:39:29.161139       1 kdapt.go:229] pod=static-pod-deployment-mem-heavy-7cbc98d99b-5lrhs node=scheduler-lab-worker reqMem=0.42 rtMem=0.09 mismatchMem=0.33 projectedMem=0.55 final=0.19
I0506 15:39:29.161145       1 kdapt.go:239] Final score: 0.18836857423074574
I0506 15:39:29.161148       1 kdapt.go:219] pod=static-pod-deployment-mem-heavy-7cbc98d99b-5lrhs node=scheduler-lab-worker3 reqCPU=0.01 rtCPU=0.01 mismatchCPU=0.00 projectedCPU=0.01 final=0.06
I0506 15:39:29.161151       1 kdapt.go:229] pod=static-pod-deployment-mem-heavy-7cbc98d99b-5lrhs node=scheduler-lab-worker3 reqMem=0.01 rtMem=0.09 mismatchMem=0.08 projectedMem=0.14 final=0.06
I0506 15:39:29.161154       1 kdapt.go:239] Final score: 0.05839262136829619
I0506 15:39:29.162010       1 kdapt.go:219] pod=static-pod-deployment-mem-heavy-7cbc98d99b-xb6sr node=scheduler-lab-worker2 reqCPU=0.01 rtCPU=0.00 mismatchCPU=0.01 projectedCPU=0.01 final=0.06
I0506 15:39:29.162041       1 kdapt.go:229] pod=static-pod-deployment-mem-heavy-7cbc98d99b-xb6sr node=scheduler-lab-worker2 reqMem=0.01 rtMem=0.08 mismatchMem=0.07 projectedMem=0.14 final=0.06
I0506 15:39:29.162052       1 kdapt.go:239] Final score: 0.05658515814615161
I0506 15:39:29.162064       1 kdapt.go:219] pod=static-pod-deployment-mem-heavy-7cbc98d99b-xb6sr node=scheduler-lab-worker3 reqCPU=0.01 rtCPU=0.01 mismatchCPU=0.00 projectedCPU=0.01 final=0.06
I0506 15:39:29.162075       1 kdapt.go:229] pod=static-pod-deployment-mem-heavy-7cbc98d99b-xb6sr node=scheduler-lab-worker3 reqMem=0.01 rtMem=0.09 mismatchMem=0.08 projectedMem=0.14 final=0.06
I0506 15:39:29.162085       1 kdapt.go:239] Final score: 0.05839262136829619
I0506 15:39:29.162095       1 kdapt.go:219] pod=static-pod-deployment-mem-heavy-7cbc98d99b-xb6sr node=scheduler-lab-worker reqCPU=0.02 rtCPU=0.01 mismatchCPU=0.01 projectedCPU=0.02 final=0.22
I0506 15:39:29.162104       1 kdapt.go:229] pod=static-pod-deployment-mem-heavy-7cbc98d99b-xb6sr node=scheduler-lab-worker reqMem=0.55 rtMem=0.09 mismatchMem=0.46 projectedMem=0.69 final=0.22
I0506 15:39:29.162114       1 kdapt.go:239] Final score: 0.21612003463495053
I0506 15:39:29.164915       1 kdapt.go:219] pod=static-pod-deployment-mem-heavy-7cbc98d99b-fw6vd node=scheduler-lab-worker2 reqCPU=0.01 rtCPU=0.00 mismatchCPU=0.01 projectedCPU=0.01 final=0.06
I0506 15:39:29.164932       1 kdapt.go:229] pod=static-pod-deployment-mem-heavy-7cbc98d99b-fw6vd node=scheduler-lab-worker2 reqMem=0.01 rtMem=0.08 mismatchMem=0.07 projectedMem=0.14 final=0.06
I0506 15:39:29.164937       1 kdapt.go:239] Final score: 0.05658515814615161
I0506 15:39:29.164943       1 kdapt.go:219] pod=static-pod-deployment-mem-heavy-7cbc98d99b-fw6vd node=scheduler-lab-worker3 reqCPU=0.01 rtCPU=0.01 mismatchCPU=0.00 projectedCPU=0.01 final=0.06
I0506 15:39:29.164946       1 kdapt.go:229] pod=static-pod-deployment-mem-heavy-7cbc98d99b-fw6vd node=scheduler-lab-worker3 reqMem=0.01 rtMem=0.09 mismatchMem=0.08 projectedMem=0.14 final=0.06
I0506 15:39:29.164949       1 kdapt.go:239] Final score: 0.05839262136829619
I0506 15:39:29.164951       1 kdapt.go:219] pod=static-pod-deployment-mem-heavy-7cbc98d99b-fw6vd node=scheduler-lab-worker reqCPU=0.02 rtCPU=0.01 mismatchCPU=0.01 projectedCPU=0.02 final=0.24
I0506 15:39:29.164954       1 kdapt.go:229] pod=static-pod-deployment-mem-heavy-7cbc98d99b-fw6vd node=scheduler-lab-worker reqMem=0.69 rtMem=0.09 mismatchMem=0.59 projectedMem=0.82 final=0.24
I0506 15:39:29.164957       1 kdapt.go:239] Final score: 0.23840871351202889
I0506 15:39:29.165259       1 kdapt.go:219] pod=static-pod-deployment-mem-heavy-7cbc98d99b-6855n node=scheduler-lab-worker2 reqCPU=0.01 rtCPU=0.00 mismatchCPU=0.01 projectedCPU=0.01 final=0.06
I0506 15:39:29.165269       1 kdapt.go:229] pod=static-pod-deployment-mem-heavy-7cbc98d99b-6855n node=scheduler-lab-worker2 reqMem=0.01 rtMem=0.08 mismatchMem=0.07 projectedMem=0.14 final=0.06
I0506 15:39:29.165273       1 kdapt.go:239] Final score: 0.05658515814615161
I0506 15:39:29.165276       1 kdapt.go:219] pod=static-pod-deployment-mem-heavy-7cbc98d99b-6855n node=scheduler-lab-worker3 reqCPU=0.01 rtCPU=0.01 mismatchCPU=0.00 projectedCPU=0.01 final=0.06
I0506 15:39:29.165279       1 kdapt.go:229] pod=static-pod-deployment-mem-heavy-7cbc98d99b-6855n node=scheduler-lab-worker3 reqMem=0.01 rtMem=0.09 mismatchMem=0.08 projectedMem=0.14 final=0.06
I0506 15:39:29.165282       1 kdapt.go:239] Final score: 0.05839262136829619
I0506 15:39:29.165284       1 kdapt.go:219] pod=static-pod-deployment-mem-heavy-7cbc98d99b-6855n node=scheduler-lab-worker reqCPU=0.02 rtCPU=0.01 mismatchCPU=0.01 projectedCPU=0.02 final=0.26
I0506 15:39:29.165286       1 kdapt.go:229] pod=static-pod-deployment-mem-heavy-7cbc98d99b-6855n node=scheduler-lab-worker reqMem=0.82 rtMem=0.09 mismatchMem=0.72 projectedMem=0.95 final=0.26
I0506 15:39:29.165290       1 kdapt.go:239] Final score: 0.25523461086198085
I0506 15:39:29.172585       1 kdapt.go:219] pod=static-pod-deployment-mem-heavy-7cbc98d99b-pnjgh node=scheduler-lab-worker2 reqCPU=0.01 rtCPU=0.00 mismatchCPU=0.01 projectedCPU=0.01 final=0.06
I0506 15:39:29.172617       1 kdapt.go:229] pod=static-pod-deployment-mem-heavy-7cbc98d99b-pnjgh node=scheduler-lab-worker2 reqMem=0.01 rtMem=0.08 mismatchMem=0.07 projectedMem=0.14 final=0.06
I0506 15:39:29.172627       1 kdapt.go:239] Final score: 0.05658515814615161
I0506 15:39:29.172643       1 kdapt.go:219] pod=static-pod-deployment-mem-heavy-7cbc98d99b-pnjgh node=scheduler-lab-worker3 reqCPU=0.01 rtCPU=0.01 mismatchCPU=0.00 projectedCPU=0.01 final=0.06
I0506 15:39:29.172659       1 kdapt.go:229] pod=static-pod-deployment-mem-heavy-7cbc98d99b-pnjgh node=scheduler-lab-worker3 reqMem=0.01 rtMem=0.09 mismatchMem=0.08 projectedMem=0.14 final=0.06
I0506 15:39:29.172672       1 kdapt.go:239] Final score: 0.05839262136829619
I0506 15:39:37.704894       1 kdapt.go:71] nodeMetrics {name=scheduler-lab-control-plane, cpuMilli=110.00, memoryBytes=1219473408.00, smoothedCpuMilli=122.00, smoothedMemoryBytes=1219097701.55}
I0506 15:39:37.704936       1 kdapt.go:71] nodeMetrics {name=scheduler-lab-worker, cpuMilli=52.00, memoryBytes=742907904.00, smoothedCpuMilli=58.33, smoothedMemoryBytes=759447247.08}
I0506 15:39:37.704942       1 kdapt.go:71] nodeMetrics {name=scheduler-lab-worker2, cpuMilli=19.00, memoryBytes=644882432.00, smoothedCpuMilli=28.79, smoothedMemoryBytes=647126523.70}
I0506 15:39:37.704946       1 kdapt.go:71] nodeMetrics {name=scheduler-lab-worker3, cpuMilli=25.00, memoryBytes=703401984.00, smoothedCpuMilli=88.04, smoothedMemoryBytes=705430023.66}
I0506 15:39:37.704950       1 kdapt.go:80] -----------------------------------------------------------------
I0506 15:39:47.702454       1 kdapt.go:71] nodeMetrics {name=scheduler-lab-control-plane, cpuMilli=140.00, memoryBytes=1220947968.00, smoothedCpuMilli=127.40, smoothedMemoryBytes=1219652781.49}
I0506 15:39:47.702508       1 kdapt.go:71] nodeMetrics {name=scheduler-lab-worker, cpuMilli=126.00, memoryBytes=857112576.00, smoothedCpuMilli=78.63, smoothedMemoryBytes=788746845.75}
I0506 15:39:47.702514       1 kdapt.go:71] nodeMetrics {name=scheduler-lab-worker2, cpuMilli=61.00, memoryBytes=659501056.00, smoothedCpuMilli=38.45, smoothedMemoryBytes=650838883.39}
I0506 15:39:47.702518       1 kdapt.go:71] nodeMetrics {name=scheduler-lab-worker3, cpuMilli=28.00, memoryBytes=703389696.00, smoothedCpuMilli=70.03, smoothedMemoryBytes=704817925.36}
```
  - It can be observed that the node `scheduler-lab-worker` starts off with a higher memory utilization, and the bin-packing algo takes requested and runtime metrics into consideration to come-up with a final score of 0.071 for this node.
  - The other two worker nodes have much lower scores of 0.06, which is expected as they have lower memory utilization and the scheduler is designed to prefer packing when possible.
  - As the burst continues and more pods are scheduled on `scheduler-lab-worker`, its score increases to 0.12, 0.16, 0.19, and eventually 0.26.
```bash
ProviderID:                   kind://docker/scheduler-lab/scheduler-lab-worker
Non-terminated Pods:          (10 in total)
  Namespace                   Name                                                CPU Requests  CPU Limits  Memory Requests  Memory Limits  Age
  ---------                   ----                                                ------------  ----------  ---------------  -------------  ---
  default                     static-pod-deployment-mem-heavy-7cbc98d99b-5lrhs    0 (0%)        0 (0%)      1Gi (13%)        0 (0%)         68m
  default                     static-pod-deployment-mem-heavy-7cbc98d99b-6855n    0 (0%)        0 (0%)      1Gi (13%)        0 (0%)         68m
  default                     static-pod-deployment-mem-heavy-7cbc98d99b-68wqd    0 (0%)        0 (0%)      1Gi (13%)        0 (0%)         68m
  default                     static-pod-deployment-mem-heavy-7cbc98d99b-bpkch    0 (0%)        0 (0%)      1Gi (13%)        0 (0%)         68m
  default                     static-pod-deployment-mem-heavy-7cbc98d99b-fw6vd    0 (0%)        0 (0%)      1Gi (13%)        0 (0%)         68m
  default                     static-pod-deployment-mem-heavy-7cbc98d99b-xb6sr    0 (0%)        0 (0%)      1Gi (13%)        0 (0%)         68m
  default                     static-pod-deployment-mem-heavy-7cbc98d99b-zllzp    0 (0%)        0 (0%)      1Gi (13%)        0 (0%)         68m
  kube-system                 kindnet-5zkmh                                       100m (1%)     100m (1%)   50Mi (0%)        50Mi (0%)      67d
  kube-system                 kube-proxy-xmjgx                                    0 (0%)        0 (0%)      0 (0%)           0 (0%)         67d
  kube-system                 metrics-server-5f54fb74d9-wp528                     100m (1%)     0 (0%)      200Mi (2%)       0 (0%)         60d
```
  - Almost 7Gb is requested by the pods on node `scheduler-lab-worker`, which is close to the node's capacity of 8Gi.
  - Then the node is out of memory and is filtered before the score phase during scheduling the last pod. Hence we see only two nodes' scores for this perticular `pod=static-pod-deployment-mem-heavy-7cbc98d99b-pnjgh`, and the pod is scheduled on `scheduler-lab-worker2`.
```bash
k8 get pods -o wide        
NAME                                               READY   STATUS    RESTARTS      AGE   IP           NODE                    NOMINATED NODE   READINESS GATES
static-pod-deployment-mem-heavy-7cbc98d99b-5lrhs   1/1     Running   1 (22m ago)   69m   10.244.2.5   scheduler-lab-worker    <none>           <none>
static-pod-deployment-mem-heavy-7cbc98d99b-6855n   1/1     Running   1 (22m ago)   69m   10.244.2.2   scheduler-lab-worker    <none>           <none>
static-pod-deployment-mem-heavy-7cbc98d99b-68wqd   1/1     Running   1 (22m ago)   69m   10.244.2.3   scheduler-lab-worker    <none>           <none>
static-pod-deployment-mem-heavy-7cbc98d99b-bpkch   1/1     Running   1 (22m ago)   69m   10.244.2.4   scheduler-lab-worker    <none>           <none>
static-pod-deployment-mem-heavy-7cbc98d99b-fw6vd   1/1     Running   1 (22m ago)   69m   10.244.2.6   scheduler-lab-worker    <none>           <none>
static-pod-deployment-mem-heavy-7cbc98d99b-pnjgh   1/1     Running   1 (22m ago)   69m   10.244.1.2   scheduler-lab-worker2   <none>           <none>
static-pod-deployment-mem-heavy-7cbc98d99b-xb6sr   1/1     Running   1 (22m ago)   69m   10.244.2.8   scheduler-lab-worker    <none>           <none>
static-pod-deployment-mem-heavy-7cbc98d99b-zllzp   1/1     Running   1 (22m ago)   69m   10.244.2.9   scheduler-lab-worker    <none>           <none>
```
  - Compared to native Kubernetes Scheduler, the custom kdapt-scheduler is able to pack more pods on the same node until it reaches the memory limit, within the constraints of the scoring function, effectively using the node's resources more efficiently.
  - Below is the scenario when the memory burst is scheduled by the kubernetes native scheduler.
```bash
k8 get pods -o wide
NAME                                            READY   STATUS    RESTARTS   AGE   IP            NODE                    NOMINATED NODE   READINESS GATES
static-pod-deployment-mem-heavy-c5795f9-7klj8   1/1     Running   0          8s    10.244.3.5    scheduler-lab-worker3   <none>           <none>
static-pod-deployment-mem-heavy-c5795f9-7lxzb   1/1     Running   0          8s    10.244.2.22   scheduler-lab-worker    <none>           <none>
static-pod-deployment-mem-heavy-c5795f9-gddmg   1/1     Running   0          8s    10.244.1.7    scheduler-lab-worker2   <none>           <none>
static-pod-deployment-mem-heavy-c5795f9-npj52   1/1     Running   0          8s    10.244.2.23   scheduler-lab-worker    <none>           <none>
static-pod-deployment-mem-heavy-c5795f9-p6xjv   1/1     Running   0          8s    10.244.1.5    scheduler-lab-worker2   <none>           <none>
static-pod-deployment-mem-heavy-c5795f9-prxj9   1/1     Running   0          8s    10.244.3.6    scheduler-lab-worker3   <none>           <none>
static-pod-deployment-mem-heavy-c5795f9-rppk8   1/1     Running   0          8s    10.244.1.6    scheduler-lab-worker2   <none>           <none>
static-pod-deployment-mem-heavy-c5795f9-xbjbh   1/1     Running   0          8s    10.244.3.7    scheduler-lab-worker3   <none>           <none>
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
