# Kdapt Scheduler Plugin
_Follow `./Scheduler-lab/Readme.md` for step-by-step instructions to setup KIND cluster for local Kubernetes development and testing._
- What is Kubernetes Scheduler Plugin?
    - Kubernetes Scheduler Plugin is a framework that allows users to extend the functionality of the Kubernetes scheduler by implementing custom scheduling logic. It provides a way to influence the scheduling decisions made by the Kubernetes scheduler, allowing users to implement their own scheduling policies and algorithms.

_Note: The following documentation is based on hands-on implementation on an ARM64 architecture, and may require adjustments for other architectures._

- To build a Kubernetes Scheduler Plugin:
    - Clone the Kubernetes repository from GitHub.
    - Create a new directory for the plugin under the `pkg/scheduler/framework/plugins` directory
    - Implement the necessary interfaces and logic for the plugin.
        - Required dependencies and imports:
            - `context`
                - Standard Go package for managing context in concurrent programming. Allows for cancellation and timeout of operations, and passing request-scoped values across API boundaries.
            - `k8s.io/api/core/v1`
                - Kubernetes API package for core v1 resources, including Pods and Nodes.
            - `k8s.io/apimachinery/pkg/runtime`
                - Kubernetes API machinery package for runtime objects and serialization.
            - `k8s.io/kube-scheduler/framework`
                - Kubernetes scheduler framework package for building scheduler plugins.
    - Add the plugin to the Kubernetes scheduler configuration, in `pkg/scheduler/framework/plugins/registry.go`.
        - Import the plugin package and add it to the registry map with a unique name.
    - Build the Kubernetes scheduler binary with the plugin included. (this is to be run from the root of the Kubernetes repository, where Makefile for the whole project is located)
        - `make WHAT=cmd/kube-scheduler KUBE_BUILD_PLATFORM=linux/arm64`.

- Move the built binary to the appropriate location (into `kdaptManifests/`), from `_output/local/bin/linux/arm64/kube-scheduler`.
- Build and push the Docker image for the plugin, using the provided Dockerfile in `kdaptManifests/`.
    - The dockerfile is a simple linux image that copies the built kube-scheduler binary into it, along with the necessary configuration files for the plugin.
- Deploy the plugin to the Kubernetes cluster:
    - Apply the rbac configuration for the plugin, using `kubectl apply -f rbac.yaml`. This will create the necessary roles and role bindings for the scheduler pod in `kube-system` namespace to run with the necessary permissions.
    - Apply the deployment configuration for the plugin, using `kubectl apply -f kdaptDeployment.yaml`.

## Scheduler Lifecycle:
- _Note: For each enabled plugin that implements scoring, the `New()` method is called once, when the scheduler starts up._
- The scheduler creats a scheduling cycle for each pod that needs to be scheduled. During this cycle, the scheduler goes through stages in this order: (All enabled plugins that implement the stage are called in sequence, and the results are aggregated based on weight to make scheduling decisions.)
    - Prefilter, Filter, Score, NormalizeScore, Reserve, Permit, PreBind, Bind, PostBind.
    - Refer to: https://kubernetes.io/docs/reference/scheduling/config/#extension-points
- The binding cycle is a separate cycle that is responsible for binding the pod to the selected node (The last three stages mentioned above are part of the binding cycle).
- For all the feasible nodes that pass the filtering stage, the scheduler calls the Score() method of the ScorePlugin to assign a score(between 0-100) to each node per pod. The scores are then normalized and aggregated based on the weight assigned to each plugin in the scheduler configuration. The node with the highest score is selected for scheduling the pod.

## Default K8s Scheduling:
- The default enabled plugins: https://kubernetes.io/docs/reference/scheduling/config/#scheduling-plugins
- The default Scheduler spreads the pods across the cluster based on resource requests and availability, and tries to balance the load across nodes.

## Resource Semantics:
- Allocatable vs Capacity:
    - Allocatable is the resources that are available for scheduling, after accounting for the requests of the already running pods.
    - Capacity is the total resources that are available on the node.
    - Allocatable = Capacity - Requests of the already running pods.
