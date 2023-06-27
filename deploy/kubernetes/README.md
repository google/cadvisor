# cAdvisor Kubernetes Daemonset

cAdvisor uses [Kustomize](https://github.com/kubernetes-sigs/kustomize) to manage Kubernetes manifests. See the [Install Kustomize](https://kubectl.docs.kubernetes.io/installation/kustomize/) for installation instructions, and for a description of how it works.

## Deploy

Pick a [cAdvisor release](https://github.com/google/cadvisor/releases)
```
VERSION=v0.42.0
```

Deploy to Kubernetes cluster with [remote build](https://github.com/kubernetes-sigs/kustomize/blob/master/examples/remoteBuild.md):
```
kustomize build "https://github.com/google/cadvisor/deploy/kubernetes/base?ref=${VERSION}" | kubectl apply -f -
```

## Usage

To update the image version([reference](https://github.com/kubernetes-sigs/kustomize/blob/master/examples/image.md)):
```
cd deploy/kubernetes/base && kustomize edit set image gcr.io/cadvisor/cadvisor:${VERSION} && cd ../../..
```

To generate the base daemonset:
```
kubectl kustomize deploy/kubernetes/base
```

To apply the base daemonset to your cluster:
```
kubectl kustomize deploy/kubernetes/base | kubectl apply -f -
```

To generate the daemonset with example patches applied:
```
kubectl kustomize deploy/kubernetes/overlays/examples
```

To apply the daemonset to your cluster with example patches applied:
```
kubectl kustomize deploy/kubernetes/overlays/examples | kubectl apply -f -
```

### cAdvisor with perf support on Kubernetes

Example of modifications needed to deploy cAdvisor with perf support is provided in [overlays/examples_perf](overlays/examples_perf) directory (modification to daemonset and configmap with perf events configuration).

To generate and apply the daemonset with patches for cAdvisor with perf support:
```
kubectl kustomize deploy/kubernetes/overlays/examples_perf | kubectl apply -f -
```

## Kustomization

On your own fork of cAdvisor, create your own overlay directoy with your patches.  Copy patches from the example folder if you intend to use them, but don't modify the originals.  Commit your changes in your local branch, and use git to manage them the same way you would any other piece of code.

To run the daemonset with your patches applied:
```
kubectl kustomize deploy/kubernetes/overlays/<my_custom_overlays> | kubectl apply -f -
```

To get changes made to the upstream cAdvisor daemonset, simply rebase your fork of cAdvisor on top of upstream.  Since you didn't make changes to the originals, you won't have any conflicts.
