# Kind cluster
To create a local Kubernetes cluster using Kind, follow these steps:
- Install Kind
- Since we require multiple nodes to test our scheduler, we will create a cluster with 3 worker nodes. Run the following command to create a Kind cluster with 3 worker nodes:
```bash
kind create cluster --name scheduler-lab --config kind-config.yaml
```
- We need to enable a metrics server to monitor the cluster's performance. Run the following command to enable the metrics server:
    - metrics-server is a lightweight kubernetes system component that collects resource usage data from every node and pod
    - It gathers information from each Kubelet and exposes it through the Kubernetes API server, allowing users to query resource usage and make informed decisions about scaling and resource allocation.
```bash
kubectl apply -f https://github.com/kubernetes-sigs/metrics-server/releases/latest/download/components.yaml
```
- Patch the metrics server to work with Kind by running the following command:
    - This command modifies the deployment of the metrics server to include the `--kubelet-insecure-tls` argument, which allows the metrics server to communicate with the kubelets without verifying their TLS certificates. This is necessary because Kind uses self-signed certificates for its kubelets, which can cause issues with secure communication.
```bash
kubectl patch deployment metrics-server -n kube-system --type='json' -p='[{"op": "add", "path": "/spec/template/spec/containers/0/args/-", "value": "--kubelet-insecure-tls"}]'
```
- This command should print the node names along with their CPU and memory usage, indicating that the metrics server is functioning correctly.
```bash
kubectl top nodes
```