- CPU request:
    - _CPU is “compressible”: if a pod wants more CPU than available, it usually gets slowed down (throttled) rather than killed. Work still continues, just with higher latency/lower throughput, i.e. CPU can be time-shared._
    - Reserves schedulable capacity at placement time.
    - Does not mean dedicated exclusive cores by default.
    - Runtime CPU is time-shared across runnable containers.
    - If a container has no CPU limit (or a high one), it can temporarily use more than its request when spare CPU exists.
- Memory request:
    - _Memory is “non-compressible”: if a pod uses more RAM than is available, the kernel can’t “throttle RAM usage” the same way. It leads to memory pressure, and eventually the process/pod may be OOMKilled. It should be physically available at the time of allocation._
    - Also reserves schedulable capacity at placement time.
    - Memory usage is not time-shared like CPU cycles.
- Example:
    - 8-core node, 4 pods each request 2 cores
    - Scheduler sees 2+2+2+2 = 8, so node is “full” for new 2-core requests, irrespective of current CPU usage.
    - At runtime, if 3 pods are mostly idle, 1 pod may use >2 cores if its CPU limit allows it

## In-detail analysis of the plugin code:
- The plugin is implemented in the `kdapt.go` file, which defines the `Kdapt` struct and implements the Kubernetes scheduler plugin interface.
    - Core plugin interface:
         ```go
          type plugin interface {
              Name() string
          }
        ```
- New() method:
    - The `New()` method is a constructor function that initializes the plugin and returns an instance of it. It takes in a `fwk.Handle` as an argument, which is used to interact with the Kubernetes scheduler framework. The `fwk.Handle` provides access to various scheduler components and allows the plugin to call framework methods safely.
- Score() method:
    - The `Score()` method is responsible for scoring nodes based on custom logic. It takes in the context, cycle state, pod information, and node information as arguments, and returns a score for each node along with a status indicating whether the scoring was successful or not.
- ScoreExtensions() method:
    - The `ScoreExtensions()` method is used to indicate whether the plugin implements any score extensions. Score extensions allow the plugin to perform additional logic after scoring, such as normalizing scores or applying weights to scores. If the plugin does not implement any score extensions, it can return nil.
- NormalizeScore() method:
    - `ScoreExtensions()` returns `NormalizeScore()` method, which is used to normalize the scores assigned to nodes by the `Score()` method. 
    - Normalization is the process of adjusting the scores to a common scale, typically between 0 and 100, to ensure that they are comparable across different plugins and nodes.

- Few types of plugins:
    - PreFilterPlugin:
        - This plugin type is responsible for performing pre-filtering logic before the scheduling process begins. It implements the `PreFilter` method, which takes in the pod and node information and returns a status indicating whether the pod can be scheduled on the node or not.
    - FilterPlugin: 
        - This plugin type is responsible for filtering nodes based on custom logic. It implements the `Filter` method, which takes in the pod and node information and returns a status indicating whether the pod can be scheduled on the node or not.
    - ScorePlugin: 
        - This plugin type is responsible for scoring nodes based on custom logic. It implements the `Score` method, which takes in the pod and node information and returns a score for each node.
            ```go
            type ScorePlugin interface {
                plugin
                Score(ctx context.Context, state *CycleState, pod *v1.Pod, nodeinfo *NodeInfo) (int64, *Status)
                ScoreExtensions() ScoreExtensions
            }
            ```
        - CycleState is a data structure that holds information about the current scheduling cycle. It allows plugins to share information and maintain state across different stages of the scheduling process.
        - Pod and NodeInfo are data structures that represent the pod being scheduled and the node being evaluated, respectively.

_Note: A single plugin file can implement multiple plugin types, as long as the corresponding methods from the interface are implemented. For example, a plugin can implement both FilterPlugin and ScorePlugin interfaces._

- Fwk package:
    - The fwk package is one of the most useful packages in the Kubernetes scheduler framework. It provides the necessary interfaces and data structures for building scheduler plugins. It defines the plugin interfaces, the CycleState, and other helper functions and types that are used by the plugins to interact with the scheduler framework.

- The fwk.Handle interface:
    - The `fwk.Handle` interface is used to interact with the Kubernetes scheduler framework. It provides access to various scheduler components, such as the node informer, pod informer, shared scheduling state, and other plugins.
- The framework ensures that:
    - It instantiates the plugin via the New() constructor.
    - It passes a valid fwk.Handle to the plugin.
    - Plugin can now call framework methods safely.
