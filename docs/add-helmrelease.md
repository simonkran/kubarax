# Add HelmRelease (ApplicationSet Equivalent)

In kubara, an `ApplicationSet` deploys a Helm chart to one or more clusters with per-cluster configuration. In kubarax, the equivalent is a **HelmRelease** combined with a **ResourceSet** for multi-cluster deployment.

## Prerequisites

- The required Helm chart is available (either in `managed-service-catalog/` or a registered `HelmRepository`)
- Any required Git/Helm repositories have been [added](add-repository.md)
- Tenant namespaces have been [configured](add-project.md) if needed

## Single-Cluster: Add a HelmRelease

### Step 1: Add the Helm Chart to the Managed Catalog

Place your umbrella chart in the managed service catalog:

```
managed-service-catalog/helm/my-new-service/
  Chart.yaml
  values.yaml
```

Example `Chart.yaml`:

```yaml
apiVersion: v2
name: my-new-service
description: My new platform service
version: 0.1.0
type: application
dependencies:
  - name: my-new-service
    version: "1.x"
    repository: "https://charts.example.com"
```

### Step 2: Add Override Values

Create cluster-specific overrides:

```bash
mkdir -p customer-service-catalog/helm/<cluster-name>/my-new-service/
```

```yaml
# customer-service-catalog/helm/<cluster-name>/my-new-service/values.yaml
my-new-service:
  replicas: 2
  ingress:
    enabled: true
    hostname: my-service.platform.example.com
```

### Step 3: Create the HelmRelease

Add to `customer-service-catalog/helm/<cluster-name>/helmreleases/`:

```yaml
# customer-service-catalog/helm/<cluster-name>/helmreleases/my-new-service.yaml
apiVersion: helm.toolkit.fluxcd.io/v2
kind: HelmRelease
metadata:
  name: my-new-service
  namespace: flux-system
spec:
  interval: 10m
  chart:
    spec:
      chart: ./managed-service-catalog/helm/my-new-service
      sourceRef:
        kind: GitRepository
        name: flux-system
      interval: 5m
  targetNamespace: my-new-service
  install:
    createNamespace: true
    remediation:
      retries: 3
  upgrade:
    remediation:
      retries: 3
  dependsOn:
    - name: cert-manager    # if it needs TLS
    - name: traefik          # if it needs ingress
  values: {}
```

### Step 4: Add to Kustomization

```yaml
# customer-service-catalog/helm/<cluster-name>/kustomization.yaml
resources:
  # ... existing resources ...
  - helmreleases/my-new-service.yaml
```

### Step 5: Commit and Push

```bash
git add .
git commit -m "feat: add my-new-service HelmRelease"
git push
```

## Multi-Cluster: Deploy with ResourceSet

To deploy a HelmRelease to multiple clusters (like kubara's ApplicationSet with cluster generators):

### Option A: Static Inputs

```yaml
apiVersion: fluxcd.controlplane.io/v1
kind: ResourceSet
metadata:
  name: my-new-service-fleet
  namespace: flux-system
spec:
  inputs:
    - id: worker-0
      cluster: worker-0
      kubeconfigSecret: worker-0-kubeconfig
      replicas: "2"
    - id: worker-1
      cluster: worker-1
      kubeconfigSecret: worker-1-kubeconfig
      replicas: "3"
  resourcesTemplate: |
    ---
    apiVersion: helm.toolkit.fluxcd.io/v2
    kind: HelmRelease
    metadata:
      name: my-new-service-<< inputs.id >>
      namespace: flux-system
    spec:
      interval: 10m
      chart:
        spec:
          chart: ./managed-service-catalog/helm/my-new-service
          sourceRef:
            kind: GitRepository
            name: flux-system
      targetNamespace: my-new-service
      install:
        createNamespace: true
      kubeConfig:
        secretRef:
          name: << inputs.kubeconfigSecret >>
      values:
        replicas: << inputs.replicas | int >>
  dependsOn:
    - apiVersion: fluxcd.controlplane.io/v1
      kind: FluxInstance
      name: flux
      namespace: flux-system
      ready: true
```

### Option B: Dynamic Inputs via ResourceSetInputProvider

Use a `ResourceSetInputProvider` to dynamically discover target clusters:

```yaml
apiVersion: fluxcd.controlplane.io/v1
kind: ResourceSetInputProvider
metadata:
  name: worker-clusters
  namespace: flux-system
spec:
  type: ExternalService
  url: https://cluster-registry.internal/api/clusters
  filter:
    labels: ["my-new-service=enabled"]
  defaultValues:
    kubeconfigSecretPrefix: "kubeconfig-"
---
apiVersion: fluxcd.controlplane.io/v1
kind: ResourceSet
metadata:
  name: my-new-service-fleet
  namespace: flux-system
spec:
  inputsFrom:
    - apiVersion: fluxcd.controlplane.io/v1
      kind: ResourceSetInputProvider
      name: worker-clusters
  resourcesTemplate: |
    # ... same template as above ...
```

## External Helm Chart (Not in Git)

To deploy a chart directly from a `HelmRepository` (like kubara's multi-source ApplicationSet):

```yaml
apiVersion: helm.toolkit.fluxcd.io/v2
kind: HelmRelease
metadata:
  name: redis
  namespace: flux-system
spec:
  interval: 10m
  chart:
    spec:
      chart: redis
      version: "18.x"
      sourceRef:
        kind: HelmRepository
        name: bitnami
      interval: 1h
  targetNamespace: redis
  install:
    createNamespace: true
  values:
    architecture: standalone
    auth:
      enabled: true
```

## Dependency Ordering

Use `dependsOn` to control deployment order (equivalent to ArgoCD sync waves):

```yaml
spec:
  dependsOn:
    - name: cert-manager          # Must be ready before this service
    - name: external-secrets      # Secrets must be synced first
    - name: kube-prometheus-stack  # Monitoring must be available
```

## Comparison with kubara

| kubara ApplicationSet | kubarax equivalent |
|---|---|
| Single ApplicationSet YAML | `HelmRelease` + `ResourceSet` for multi-cluster |
| Cluster generator + label selectors | `ResourceSetInputProvider` with filters |
| `values.yaml` + `additional-values.yaml` | `values:` inline + `valuesFrom:` ConfigMap/Secret |
| Sync waves for ordering | `dependsOn` on HelmRelease |
| Chart in managed catalog | Chart in `managed-service-catalog/helm/` |
| Per-cluster overlay values | Per-cluster directory in `customer-service-catalog/helm/` |
