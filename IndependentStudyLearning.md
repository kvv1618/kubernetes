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
    - Build the Kubernetes scheduler binary with the plugin included.
        - `make WHAT=cmd/kube-scheduler KUBE_BUILD_PLATFORM=linux/arm64`.

- Move the built binary to the appropriate location (into `kdaptManifests/`), from `_output/local/bin/arm64/kube-scheduler`.
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
- The default enabled pluigins: https://kubernetes.io/docs/reference/scheduling/config/#scheduling-plugins

## In-detail analysis of the plugin code:
- The plugin is implemented in the `kdapt.go` file, which defines the `Kdapt` struct and implements the Kubernetes scheduler plugin interface.
    - Core plugin interface:
         ```
          type plugin interface {
              Name() string
          }
        ```
- Few types of plugins:
    - PreFilterPlugin:
        - This plugin type is responsible for performing pre-filtering logic before the scheduling process begins. It implements the `PreFilter` method, which takes in the pod and node information and returns a status indicating whether the pod can be scheduled on the node or not.
    - FilterPlugin: 
        - This plugin type is responsible for filtering nodes based on custom logic. It implements the `Filter` method, which takes in the pod and node information and returns a status indicating whether the pod can be scheduled on the node or not.
    - ScorePlugin: 
        - This plugin type is responsible for scoring nodes based on custom logic. It implements the `Score` method, which takes in the pod and node information and returns a score for each node.
            ```
            type ScorePlugin interface {
                plugin
                Score(ctx context.Context, state *CycleState, pod *v1.Pod, nodeinfo *NodeInfo) (int64, *Status)
                ScoreExtensions() ScoreExtensions
            }
            ```
        - CycleState is a data structure that holds information about the current scheduling cycle. It allows plugins to share information and maintain state across different stages of the scheduling process.
        - Pod and NodeInfo are data structures that represent the pod being scheduled and the node being evaluated, respectively.
- The fwk.Handle interface:
    - The `fwk.Handle` interface is used to interact with the Kubernetes scheduler framework. It provides access to various scheduler components, such as the node informer, pod informer, shared scheduling state, and other plugins.
- The framework ensures that:
    - It instantiates the plugin via the New() constructor.
    - It passes a valid fwk.Handle to the plugin.
    - Plugin can now call framework methods safely.
