## Hands-on progress till now:
- Setup KIND cluster for local Kubernetes development and testing.
- Forked Kubernetes repo and set up local development environment.
- Implemented basic Kdapt Scheduler plugin with scoring functionality. (returning static scores for testing)
- Genralised the plugin as per the Kubernetes scheduler plugin framework, and test it with static pod resource requests to verify that the plugin is being invoked correctly during scheduling.
- Written a simple bin pack scoring algo based on Node resource utilization, combining CPU and memory utilization, giving more weight to CPU.

### Testing methodology:
#### Static Pod testing:
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

## Next Steps:
- Read Borg design paper to understand the inspiration behind Kubernetes scheduling.
- Re-create KIND cluster with uneven node resources to test the bin pack scoring algorithm on static pods with varying resource requests.
- Explore more complex scheduling scenarios with dynamic resource requests.


## To Document:
```
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

```
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

#  9 7 6 | 7 7 5

```
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

```
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

```
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

```
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
