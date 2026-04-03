# Add Worker Cluster

This guide walks through adding a new Kubernetes worker cluster (spoke) to your kubarax-managed platform. The control plane cluster (hub) runs the Flux Operator and deploys platform components to worker clusters remotely.

## Prerequisites

- A running control plane cluster with kubarax bootstrapped
- Access to the worker cluster's kubeconfig
- A secret backend (Vault, AWS Secrets Manager, etc.) if using External Secrets

## Step 1: Update config.yaml

Add a new cluster entry with `type: worker`:

```yaml
clusters:
  - name: controlplane-0
    stage: prod
    type: controlplane
    dnsName: cp.platform.example.com
    # ... existing controlplane config ...

  - name: worker-0
    stage: dev
    type: worker
    dnsName: worker-0.platform.example.com
    ingressClassName: traefik
    fluxcd:
      distribution:
        version: "2.x"
        registry: ghcr.io/fluxcd
      cluster:
        type: kubernetes
        size: small
      sync:
        kind: GitRepository
        url: https://github.com/org/platform-repo
        ref: refs/heads/main
        path: clusters/worker-0
        interval: 5m
    services:
      traefik:
        status: enabled
      certManager:
        status: enabled
      externalDns:
        status: enabled
      externalSecrets:
        status: enabled
      kubePrometheusStack:
        status: enabled
      loki:
        status: enabled
      metricsServer:
        status: enabled
      kyverno:
        status: disabled
      kyvernoPolicies:
        status: disabled
      kyvernoPolicyReporter:
        status: disabled
      oauth2Proxy:
        status: disabled
      longhorn:
        status: disabled
      metallb:
        status: disabled
      fluxWebUI:
        status: disabled
      homeDashboard:
        status: enabled
      forgejo:
        status: disabled
```

## Step 2: Generate Templates

```bash
kubarax generate --helm
```

This creates:
- `managed-service-catalog/helm/` - Helm chart wrappers (shared across clusters)
- `customer-service-catalog/helm/worker-0/` - Worker-specific FluxCD resources:
  - `flux-operator/fluxinstance.yaml` - FluxInstance config for the worker
  - `flux-operator/resourceset-platform.yaml` - ResourceSet for platform deployment
  - `helmreleases/*.yaml` - HelmReleases for each enabled service
  - `kustomization.yaml` - Kustomize aggregator

## Step 3: Store the Worker Kubeconfig

The control plane cluster needs access to the worker cluster via a kubeconfig secret.

### Option A: Direct Secret (simple setups)

Create a secret in the `flux-system` namespace on the control plane:

```bash
kubectl create secret generic worker-0-kubeconfig \
  --from-file=value=./worker-0-kubeconfig.yaml \
  -n flux-system \
  --context=controlplane-0
```

### Option B: External Secrets (production)

Store the kubeconfig in your secret backend (Vault, AWS Secrets Manager, etc.), then create an `ExternalSecret` that syncs it:

```yaml
apiVersion: external-secrets.io/v1beta1
kind: ExternalSecret
metadata:
  name: worker-0-kubeconfig
  namespace: flux-system
spec:
  refreshInterval: 1h
  secretStoreRef:
    kind: ClusterSecretStore
    name: controlplane-vault
  target:
    name: worker-0-kubeconfig
  data:
    - secretKey: value
      remoteRef:
        key: k8s/worker-0
        property: kubeconfig
```

## Step 4: Register Worker in Controlplane Config

Add the worker cluster to the controlplane's `workerClusters` list in `config.yaml`:

```yaml
clusters:
  - name: controlplane-0
    stage: prod
    type: controlplane
    # ... existing controlplane config ...
    workerClusters:
      - name: worker-0
        kubeconfigSecret: worker-0-kubeconfig
```

Then regenerate:

```bash
kubarax generate --helm
```

This automatically populates the `resourceset-worker-clusters.yaml` on the control plane with the correct inputs and `resourcesTemplate`. The `kubeConfig.secretRef` tells FluxCD to deploy resources to the remote worker cluster.

## Step 5: Commit and Push

```bash
git add .
git commit -m "feat: add worker-0 cluster"
git push
```

The Flux Operator on the control plane will:
1. Detect the new ResourceSet input
2. Create a `Kustomization` targeting the worker cluster
3. Deploy all enabled HelmReleases to the worker via `kubeConfig.secretRef`

## Step 6: Verify

```bash
# On the control plane - check ResourceSet status
kubectl get resourceset worker-clusters -n flux-system

# Check the generated Kustomization for the worker
kubectl get kustomizations -n flux-system

# Check HelmReleases targeting the worker
kubectl get helmreleases -n flux-system

# On the worker cluster - verify components are running
kubectl get pods -A --context=worker-0
```

## Notes

- Worker clusters do **not** need the Flux Operator installed — the control plane manages them remotely
- Each worker gets its own set of HelmReleases with independent `values.yaml` for customization
- Use `additional-values.yaml` in the customer catalog for cluster-specific overrides that survive regeneration
- The `dependsOn` on the ResourceSet ensures the FluxInstance is ready before deploying to workers
