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

- Experiment: Burst scheduling with runtime-aware bin-packing
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
    - The score for other nodes along with their requestedCPU utilization remained the same throughtout the experiment - (0.053~0.054) - because no pod was scheduled on them at any point of time.
  - Take aways:
    - The scheduler successfully used a hybrid score combining requests, runtime signals, mismatch weighting, and projected utilization.
    - It behaved deterministically and consistently across all eight scheduling decisions.
    - Runtime metrics stayed almost constant, while request-based projected utilization changed only on a worker node as expected. (As the pods were not compute heavy)
    - Worker node=scheduler-lab-worker began with the highest score, and the score increased as the requestedCPU increased (0.06 → 0.09 → 0.11 → 0.13 → 0.13 → ...) within marginal utility.
    - Then the score decreased as node fills up, but not enough to overturn its lead
    - We now have a adaptive runtime-aware bin-packing scheduler that intentionally prefers consolidation (packing) while applying safety penalties to prevent overload.

## Next Steps:
- Staged arrivals
```
Profile breakdown before the YAML:

| Deployment | CPU Request | Memory Request | Purpose |
|------------|-------------|----------------|---------|
| cpu-heavy | 3000m | 256Mi | Stresses CPU scheduling |
| memory-heavy | 200m | 2Gi | Stresses memory scheduling |
| balanced | 1000m | 1Gi | General workload |
| minimal | 100m | 128Mi | Low-priority / best-effort |
| cpu-only | 2000m | (none) | CPU request only |
| memory-only | (none) | 1536Mi | Memory request only |


# Node capacity: 10000m CPU, 8Gi memory — 3 node target

# --- CPU-heavy: burns CPU, barely touches memory ---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: deploy-cpu-heavy
spec:
  replicas: 3
  selector:
    matchLabels:
      app: cpu-heavy
  template:
    metadata:
      labels:
        app: cpu-heavy
    spec:
      schedulerName: kdapt-scheduler
      containers:
        - name: cpu-heavy
          image: nginx:latest
          ports:
            - containerPort: 80
          resources:
            requests:
              cpu: "3000m"
              memory: "256Mi"
            limits:
              cpu: "3000m"
              memory: "256Mi"

---
# Memory-heavy: large memory footprint, minimal CPU
apiVersion: apps/v1
kind: Deployment
metadata:
  name: deploy-memory-heavy
spec:
  replicas: 3
  selector:
    matchLabels:
      app: memory-heavy
  template:
    metadata:
      labels:
        app: memory-heavy
    spec:
      schedulerName: kdapt-scheduler
      containers:
        - name: memory-heavy
          image: nginx:latest
          ports:
            - containerPort: 80
          resources:
            requests:
              cpu: "200m"
              memory: "2Gi"
            limits:
              cpu: "200m"
              memory: "2Gi"

---
# Balanced: moderate CPU and memory — general workload simulation
apiVersion: apps/v1
kind: Deployment
metadata:
  name: deploy-balanced
spec:
  replicas: 3
  selector:
    matchLabels:
      app: balanced
  template:
    metadata:
      labels:
        app: balanced
    spec:
      schedulerName: kdapt-scheduler
      containers:
        - name: balanced
          image: nginx:latest
          ports:
            - containerPort: 80
          resources:
            requests:
              cpu: "1000m"
              memory: "1Gi"
            limits:
              cpu: "1000m"
              memory: "1Gi"

---
# Minimal: tiny requests — best-effort / low-priority workloads
apiVersion: apps/v1
kind: Deployment
metadata:
  name: deploy-minimal
spec:
  replicas: 3
  selector:
    matchLabels:
      app: minimal
  template:
    metadata:
      labels:
        app: minimal
    spec:
      schedulerName: kdapt-scheduler
      containers:
        - name: minimal
          image: nginx:latest
          ports:
            - containerPort: 80
          resources:
            requests:
              cpu: "100m"
              memory: "128Mi"
            limits:
              cpu: "100m"
              memory: "128Mi"

---
# CPU-only: requests CPU but no memory request
apiVersion: apps/v1
kind: Deployment
metadata:
  name: deploy-cpu-only
spec:
  replicas: 3
  selector:
    matchLabels:
      app: cpu-only
  template:
    metadata:
      labels:
        app: cpu-only
    spec:
      schedulerName: kdapt-scheduler
      containers:
        - name: cpu-only
          image: nginx:latest
          ports:
            - containerPort: 80
          resources:
            requests:
              cpu: "2000m"

---
# Memory-only: requests memory but no CPU request
apiVersion: apps/v1
kind: Deployment
metadata:
  name: deploy-memory-only
spec:
  replicas: 3
  selector:
    matchLabels:
      app: memory-only
  template:
    metadata:
      labels:
        app: memory-only
    spec:
      schedulerName: kdapt-scheduler
      containers:
        - name: memory-only
          image: nginx:latest
          ports:
            - containerPort: 80
          resources:
            requests:
              memory: "1536Mi"
Scheduling notes:

cpu-heavy at 3000m means max 3 of them fit per node CPU-wise — good for testing bin-packing.
memory-heavy at 2Gi means max 4 fit per node memory-wise — will compete with other memory consumers.
cpu-only / memory-only are useful for testing how your kdapt-scheduler handles one-dimensional resource pressure.
All use replicas: 3 to match your 3-pods-per-node pattern.
staticpods.yaml
These are all requests based. I need to have a mix of requests and actual utilization.
Good point — nginx just idles, so requests don't reflect real utilization. To get actual resource consumption you need a stress workload. Here's a mix using polinux/stress for active pods and nginx for idle/over-provisioned ones:

Profile summary:

Deployment	Requests (CPU / Mem)	Actual Usage	Pattern
over-prov-cpu	3000m / 512Mi	~100m / ~50Mi	Over-provisioned CPU
over-prov-mem	500m / 2Gi	~100m / ~200Mi	Over-provisioned Memory
cpu-matched	2000m / 256Mi	~2000m / ~10Mi	CPU well-matched
mem-matched	200m / 1Gi	~200m / ~900Mi	Memory well-matched
cpu-hungry	500m / 256Mi	~2000m / ~10Mi	Under-provisioned CPU (bursts past request)
mem-hungry	200m / 512Mi	~200m / ~1.5Gi	Under-provisioned Memory (bursts past request)

# Node capacity: 10000m CPU, 8Gi mem | 3 pods/node | schedulerName: kdapt-scheduler

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

---
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

---
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

---
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

---
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

---
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
- Read Borg design paper to understand the inspiration behind Kubernetes scheduling.
- Validate against kube-scheduler 
  - pod placement distribution by node
  - scheduling latency per pod
  - how tightly each scheduler packs before spilling to the next node
- Explore more complex scheduling scenarios with dynamic resource requests.


## To Document:
- CPU burst experiment with default kube-scheduler and compare the results with kdapt-scheduler
- Discuss on how tightly each scheduler packs before spilling to the next node
- Node placement:
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
- Node usage:
```bash
k8 top nodes
NAME                          CPU(cores)   CPU(%)   MEMORY(bytes)   MEMORY(%)
scheduler-lab-control-plane   143m         1%       1160Mi          14%
scheduler-lab-worker          39m          0%       751Mi           9%
scheduler-lab-worker2         36m          0%       650Mi           8%
scheduler-lab-worker3         30m          0%       673Mi           8%
```
- Node usage with kdapt-scheduler:
```bash
 k8 top nodes
