# cAdvisor Kubernetes Daemonset

cAdvisor uses [Kustomize](https://github.com/kubernetes-sigs/kustomize) to manage kubernetes yaml files.  See the [Kustomize](https://github.com/kubernetes-sigs/kustomize) readme for installation instructions, and for a description of how it works.

## Usage

From the cadvisor/ directory, to generate the base daemonset:
```
kustomize build deploy/kubernetes/base
```

To apply the base daemonset to your cluster:
```
kustomize build deploy/kubernetes/base | kubectl apply -f -
```

To generate the daemonset with example patches applied:
```
kustomize build deploy/kubernetes/overlays/examples
```

To apply the daemonset to your cluster with example patches applied:
```
kustomize build deploy/kubernetes/overlays/examples | kubectl apply -f -
```

### cAdvisor with perf support on Kubernetes

Example of modifications needed to deploy cAdvisor with perf support is provided in [overlays/examples_perf](overlays/examples_perf) directory (modification to daemonset and configmap with perf events configuration).

To generate and apply the daemonset with patches for cAdvisor with perf support:
```
kustomize build deploy/kubernetes/overlays/examples_perf | kubectl apply -f -
```

## Kustomization

On your own fork of cAdvisor, create your own overlay directoy with your patches.  Copy patches from the example folder if you intend to use them, but don't modify the originals.  Commit your changes in your local branch, and use git to manage them the same way you would any other piece of code.

To run the daemonset with your patches applied:
```
kustomize build deploy/kubernetes/overlays/<my_custom_overlays> | kubectl apply -f -
```

To get changes made to the upstream cAdvisor daemonset, simply rebase your fork of cAdvisor on top of upstream.  Since you didn't make changes to the originals, you won't have any conflicts.
