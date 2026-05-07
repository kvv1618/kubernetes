# Experiment: Staged Gradual Rollout with 15sec delay
-  Scenario: gradual | Delay: 15s between arrivals
    - Applied files in this order:
        - cpu.yaml
        - mem.yaml
        - heavy-cpu-request.yaml
        - heavy-mem-request.yaml
        - heavy-cpu-util.yaml
        - heavy-mem-util.yaml
## Before deployments:
- All workers nearly idle.
- Worker had slightly more existing requests (~0.02 CPU util) than worker2/worker3 (~0.01)
- This tiny difference is what drives all early bin-packing decisions, as the scheduler tries to pack on the node with slightly higher existing utilization to maximize packing without risking overcommitment.
## During deployments:
- Wave 1 — cpu.yaml → all 3 pods → worker
    - The scheduler packed all 3 pods on worker, which had slightly higher existing CPU utilization, to maximize packing without risking overcommitment.
        - pod h4jpm: worker=0.1155  worker2=0.1032  worker3=0.1028  → worker wins
        - pod c2rn6: worker=0.1692  worker2=0.1032  worker3=0.1028  → worker wins (now has h4jpm's request)
        - pod 4b5dv: worker=0.1842  worker2=0.1032  worker3=0.1028  → worker wins (now has both)
- Wave 2 — mem.yaml → all 3 pods → worker
    - The scheduler again packed all 3 pods on worker.
    - Worker's requestedCpuOnNode=0.62, requestedMemOnNode=0.13. Beta is low (raw≈smoothed, cluster stable) so effective≈smoothed. 
    - Worker keeps snowballing.
        - pod ddhwk: worker=0.2291  worker2=0.0631  worker3=0.0628  → worker wins
        - pod 64grg: worker=0.2642  worker2=0.0631  worker3=0.0628  → worker wins (now has ddhwk's requests)
        - pod pthmd: worker=0.2935  worker2=0.0631  worker3=0.0628  → worker wins (now has both)
- Wave 3 — heavy-cpu-request.yaml → worker(1) + worker2(2)
    - Worker is now at 6012m raw CPU, effective≈0.46
    - beta blending in action: raw=0.60, smoothed=0.22, beta≈0.63 → effective=0.46
        - pod t577k: worker=0.6101  worker2=0.1564  worker3=0.1560
            - worker (still highest) worker: req=0.68, rt=0.46, proj=0.98 ← near full
        - pod 7hrt7: worker2=0.1564  worker3=0.1560  → worker2 wins (now has t577k's request)
            - [worker filtered out — proj would exceed 1.0] (Filter phase prevents scheduling on worker)
        - pod nldqw: worker2=0.2168  worker3=0.1560
            - worker2 (already has 7hrt7's request)
- Wave 4 - heavy-mem-request.yaml → worker2(3)
    - Worker2 now has 2 over-prov-cpu pods, so it has elevated requests
    - Worker is filtered out (too full on CPU)
    - Worker3 still nearly empty
    - Bin-packing continues: worker2 snowballs same as worker did in wave 1.
        - pod ttjbr: worker2=0.2250  worker3=0.1222  → worker2
        - pod l554h: worker2=0.2777  worker3=0.1222  → worker2
        - pod dhjz7: worker2=0.3061  worker3=0.1222  → worker2
- Wave 5 - heavy-cpu-util.yaml → worker2(2) + worker3(1)
    - Worker2 accepts two cpu-hungry pods before Filter blocks it
        - pod djs98: worker2=0.2714  worker3=0.0406  → worker2
        - pod tvvps: worker2=0.2564  worker3=0.0406  → worker2
        - pod pk6l9: [worker2 filtered out — CPU requests full]  → worker3
- Wave 6 - heavy-cpu-util.yaml → worker(1) + worker3(2)
    - Worker's CPU is full but memory headroom exists
    - Worker was not selected for heavy-mem-request because it had a cpu request of 500m. This heavy-cpu-util pod has an initial request of 200m only, which the Worker can accomudate.
    - _Note: This will make Worker's projected cpuUtil to 1, which should kick-in penality, but is not happening as of now_
        - pod cb2fp: worker=0.6652  worker3=0.0704  → worker
           - worker: req=0.98, rt=0.56 (effective after beta), proj=1.0, mismatch=0.42
        - pod 8jslm: → worker3
        - pod 9kmtn: → worker3  ← OOMKilled

- Raw scheduler logs:
```bash
I0507 19:53:21.675460       1 kdapt.go:71] nodeMetrics {name=scheduler-lab-control-plane, cpuMilli=147.00, memoryBytes=768376832.00, smoothedCpuMilli=150.76, smoothedMemoryBytes=766024562.85}
I0507 19:53:21.675517       1 kdapt.go:71] nodeMetrics {name=scheduler-lab-worker, cpuMilli=28.00, memoryBytes=239210496.00, smoothedCpuMilli=29.84, smoothedMemoryBytes=238672007.01}
I0507 19:53:21.675523       1 kdapt.go:71] nodeMetrics {name=scheduler-lab-worker2, cpuMilli=31.00, memoryBytes=231190528.00, smoothedCpuMilli=32.29, smoothedMemoryBytes=231050633.17}
I0507 19:53:21.675527       1 kdapt.go:71] nodeMetrics {name=scheduler-lab-worker3, cpuMilli=18.00, memoryBytes=194383872.00, smoothedCpuMilli=26.20, smoothedMemoryBytes=193971492.23}
I0507 19:53:21.675531       1 kdapt.go:80] -----------------------------------------------------------------
I0507 19:53:31.672653       1 kdapt.go:71] nodeMetrics {name=scheduler-lab-control-plane, cpuMilli=147.00, memoryBytes=768376832.00, smoothedCpuMilli=149.63, smoothedMemoryBytes=766730243.59}
I0507 19:53:31.672720       1 kdapt.go:71] nodeMetrics {name=scheduler-lab-worker, cpuMilli=28.00, memoryBytes=239210496.00, smoothedCpuMilli=29.29, smoothedMemoryBytes=238833553.71}
I0507 19:53:31.672727       1 kdapt.go:71] nodeMetrics {name=scheduler-lab-worker2, cpuMilli=31.00, memoryBytes=231190528.00, smoothedCpuMilli=31.91, smoothedMemoryBytes=231092601.62}
I0507 19:53:31.672732       1 kdapt.go:71] nodeMetrics {name=scheduler-lab-worker3, cpuMilli=18.00, memoryBytes=194383872.00, smoothedCpuMilli=23.74, smoothedMemoryBytes=194095206.16}
I0507 19:53:31.672736       1 kdapt.go:80] -----------------------------------------------------------------
I0507 19:53:41.384855       1 kdapt.go:241] pod=deploy-cpu-matched-67dc8cb568-h4jpm node=scheduler-lab-worker | cpu: req=0.02 rt=0.00 mismatch=0.02 proj=0.22 alpha=0.31 | mem: req=0.03 rt=0.03 mismatch=0.00 proj=0.06 alpha=0.10 | penalty=0.00 final=0.1155
I0507 19:53:41.384848       1 kdapt.go:241] pod=deploy-cpu-matched-67dc8cb568-h4jpm node=scheduler-lab-worker3 | cpu: req=0.01 rt=0.00 mismatch=0.01 proj=0.21 alpha=0.31 | mem: req=0.01 rt=0.02 mismatch=0.02 proj=0.04 alpha=0.11 | penalty=0.00 final=0.1028
I0507 19:53:41.384852       1 kdapt.go:241] pod=deploy-cpu-matched-67dc8cb568-h4jpm node=scheduler-lab-worker2 | cpu: req=0.01 rt=0.00 mismatch=0.01 proj=0.21 alpha=0.31 | mem: req=0.01 rt=0.03 mismatch=0.02 proj=0.04 alpha=0.11 | penalty=0.00 final=0.1032
I0507 19:53:41.387396       1 kdapt.go:241] pod=deploy-cpu-matched-67dc8cb568-c2rn6 node=scheduler-lab-worker | cpu: req=0.22 rt=0.00 mismatch=0.22 proj=0.42 alpha=0.47 | mem: req=0.06 rt=0.03 mismatch=0.04 proj=0.10 alpha=0.11 | penalty=0.00 final=0.1692
I0507 19:53:41.387410       1 kdapt.go:241] pod=deploy-cpu-matched-67dc8cb568-c2rn6 node=scheduler-lab-worker2 | cpu: req=0.01 rt=0.00 mismatch=0.01 proj=0.21 alpha=0.31 | mem: req=0.01 rt=0.03 mismatch=0.02 proj=0.04 alpha=0.11 | penalty=0.00 final=0.1032
I0507 19:53:41.387420       1 kdapt.go:241] pod=deploy-cpu-matched-67dc8cb568-c2rn6 node=scheduler-lab-worker3 | cpu: req=0.01 rt=0.00 mismatch=0.01 proj=0.21 alpha=0.31 | mem: req=0.01 rt=0.02 mismatch=0.02 proj=0.04 alpha=0.11 | penalty=0.00 final=0.1028
I0507 19:53:41.388198       1 kdapt.go:241] pod=deploy-cpu-matched-67dc8cb568-4b5dv node=scheduler-lab-worker | cpu: req=0.42 rt=0.00 mismatch=0.42 proj=0.62 alpha=0.63 | mem: req=0.10 rt=0.03 mismatch=0.07 proj=0.13 alpha=0.13 | penalty=0.00 final=0.1842
I0507 19:53:41.388218       1 kdapt.go:241] pod=deploy-cpu-matched-67dc8cb568-4b5dv node=scheduler-lab-worker3 | cpu: req=0.01 rt=0.00 mismatch=0.01 proj=0.21 alpha=0.31 | mem: req=0.01 rt=0.02 mismatch=0.02 proj=0.04 alpha=0.11 | penalty=0.00 final=0.1028
I0507 19:53:41.388204       1 kdapt.go:241] pod=deploy-cpu-matched-67dc8cb568-4b5dv node=scheduler-lab-worker2 | cpu: req=0.01 rt=0.00 mismatch=0.01 proj=0.21 alpha=0.31 | mem: req=0.01 rt=0.03 mismatch=0.02 proj=0.04 alpha=0.11 | penalty=0.00 final=0.1032
I0507 19:53:41.669535       1 kdapt.go:71] nodeMetrics {name=scheduler-lab-control-plane, cpuMilli=141.00, memoryBytes=769249280.00, smoothedCpuMilli=147.04, smoothedMemoryBytes=767485954.52}
I0507 19:53:41.669568       1 kdapt.go:71] nodeMetrics {name=scheduler-lab-worker, cpuMilli=28.00, memoryBytes=239693824.00, smoothedCpuMilli=28.90, smoothedMemoryBytes=239091634.80}
I0507 19:53:41.669571       1 kdapt.go:71] nodeMetrics {name=scheduler-lab-worker2, cpuMilli=27.00, memoryBytes=232308736.00, smoothedCpuMilli=30.43, smoothedMemoryBytes=231457441.93}
I0507 19:53:41.669572       1 kdapt.go:71] nodeMetrics {name=scheduler-lab-worker3, cpuMilli=19.00, memoryBytes=195207168.00, smoothedCpuMilli=22.32, smoothedMemoryBytes=194428794.71}
I0507 19:53:41.669574       1 kdapt.go:80] -----------------------------------------------------------------
I0507 19:53:51.670002       1 kdapt.go:71] nodeMetrics {name=scheduler-lab-control-plane, cpuMilli=134.00, memoryBytes=779100160.00, smoothedCpuMilli=143.13, smoothedMemoryBytes=770970216.16}
I0507 19:53:51.670080       1 kdapt.go:71] nodeMetrics {name=scheduler-lab-worker, cpuMilli=1017.00, memoryBytes=286601216.00, smoothedCpuMilli=325.33, smoothedMemoryBytes=253344509.16}
I0507 19:53:51.670084       1 kdapt.go:71] nodeMetrics {name=scheduler-lab-worker2, cpuMilli=29.00, memoryBytes=231874560.00, smoothedCpuMilli=30.00, smoothedMemoryBytes=231582577.35}
I0507 19:53:51.670087       1 kdapt.go:71] nodeMetrics {name=scheduler-lab-worker3, cpuMilli=16.00, memoryBytes=195149824.00, smoothedCpuMilli=20.42, smoothedMemoryBytes=194645103.50}
I0507 19:53:51.670090       1 kdapt.go:80] -----------------------------------------------------------------
I0507 19:53:56.497944       1 kdapt.go:241] pod=deploy-mem-matched-5c49fb5d9d-ddhwk node=scheduler-lab-worker | cpu: req=0.62 rt=0.08 mismatch=0.54 proj=0.64 alpha=0.73 | mem: req=0.13 rt=0.03 mismatch=0.10 proj=0.26 alpha=0.14 | penalty=0.00 final=0.2291
I0507 19:53:56.497970       1 kdapt.go:241] pod=deploy-mem-matched-5c49fb5d9d-ddhwk node=scheduler-lab-worker2 | cpu: req=0.01 rt=0.00 mismatch=0.01 proj=0.03 alpha=0.31 | mem: req=0.01 rt=0.03 mismatch=0.02 proj=0.14 alpha=0.11 | penalty=0.00 final=0.0631
I0507 19:53:56.497980       1 kdapt.go:241] pod=deploy-mem-matched-5c49fb5d9d-ddhwk node=scheduler-lab-worker3 | cpu: req=0.01 rt=0.00 mismatch=0.01 proj=0.03 alpha=0.31 | mem: req=0.01 rt=0.02 mismatch=0.02 proj=0.14 alpha=0.11 | penalty=0.00 final=0.0628
I0507 19:53:56.504229       1 kdapt.go:241] pod=deploy-mem-matched-5c49fb5d9d-64grg node=scheduler-lab-worker | cpu: req=0.64 rt=0.08 mismatch=0.56 proj=0.66 alpha=0.75 | mem: req=0.26 rt=0.03 mismatch=0.23 proj=0.39 alpha=0.19 | penalty=0.00 final=0.2642
I0507 19:53:56.504260       1 kdapt.go:241] pod=deploy-mem-matched-5c49fb5d9d-64grg node=scheduler-lab-worker2 | cpu: req=0.01 rt=0.00 mismatch=0.01 proj=0.03 alpha=0.31 | mem: req=0.01 rt=0.03 mismatch=0.02 proj=0.14 alpha=0.11 | penalty=0.00 final=0.0631
I0507 19:53:56.504279       1 kdapt.go:241] pod=deploy-mem-matched-5c49fb5d9d-64grg node=scheduler-lab-worker3 | cpu: req=0.01 rt=0.00 mismatch=0.01 proj=0.03 alpha=0.31 | mem: req=0.01 rt=0.02 mismatch=0.02 proj=0.14 alpha=0.11 | penalty=0.00 final=0.0628
I0507 19:53:56.504691       1 kdapt.go:241] pod=deploy-mem-matched-5c49fb5d9d-pthmd node=scheduler-lab-worker | cpu: req=0.66 rt=0.08 mismatch=0.58 proj=0.68 alpha=0.76 | mem: req=0.39 rt=0.03 mismatch=0.36 proj=0.52 alpha=0.24 | penalty=0.00 final=0.2935
I0507 19:53:56.504714       1 kdapt.go:241] pod=deploy-mem-matched-5c49fb5d9d-pthmd node=scheduler-lab-worker2 | cpu: req=0.01 rt=0.00 mismatch=0.01 proj=0.03 alpha=0.31 | mem: req=0.01 rt=0.03 mismatch=0.02 proj=0.14 alpha=0.11 | penalty=0.00 final=0.0631
I0507 19:53:56.504722       1 kdapt.go:241] pod=deploy-mem-matched-5c49fb5d9d-pthmd node=scheduler-lab-worker3 | cpu: req=0.01 rt=0.00 mismatch=0.01 proj=0.03 alpha=0.31 | mem: req=0.01 rt=0.02 mismatch=0.02 proj=0.14 alpha=0.11 | penalty=0.00 final=0.0628
I0507 19:54:01.783103       1 kdapt.go:71] nodeMetrics {name=scheduler-lab-control-plane, cpuMilli=134.00, memoryBytes=779100160.00, smoothedCpuMilli=140.39, smoothedMemoryBytes=773409199.31}
I0507 19:54:01.783418       1 kdapt.go:71] nodeMetrics {name=scheduler-lab-worker, cpuMilli=1017.00, memoryBytes=286601216.00, smoothedCpuMilli=532.83, smoothedMemoryBytes=263321521.21}
I0507 19:54:01.783425       1 kdapt.go:71] nodeMetrics {name=scheduler-lab-worker2, cpuMilli=29.00, memoryBytes=231874560.00, smoothedCpuMilli=29.70, smoothedMemoryBytes=231670172.15}
I0507 19:54:01.783429       1 kdapt.go:71] nodeMetrics {name=scheduler-lab-worker3, cpuMilli=16.00, memoryBytes=195149824.00, smoothedCpuMilli=19.10, smoothedMemoryBytes=194796519.65}
I0507 19:54:01.783433       1 kdapt.go:80] -----------------------------------------------------------------
I0507 19:54:11.675127       1 kdapt.go:71] nodeMetrics {name=scheduler-lab-control-plane, cpuMilli=122.00, memoryBytes=782229504.00, smoothedCpuMilli=134.87, smoothedMemoryBytes=776055290.72}
I0507 19:54:11.675255       1 kdapt.go:71] nodeMetrics {name=scheduler-lab-worker, cpuMilli=6012.00, memoryBytes=286371840.00, smoothedCpuMilli=2176.58, smoothedMemoryBytes=270236616.85}
I0507 19:54:11.675259       1 kdapt.go:71] nodeMetrics {name=scheduler-lab-worker2, cpuMilli=23.00, memoryBytes=237293568.00, smoothedCpuMilli=27.69, smoothedMemoryBytes=233357190.90}
I0507 19:54:11.675261       1 kdapt.go:71] nodeMetrics {name=scheduler-lab-worker3, cpuMilli=18.00, memoryBytes=195452928.00, smoothedCpuMilli=18.77, smoothedMemoryBytes=194993442.15}
I0507 19:54:11.675263       1 kdapt.go:80] -----------------------------------------------------------------
I0507 19:54:11.766997       1 kdapt.go:241] pod=deploy-over-prov-cpu-5b74dd784d-t577k node=scheduler-lab-worker | cpu: req=0.68 rt=0.46 mismatch=0.22 proj=0.98 alpha=0.47 | mem: req=0.52 rt=0.03 mismatch=0.49 proj=0.59 alpha=0.30 | penalty=0.00 final=0.6101
I0507 19:54:11.768294       1 kdapt.go:241] pod=deploy-over-prov-cpu-5b74dd784d-t577k node=scheduler-lab-worker3 | cpu: req=0.01 rt=0.00 mismatch=0.01 proj=0.31 alpha=0.31 | mem: req=0.01 rt=0.02 mismatch=0.02 proj=0.07 alpha=0.11 | penalty=0.00 final=0.1560
I0507 19:54:11.768252       1 kdapt.go:241] pod=deploy-over-prov-cpu-5b74dd784d-t577k node=scheduler-lab-worker2 | cpu: req=0.01 rt=0.00 mismatch=0.01 proj=0.31 alpha=0.31 | mem: req=0.01 rt=0.03 mismatch=0.02 proj=0.07 alpha=0.11 | penalty=0.00 final=0.1564
I0507 19:54:11.795534       1 kdapt.go:241] pod=deploy-over-prov-cpu-5b74dd784d-7hrt7 node=scheduler-lab-worker2 | cpu: req=0.01 rt=0.00 mismatch=0.01 proj=0.31 alpha=0.31 | mem: req=0.01 rt=0.03 mismatch=0.02 proj=0.07 alpha=0.11 | penalty=0.00 final=0.1564
I0507 19:54:11.795575       1 kdapt.go:241] pod=deploy-over-prov-cpu-5b74dd784d-7hrt7 node=scheduler-lab-worker3 | cpu: req=0.01 rt=0.00 mismatch=0.01 proj=0.31 alpha=0.31 | mem: req=0.01 rt=0.02 mismatch=0.02 proj=0.07 alpha=0.11 | penalty=0.00 final=0.1560
I0507 19:54:11.796873       1 kdapt.go:241] pod=deploy-over-prov-cpu-5b74dd784d-nldqw node=scheduler-lab-worker2 | cpu: req=0.31 rt=0.00 mismatch=0.31 proj=0.61 alpha=0.55 | mem: req=0.07 rt=0.03 mismatch=0.04 proj=0.14 alpha=0.12 | penalty=0.00 final=0.2168
I0507 19:54:11.796913       1 kdapt.go:241] pod=deploy-over-prov-cpu-5b74dd784d-nldqw node=scheduler-lab-worker3 | cpu: req=0.01 rt=0.00 mismatch=0.01 proj=0.31 alpha=0.31 | mem: req=0.01 rt=0.02 mismatch=0.02 proj=0.07 alpha=0.11 | penalty=0.00 final=0.1560
I0507 19:54:21.693272       1 kdapt.go:71] nodeMetrics {name=scheduler-lab-control-plane, cpuMilli=192.00, memoryBytes=783548416.00, smoothedCpuMilli=152.01, smoothedMemoryBytes=778303228.30}
I0507 19:54:21.693497       1 kdapt.go:71] nodeMetrics {name=scheduler-lab-worker, cpuMilli=6616.00, memoryBytes=3179069440.00, smoothedCpuMilli=3508.41, smoothedMemoryBytes=1142886463.79}
I0507 19:54:21.693505       1 kdapt.go:71] nodeMetrics {name=scheduler-lab-worker2, cpuMilli=47.00, memoryBytes=237805568.00, smoothedCpuMilli=33.48, smoothedMemoryBytes=234691704.03}
I0507 19:54:21.693511       1 kdapt.go:71] nodeMetrics {name=scheduler-lab-worker3, cpuMilli=37.00, memoryBytes=196411392.00, smoothedCpuMilli=24.24, smoothedMemoryBytes=195418827.11}
I0507 19:54:21.693519       1 kdapt.go:80] -----------------------------------------------------------------
I0507 19:54:27.054341       1 kdapt.go:241] pod=deploy-over-prov-mem-69ffc6cdd4-ttjbr node=scheduler-lab-worker3 | cpu: req=0.01 rt=0.00 mismatch=0.01 proj=0.06 alpha=0.31 | mem: req=0.01 rt=0.02 mismatch=0.02 proj=0.27 alpha=0.11 | penalty=0.00 final=0.1222
I0507 19:54:27.053986       1 kdapt.go:241] pod=deploy-over-prov-mem-69ffc6cdd4-ttjbr node=scheduler-lab-worker2 | cpu: req=0.61 rt=0.00 mismatch=0.61 proj=0.66 alpha=0.79 | mem: req=0.14 rt=0.03 mismatch=0.11 proj=0.40 alpha=0.14 | penalty=0.00 final=0.2250
I0507 19:54:27.060957       1 kdapt.go:241] pod=deploy-over-prov-mem-69ffc6cdd4-l554h node=scheduler-lab-worker2 | cpu: req=0.66 rt=0.00 mismatch=0.66 proj=0.71 alpha=0.83 | mem: req=0.40 rt=0.03 mismatch=0.37 proj=0.66 alpha=0.25 | penalty=0.00 final=0.2777
I0507 19:54:27.061559       1 kdapt.go:241] pod=deploy-over-prov-mem-69ffc6cdd4-l554h node=scheduler-lab-worker3 | cpu: req=0.01 rt=0.00 mismatch=0.01 proj=0.06 alpha=0.31 | mem: req=0.01 rt=0.02 mismatch=0.02 proj=0.27 alpha=0.11 | penalty=0.00 final=0.1222
I0507 19:54:27.063295       1 kdapt.go:241] pod=deploy-over-prov-mem-69ffc6cdd4-dhjz7 node=scheduler-lab-worker2 | cpu: req=0.71 rt=0.00 mismatch=0.71 proj=0.76 alpha=0.87 | mem: req=0.66 rt=0.03 mismatch=0.63 proj=0.92 alpha=0.35 | penalty=0.00 final=0.3061
I0507 19:54:27.063376       1 kdapt.go:241] pod=deploy-over-prov-mem-69ffc6cdd4-dhjz7 node=scheduler-lab-worker3 | cpu: req=0.01 rt=0.00 mismatch=0.01 proj=0.06 alpha=0.31 | mem: req=0.01 rt=0.02 mismatch=0.02 proj=0.27 alpha=0.11 | penalty=0.00 final=0.1222
I0507 19:54:31.747361       1 kdapt.go:71] nodeMetrics {name=scheduler-lab-control-plane, cpuMilli=192.00, memoryBytes=783548416.00, smoothedCpuMilli=164.01, smoothedMemoryBytes=779876784.61}
I0507 19:54:31.747522       1 kdapt.go:71] nodeMetrics {name=scheduler-lab-worker, cpuMilli=6616.00, memoryBytes=3179069440.00, smoothedCpuMilli=4440.69, smoothedMemoryBytes=1753741356.66}
I0507 19:54:31.747526       1 kdapt.go:71] nodeMetrics {name=scheduler-lab-worker2, cpuMilli=47.00, memoryBytes=237805568.00, smoothedCpuMilli=37.54, smoothedMemoryBytes=235625863.22}
I0507 19:54:31.747527       1 kdapt.go:71] nodeMetrics {name=scheduler-lab-worker3, cpuMilli=37.00, memoryBytes=196411392.00, smoothedCpuMilli=28.07, smoothedMemoryBytes=195716596.58}
I0507 19:54:31.747532       1 kdapt.go:80] -----------------------------------------------------------------
I0507 19:54:41.691245       1 kdapt.go:71] nodeMetrics {name=scheduler-lab-control-plane, cpuMilli=241.00, memoryBytes=784674816.00, smoothedCpuMilli=187.11, smoothedMemoryBytes=781316194.03}
I0507 19:54:41.691382       1 kdapt.go:71] nodeMetrics {name=scheduler-lab-worker, cpuMilli=6616.00, memoryBytes=3181658112.00, smoothedCpuMilli=5093.28, smoothedMemoryBytes=2182116383.26}
I0507 19:54:41.691385       1 kdapt.go:71] nodeMetrics {name=scheduler-lab-worker2, cpuMilli=183.00, memoryBytes=326070272.00, smoothedCpuMilli=81.18, smoothedMemoryBytes=262759185.86}
I0507 19:54:41.691387       1 kdapt.go:71] nodeMetrics {name=scheduler-lab-worker3, cpuMilli=42.00, memoryBytes=196866048.00, smoothedCpuMilli=32.25, smoothedMemoryBytes=196061432.00}
I0507 19:54:41.691389       1 kdapt.go:80] -----------------------------------------------------------------
I0507 19:54:42.281619       1 kdapt.go:241] pod=deploy-cpu-hungry-5464bdc95b-djs98 node=scheduler-lab-worker3 | cpu: req=0.01 rt=0.00 mismatch=0.01 proj=0.06 alpha=0.31 | mem: req=0.01 rt=0.02 mismatch=0.02 proj=0.04 alpha=0.11 | penalty=0.00 final=0.0406
I0507 19:54:42.281714       1 kdapt.go:241] pod=deploy-cpu-hungry-5464bdc95b-djs98 node=scheduler-lab-worker2 | cpu: req=0.76 rt=0.01 mismatch=0.75 proj=0.81 alpha=0.90 | mem: req=0.92 rt=0.03 mismatch=0.89 proj=0.95 alpha=0.46 | penalty=0.00 final=0.2714
I0507 19:54:42.303325       1 kdapt.go:241] pod=deploy-cpu-hungry-5464bdc95b-tvvps node=scheduler-lab-worker2 | cpu: req=0.81 rt=0.01 mismatch=0.80 proj=0.86 alpha=0.94 | mem: req=0.95 rt=0.03 mismatch=0.92 proj=0.99 alpha=0.47 | penalty=0.00 final=0.2564
I0507 19:54:42.303477       1 kdapt.go:241] pod=deploy-cpu-hungry-5464bdc95b-tvvps node=scheduler-lab-worker3 | cpu: req=0.01 rt=0.00 mismatch=0.01 proj=0.06 alpha=0.31 | mem: req=0.01 rt=0.02 mismatch=0.02 proj=0.04 alpha=0.11 | penalty=0.00 final=0.0406
I0507 19:54:51.910313       1 kdapt.go:71] nodeMetrics {name=scheduler-lab-control-plane, cpuMilli=232.00, memoryBytes=788250624.00, smoothedCpuMilli=200.57, smoothedMemoryBytes=783396523.02}
I0507 19:54:51.910474       1 kdapt.go:71] nodeMetrics {name=scheduler-lab-worker, cpuMilli=6350.00, memoryBytes=3184058368.00, smoothedCpuMilli=5470.30, smoothedMemoryBytes=2482698978.68}
I0507 19:54:51.910480       1 kdapt.go:71] nodeMetrics {name=scheduler-lab-worker2, cpuMilli=77.00, memoryBytes=339275776.00, smoothedCpuMilli=79.92, smoothedMemoryBytes=285714162.90}
I0507 19:54:51.910482       1 kdapt.go:71] nodeMetrics {name=scheduler-lab-worker3, cpuMilli=34.00, memoryBytes=197070848.00, smoothedCpuMilli=32.77, smoothedMemoryBytes=196364256.80}
I0507 19:54:51.910485       1 kdapt.go:80] -----------------------------------------------------------------
I0507 19:54:57.946819       1 kdapt.go:241] pod=deploy-mem-hungry-657fb577df-cb2fp node=scheduler-lab-worker | cpu: req=0.98 rt=0.56 mismatch=0.42 proj=1.00 alpha=0.64 | mem: req=0.59 rt=0.32 mismatch=0.27 proj=0.65 alpha=0.21 | penalty=0.00 final=0.6652
I0507 19:54:57.950941       1 kdapt.go:241] pod=deploy-mem-hungry-657fb577df-cb2fp node=scheduler-lab-worker3 | cpu: req=0.06 rt=0.00 mismatch=0.06 proj=0.08 alpha=0.35 | mem: req=0.04 rt=0.02 mismatch=0.02 proj=0.10 alpha=0.11 | penalty=0.00 final=0.0704
I0507 19:55:01.706711       1 kdapt.go:71] nodeMetrics {name=scheduler-lab-control-plane, cpuMilli=232.00, memoryBytes=788250624.00, smoothedCpuMilli=210.00, smoothedMemoryBytes=784852753.31}
I0507 19:55:01.712213       1 kdapt.go:71] nodeMetrics {name=scheduler-lab-worker, cpuMilli=6350.00, memoryBytes=3184058368.00, smoothedCpuMilli=5734.21, smoothedMemoryBytes=2693106795.48}
I0507 19:55:01.712243       1 kdapt.go:71] nodeMetrics {name=scheduler-lab-worker2, cpuMilli=77.00, memoryBytes=339275776.00, smoothedCpuMilli=79.05, smoothedMemoryBytes=301782646.83}
I0507 19:55:01.712254       1 kdapt.go:71] nodeMetrics {name=scheduler-lab-worker3, cpuMilli=34.00, memoryBytes=197070848.00, smoothedCpuMilli=33.14, smoothedMemoryBytes=196576234.16}
I0507 19:55:01.712263       1 kdapt.go:80] -----------------------------------------------------------------
I0507 19:55:13.927814       1 kdapt.go:71] nodeMetrics {name=scheduler-lab-control-plane, cpuMilli=294.00, memoryBytes=791105536.00, smoothedCpuMilli=235.20, smoothedMemoryBytes=786728588.12}
I0507 19:55:13.931634       1 kdapt.go:71] nodeMetrics {name=scheduler-lab-worker, cpuMilli=4087.00, memoryBytes=3185438720.00, smoothedCpuMilli=5240.04, smoothedMemoryBytes=2840806372.83}
I0507 19:55:13.931646       1 kdapt.go:71] nodeMetrics {name=scheduler-lab-worker2, cpuMilli=2674.00, memoryBytes=361754624.00, smoothedCpuMilli=857.53, smoothedMemoryBytes=319774239.98}
I0507 19:55:13.931651       1 kdapt.go:71] nodeMetrics {name=scheduler-lab-worker3, cpuMilli=1609.00, memoryBytes=234983424.00, smoothedCpuMilli=505.90, smoothedMemoryBytes=208098391.11}
```
- Pod distribution:
```bash
k8 get pods -o wide                                       
NAME                                    READY   STATUS      RESTARTS      AGE    IP            NODE                    NOMINATED NODE   READINESS GATES
deploy-cpu-hungry-5464bdc95b-djs98      1/1     Running     0             42s    10.244.1.35   scheduler-lab-worker2   <none>           <none>
deploy-cpu-hungry-5464bdc95b-pk6l9      1/1     Running     0             42s    10.244.3.21   scheduler-lab-worker3   <none>           <none>
deploy-cpu-hungry-5464bdc95b-tvvps      1/1     Running     0             42s    10.244.1.34   scheduler-lab-worker2   <none>           <none>
deploy-cpu-matched-67dc8cb568-4b5dv     1/1     Running     0             103s   10.244.2.46   scheduler-lab-worker    <none>           <none>
deploy-cpu-matched-67dc8cb568-c2rn6     1/1     Running     0             103s   10.244.2.44   scheduler-lab-worker    <none>           <none>
deploy-cpu-matched-67dc8cb568-h4jpm     1/1     Running     0             103s   10.244.2.45   scheduler-lab-worker    <none>           <none>
deploy-mem-hungry-657fb577df-8jslm      1/1     Running     1 (9s ago)    27s    10.244.3.22   scheduler-lab-worker3   <none>           <none>
deploy-mem-hungry-657fb577df-9kmtn      0/1     OOMKilled   1 (14s ago)   27s    10.244.3.23   scheduler-lab-worker3   <none>           <none>
deploy-mem-hungry-657fb577df-cb2fp      1/1     Running     0             27s    10.244.2.51   scheduler-lab-worker    <none>           <none>
deploy-mem-matched-5c49fb5d9d-64grg     1/1     Running     0             88s    10.244.2.48   scheduler-lab-worker    <none>           <none>
deploy-mem-matched-5c49fb5d9d-ddhwk     1/1     Running     0             88s    10.244.2.47   scheduler-lab-worker    <none>           <none>
deploy-mem-matched-5c49fb5d9d-pthmd     1/1     Running     0             88s    10.244.2.49   scheduler-lab-worker    <none>           <none>
deploy-over-prov-cpu-5b74dd784d-7hrt7   1/1     Running     0             73s    10.244.1.30   scheduler-lab-worker2   <none>           <none>
deploy-over-prov-cpu-5b74dd784d-nldqw   1/1     Running     0             73s    10.244.1.29   scheduler-lab-worker2   <none>           <none>
deploy-over-prov-cpu-5b74dd784d-t577k   1/1     Running     0             73s    10.244.2.50   scheduler-lab-worker    <none>           <none>
deploy-over-prov-mem-69ffc6cdd4-dhjz7   1/1     Running     0             57s    10.244.1.31   scheduler-lab-worker2   <none>           <none>
deploy-over-prov-mem-69ffc6cdd4-l554h   1/1     Running     0             57s    10.244.1.33   scheduler-lab-worker2   <none>           <none>
deploy-over-prov-mem-69ffc6cdd4-ttjbr   1/1     Running     0             57s    10.244.1.32   scheduler-lab-worker2   <none>           <none>
```
