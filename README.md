# nfspvc-operator

The `nfspvc-operator` is an operator that reconciles `NfsPvc` CRs.

`NfsPvc` provides a simple interface (CRD) to create `PVC` and `PV` in a Kubernetes cluster with an NFS backend, allowing you to connect pre-existing NFS storage to your workloads without using any CSI driver.

This is useful when you have an already-created NFS, which you would like to mount in different places (Kubernetes Pod, Virtual Machine, etc...).

## How to Use

An example `NfsPvc` CR looks as follows:

```yaml
apiVersion: nfspvc.dana.io/v1alpha1
kind: NfsPvc
metadata:
  name: test
  namespace: test
spec:
  accessModes:
    - ReadWriteOnce
  capacity:
    storage: 200Gi
  path: /test
  server: vs-nas-test
```

The supported `accessModes` are the same as those of the `PVC` resource: `ReadWriteOnce`, `ReadOnlyMany`, `ReadWriteMany` and `ReadWriteOncePod`

### Status

The status of a `NfsPvc` resource shows the status of the `PVC` and `PV` it creates. For example:

```yaml
...
status:
  pvPhase: Bound
  pvcPhase: Bound
```

### Lifecycle

Once a `NfsPvc` CR is created, then corresponding `PVC` and `PV` objects are created. When the CR is removed, then the `PVC` and `PV` objects are removed. The `ReclaimPolicy` is [defined by the `configuration-nfspvc` `ConfigMap`](#how-to-deploy).

If the underlying `PVC` or `PV` is deleted but the corresponding `NfsPvc` still exists, then the operator will re-create the `PVC` or `PV`.

## How to Deploy

### Config

The controller makes use of a `ConfigMap` of the name `configuration-nfspvc` with default values for `StorageClass` and `ReclaimPolicy`:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: configuration-nfspvc
  namespace: system
data:
  STORAGE_CLASS: brown
  RECLAIM_POLICY: Retain
```

### Deploying the controller

```bash
$ make deploy IMG=ghcr.io/dana-team/nfspvc-operator:<release>
```

#### Build your own image

```bash
$ make docker-build docker-push IMG=<registry>/nfspvc-operator:<tag>
```