NAME                          CPU(cores)   CPU(%)   MEMORY(bytes)   MEMORY(%)
scheduler-lab-control-plane   160m         1%       1163Mi          14%
scheduler-lab-worker          142m         1%       825Mi           10%
scheduler-lab-worker2         30m          0%       601Mi           7%
scheduler-lab-worker3         40m          0%       644Mi           8%
```
- Memory Heavy burst experiment: (Because memory is treated more conservatively in the scoring algorithm, it is expected that the scheduler will prefer spreading the pods across nodes rather than packing them on a single node, to avoid memory pressure and potential OOM kills.)
- Kubescheduler:
```bash
k8 get pods -o wide
NAME                                            READY   STATUS    RESTARTS   AGE   IP            NODE                    NOMINATED NODE   READINESS GATES
static-pod-deployment-cpu-heavy-c5795f9-7klj8   1/1     Running   0          8s    10.244.3.5    scheduler-lab-worker3   <none>           <none>
static-pod-deployment-cpu-heavy-c5795f9-7lxzb   1/1     Running   0          8s    10.244.2.22   scheduler-lab-worker    <none>           <none>
static-pod-deployment-cpu-heavy-c5795f9-gddmg   1/1     Running   0          8s    10.244.1.7    scheduler-lab-worker2   <none>           <none>
static-pod-deployment-cpu-heavy-c5795f9-npj52   1/1     Running   0          8s    10.244.2.23   scheduler-lab-worker    <none>           <none>
static-pod-deployment-cpu-heavy-c5795f9-p6xjv   1/1     Running   0          8s    10.244.1.5    scheduler-lab-worker2   <none>           <none>
static-pod-deployment-cpu-heavy-c5795f9-prxj9   1/1     Running   0          8s    10.244.3.6    scheduler-lab-worker3   <none>           <none>
static-pod-deployment-cpu-heavy-c5795f9-rppk8   1/1     Running   0          8s    10.244.1.6    scheduler-lab-worker2   <none>           <none>
static-pod-deployment-cpu-heavy-c5795f9-xbjbh   1/1     Running   0          8s    10.244.3.7    scheduler-lab-worker3   <none>           <none>
```
```bash
k8 top nodes
NAME                          CPU(cores)   CPU(%)   MEMORY(bytes)   MEMORY(%)
scheduler-lab-control-plane   149m         1%       1167Mi          14%
scheduler-lab-worker          35m          0%       739Mi           9%
scheduler-lab-worker2         36m          0%       644Mi           8%
scheduler-lab-worker3         39m          0%       696Mi           8%
```
- Kdapt-scheduler:
```bash
I0504 22:38:09.216170       1 kdapt.go:71] nodeMetrics {name=scheduler-lab-worker, cpuMilli=56.00, memoryBytes=743059456.00, smoothedCpuMilli=39.95, smoothedMemoryBytes=763433630.87}
I0504 22:38:09.216177       1 kdapt.go:71] nodeMetrics {name=scheduler-lab-worker2, cpuMilli=99.00, memoryBytes=629538816.00, smoothedCpuMilli=50.12, smoothedMemoryBytes=658346476.48}
I0504 22:38:09.216181       1 kdapt.go:71] nodeMetrics {name=scheduler-lab-worker3, cpuMilli=92.00, memoryBytes=682819584.00, smoothedCpuMilli=51.18, smoothedMemoryBytes=713263989.88}
I0504 22:38:09.216185       1 kdapt.go:80] -----------------------------------------------------------------
I0504 22:38:19.212364       1 kdapt.go:71] nodeMetrics {name=scheduler-lab-control-plane, cpuMilli=147.00, memoryBytes=1229541376.00, smoothedCpuMilli=148.54, smoothedMemoryBytes=1226810603.14}
I0504 22:38:19.212414       1 kdapt.go:71] nodeMetrics {name=scheduler-lab-worker, cpuMilli=28.00, memoryBytes=743329792.00, smoothedCpuMilli=36.37, smoothedMemoryBytes=757402479.21}
I0504 22:38:19.212418       1 kdapt.go:71] nodeMetrics {name=scheduler-lab-worker2, cpuMilli=20.00, memoryBytes=630169600.00, smoothedCpuMilli=41.09, smoothedMemoryBytes=649893413.54}
I0504 22:38:19.212441       1 kdapt.go:71] nodeMetrics {name=scheduler-lab-worker3, cpuMilli=22.00, memoryBytes=683511808.00, smoothedCpuMilli=42.42, smoothedMemoryBytes=704338335.32}
I0504 22:38:19.212447       1 kdapt.go:80] -----------------------------------------------------------------
I0504 22:38:26.312126       1 kdapt.go:219] pod=static-pod-deployment-cpu-heavy-7cbc98d99b-hlc5l node=scheduler-lab-worker reqCPU=0.02 rtCPU=0.00 mismatchCPU=0.02 projectedCPU=0.02 final=0.07
I0504 22:38:26.312176       1 kdapt.go:229] Final score: 0.07045380288849411
I0504 22:38:26.312144       1 kdapt.go:219] pod=static-pod-deployment-cpu-heavy-7cbc98d99b-hlc5l node=scheduler-lab-worker2 reqCPU=0.01 rtCPU=0.00 mismatchCPU=0.01 projectedCPU=0.01 final=0.06
I0504 22:38:26.312205       1 kdapt.go:229] Final score: 0.056745145085702046
I0504 22:38:26.312238       1 kdapt.go:219] pod=static-pod-deployment-cpu-heavy-7cbc98d99b-hlc5l node=scheduler-lab-worker3 reqCPU=0.01 rtCPU=0.00 mismatchCPU=0.01 projectedCPU=0.01 final=0.06
I0504 22:38:26.312271       1 kdapt.go:229] Final score: 0.057057625945172614
I0504 22:38:26.314701       1 kdapt.go:219] pod=static-pod-deployment-cpu-heavy-7cbc98d99b-ppxwb node=scheduler-lab-worker reqCPU=0.02 rtCPU=0.00 mismatchCPU=0.02 projectedCPU=0.02 final=0.12
I0504 22:38:26.314720       1 kdapt.go:229] Final score: 0.11590463314917593
I0504 22:38:26.314725       1 kdapt.go:219] pod=static-pod-deployment-cpu-heavy-7cbc98d99b-ppxwb node=scheduler-lab-worker2 reqCPU=0.01 rtCPU=0.00 mismatchCPU=0.01 projectedCPU=0.01 final=0.06
I0504 22:38:26.314727       1 kdapt.go:229] Final score: 0.056745145085702046
I0504 22:38:26.314729       1 kdapt.go:219] pod=static-pod-deployment-cpu-heavy-7cbc98d99b-ppxwb node=scheduler-lab-worker3 reqCPU=0.01 rtCPU=0.00 mismatchCPU=0.01 projectedCPU=0.01 final=0.06
I0504 22:38:26.314732       1 kdapt.go:229] Final score: 0.057057625945172614
I0504 22:38:26.315201       1 kdapt.go:219] pod=static-pod-deployment-cpu-heavy-7cbc98d99b-swdms node=scheduler-lab-worker reqCPU=0.02 rtCPU=0.00 mismatchCPU=0.02 projectedCPU=0.02 final=0.15
I0504 22:38:26.315231       1 kdapt.go:229] Final score: 0.1545351911556054
I0504 22:38:26.315240       1 kdapt.go:219] pod=static-pod-deployment-cpu-heavy-7cbc98d99b-swdms node=scheduler-lab-worker2 reqCPU=0.01 rtCPU=0.00 mismatchCPU=0.01 projectedCPU=0.01 final=0.06
I0504 22:38:26.315248       1 kdapt.go:229] Final score: 0.056745145085702046
I0504 22:38:26.315257       1 kdapt.go:219] pod=static-pod-deployment-cpu-heavy-7cbc98d99b-swdms node=scheduler-lab-worker3 reqCPU=0.01 rtCPU=0.00 mismatchCPU=0.01 projectedCPU=0.01 final=0.06
I0504 22:38:26.315263       1 kdapt.go:229] Final score: 0.057057625945172614
I0504 22:38:26.318300       1 kdapt.go:219] pod=static-pod-deployment-cpu-heavy-7cbc98d99b-f7nz9 node=scheduler-lab-worker reqCPU=0.02 rtCPU=0.00 mismatchCPU=0.02 projectedCPU=0.02 final=0.19
I0504 22:38:26.318315       1 kdapt.go:229] Final score: 0.1877029676349084
I0504 22:38:26.318319       1 kdapt.go:219] pod=static-pod-deployment-cpu-heavy-7cbc98d99b-f7nz9 node=scheduler-lab-worker2 reqCPU=0.01 rtCPU=0.00 mismatchCPU=0.01 projectedCPU=0.01 final=0.06
I0504 22:38:26.318322       1 kdapt.go:229] Final score: 0.056745145085702046
I0504 22:38:26.318325       1 kdapt.go:219] pod=static-pod-deployment-cpu-heavy-7cbc98d99b-f7nz9 node=scheduler-lab-worker3 reqCPU=0.01 rtCPU=0.00 mismatchCPU=0.01 projectedCPU=0.01 final=0.06
I0504 22:38:26.318327       1 kdapt.go:229] Final score: 0.057057625945172614
I0504 22:38:26.323230       1 kdapt.go:219] pod=static-pod-deployment-cpu-heavy-7cbc98d99b-2jmpw node=scheduler-lab-worker reqCPU=0.02 rtCPU=0.00 mismatchCPU=0.02 projectedCPU=0.02 final=0.22
I0504 22:38:26.323249       1 kdapt.go:229] Final score: 0.21540796258708508
I0504 22:38:26.323257       1 kdapt.go:219] pod=static-pod-deployment-cpu-heavy-7cbc98d99b-2jmpw node=scheduler-lab-worker2 reqCPU=0.01 rtCPU=0.00 mismatchCPU=0.01 projectedCPU=0.01 final=0.06
I0504 22:38:26.323260       1 kdapt.go:229] Final score: 0.056745145085702046
I0504 22:38:26.323262       1 kdapt.go:219] pod=static-pod-deployment-cpu-heavy-7cbc98d99b-2jmpw node=scheduler-lab-worker3 reqCPU=0.01 rtCPU=0.00 mismatchCPU=0.01 projectedCPU=0.01 final=0.06
I0504 22:38:26.323264       1 kdapt.go:229] Final score: 0.057057625945172614
I0504 22:38:26.323512       1 kdapt.go:219] pod=static-pod-deployment-cpu-heavy-7cbc98d99b-jp4gc node=scheduler-lab-worker reqCPU=0.02 rtCPU=0.00 mismatchCPU=0.02 projectedCPU=0.02 final=0.24
I0504 22:38:26.323524       1 kdapt.go:229] Final score: 0.2376501760121353
I0504 22:38:26.323529       1 kdapt.go:219] pod=static-pod-deployment-cpu-heavy-7cbc98d99b-jp4gc node=scheduler-lab-worker2 reqCPU=0.01 rtCPU=0.00 mismatchCPU=0.01 projectedCPU=0.01 final=0.06
I0504 22:38:26.323541       1 kdapt.go:229] Final score: 0.056745145085702046
I0504 22:38:26.323544       1 kdapt.go:219] pod=static-pod-deployment-cpu-heavy-7cbc98d99b-jp4gc node=scheduler-lab-worker3 reqCPU=0.01 rtCPU=0.00 mismatchCPU=0.01 projectedCPU=0.01 final=0.06
I0504 22:38:26.323547       1 kdapt.go:229] Final score: 0.057057625945172614
I0504 22:38:26.323688       1 kdapt.go:219] pod=static-pod-deployment-cpu-heavy-7cbc98d99b-87cv6 node=scheduler-lab-worker reqCPU=0.02 rtCPU=0.00 mismatchCPU=0.02 projectedCPU=0.02 final=0.25
I0504 22:38:26.323697       1 kdapt.go:229] Final score: 0.2544296079100591
I0504 22:38:26.323700       1 kdapt.go:219] pod=static-pod-deployment-cpu-heavy-7cbc98d99b-87cv6 node=scheduler-lab-worker2 reqCPU=0.01 rtCPU=0.00 mismatchCPU=0.01 projectedCPU=0.01 final=0.06
I0504 22:38:26.323703       1 kdapt.go:229] Final score: 0.056745145085702046
I0504 22:38:26.323705       1 kdapt.go:219] pod=static-pod-deployment-cpu-heavy-7cbc98d99b-87cv6 node=scheduler-lab-worker3 reqCPU=0.01 rtCPU=0.00 mismatchCPU=0.01 projectedCPU=0.01 final=0.06
I0504 22:38:26.323707       1 kdapt.go:229] Final score: 0.057057625945172614
I0504 22:38:26.325517       1 kdapt.go:219] pod=static-pod-deployment-cpu-heavy-7cbc98d99b-2f8ss node=scheduler-lab-worker3 reqCPU=0.01 rtCPU=0.00 mismatchCPU=0.01 projectedCPU=0.01 final=0.06
I0504 22:38:26.325532       1 kdapt.go:229] Final score: 0.057057625945172614
I0504 22:38:26.325450       1 kdapt.go:219] pod=static-pod-deployment-cpu-heavy-7cbc98d99b-2f8ss node=scheduler-lab-worker2 reqCPU=0.01 rtCPU=0.00 mismatchCPU=0.01 projectedCPU=0.01 final=0.06
I0504 22:38:26.325583       1 kdapt.go:229] Final score: 0.056745145085702046
I0504 22:38:29.210437       1 kdapt.go:71] nodeMetrics {name=scheduler-lab-control-plane, cpuMilli=147.00, memoryBytes=1229541376.00, smoothedCpuMilli=148.08, smoothedMemoryBytes=1227629835.00}
I0504 22:38:29.210479       1 kdapt.go:71] nodeMetrics {name=scheduler-lab-worker, cpuMilli=28.00, memoryBytes=743329792.00, smoothedCpuMilli=33.86, smoothedMemoryBytes=753180673.05}
I0504 22:38:29.210484       1 kdapt.go:71] nodeMetrics {name=scheduler-lab-worker2, cpuMilli=20.00, memoryBytes=630169600.00, smoothedCpuMilli=34.76, smoothedMemoryBytes=643976269.48}
I0504 22:38:29.210487       1 kdapt.go:71] nodeMetrics {name=scheduler-lab-worker3, cpuMilli=22.00, memoryBytes=683511808.00, smoothedCpuMilli=36.30, smoothedMemoryBytes=698090377.12}
I0504 22:38:29.210492       1 kdapt.go:80] -----------------------------------------------------------------
I0504 22:38:39.209823       1 kdapt.go:71] nodeMetrics {name=scheduler-lab-control-plane, cpuMilli=155.00, memoryBytes=1227595776.00, smoothedCpuMilli=150.16, smoothedMemoryBytes=1227619617.30}
I0504 22:38:39.209852       1 kdapt.go:71] nodeMetrics {name=scheduler-lab-worker, cpuMilli=117.00, memoryBytes=850411520.00, smoothedCpuMilli=58.80, smoothedMemoryBytes=782349927.13}
I0504 22:38:39.209856       1 kdapt.go:71] nodeMetrics {name=scheduler-lab-worker2, cpuMilli=23.00, memoryBytes=630263808.00, smoothedCpuMilli=31.23, smoothedMemoryBytes=639862531.03}
I0504 22:38:39.209859       1 kdapt.go:71] nodeMetrics {name=scheduler-lab-worker3, cpuMilli=31.00, memoryBytes=682491904.00, smoothedCpuMilli=34.71, smoothedMemoryBytes=693410835.19}
I0504 22:38:39.209862       1 kdapt.go:80] -----------------------------------------------------------------
```
```
k8 get pods -o wide
NAME                                               READY   STATUS    RESTARTS   AGE   IP            NODE                    NOMINATED NODE   READINESS GATES
static-pod-deployment-cpu-heavy-7cbc98d99b-2f8ss   1/1     Running   0          81s   10.244.1.8    scheduler-lab-worker2   <none>           <none>
static-pod-deployment-cpu-heavy-7cbc98d99b-2jmpw   1/1     Running   0          81s   10.244.2.27   scheduler-lab-worker    <none>           <none>
static-pod-deployment-cpu-heavy-7cbc98d99b-87cv6   1/1     Running   0          81s   10.244.2.30   scheduler-lab-worker    <none>           <none>
static-pod-deployment-cpu-heavy-7cbc98d99b-f7nz9   1/1     Running   0          81s   10.244.2.28   scheduler-lab-worker    <none>           <none>
static-pod-deployment-cpu-heavy-7cbc98d99b-hlc5l   1/1     Running   0          81s   10.244.2.25   scheduler-lab-worker    <none>           <none>
static-pod-deployment-cpu-heavy-7cbc98d99b-jp4gc   1/1     Running   0          81s   10.244.2.29   scheduler-lab-worker    <none>           <none>
static-pod-deployment-cpu-heavy-7cbc98d99b-ppxwb   1/1     Running   0          81s   10.244.2.26   scheduler-lab-worker    <none>           <none>
static-pod-deployment-cpu-heavy-7cbc98d99b-swdms   1/1     Running   0          81s   10.244.2.24   scheduler-lab-worker    <none>           <none>
```
```
k8 top nodes
NAME                          CPU(cores)   CPU(%)   MEMORY(bytes)   MEMORY(%)
scheduler-lab-control-plane   156m         1%       1170Mi          14%
scheduler-lab-worker          53m          0%       817Mi           10%
scheduler-lab-worker2         30m          0%       615Mi           7%
scheduler-lab-worker3         32m          0%       654Mi           8%
```