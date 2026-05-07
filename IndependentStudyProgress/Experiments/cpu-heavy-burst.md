# Experiment: CPU heavy burst scheduling with runtime-aware bin-packing
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
## Before burst:
- All nodes are underutilized, and the cluster is balanced.
- Each node has approximately 35 milli CPU utilization and 703205376.00 bytes (703.20 MB) of memory utilization.
## During burst:
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
## Take aways:
- The scheduler successfully used a hybrid score combining requests, runtime signals, mismatch weighting, and projected utilization.
- It behaved deterministically and consistently across all eight scheduling decisions.
- Runtime metrics stayed almost constant, while request-based projected utilization changed only on a worker node as expected. (As the pods were not compute heavy)
- Worker node=scheduler-lab-worker began with the highest score, and the score increased as the requestedCPU increased (0.06 → 0.09 → 0.11 → 0.13 → 0.13 → ...) within marginal utility.
- Then the score decreased as node fills up, but not enough to overturn its lead
- We now have a adaptive runtime-aware bin-packing scheduler that intentionally prefers consolidation (packing) while applying safety penalties to prevent overload.
