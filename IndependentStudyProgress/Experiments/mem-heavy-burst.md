# Experiment: Memory heavy burst scheduling with runtime-aware bin-packing
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
## Before burst:
- All nodes are underutilized, and the cluster is balanced.
- Each node has approximately 30 milli CPU utilization and 703205376.00 bytes (703.20 MB) of memory utilization.
## During burst:
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
