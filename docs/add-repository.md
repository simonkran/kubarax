# Add Repository

To deploy applications with FluxCD, you need to register Git repositories and Helm repositories as Flux source objects. This is the kubarax equivalent of adding a repository in kubara's ArgoCD configuration.

## Add a Git Repository

### Step 1: Store Credentials

Create a Kubernetes secret with the repository credentials:

```bash
kubectl create secret generic my-app-repo-credentials \
  --from-literal=username=git \
  --from-literal=password=ghp_your_token_here \
  -n flux-system
```

For production, use External Secrets to sync from your vault:

```yaml
apiVersion: external-secrets.io/v1beta1
kind: ExternalSecret
metadata:
  name: my-app-repo-credentials
  namespace: flux-system
spec:
  refreshInterval: 1h
  secretStoreRef:
    kind: ClusterSecretStore
    name: controlplane-vault
  target:
    name: my-app-repo-credentials
  data:
    - secretKey: username
      remoteRef:
        key: git/my-app-repo
        property: username
    - secretKey: password
      remoteRef:
        key: git/my-app-repo
        property: pat
```

### Step 2: Create a GitRepository Source

Add to `customer-service-catalog/helm/<cluster-name>/sources/`:

```yaml
# customer-service-catalog/helm/<cluster-name>/sources/my-app-repo.yaml
apiVersion: source.toolkit.fluxcd.io/v1
kind: GitRepository
metadata:
  name: my-app-repo
  namespace: flux-system
spec:
  interval: 5m
  url: https://github.com/org/my-app-repo
  ref:
    branch: main
  secretRef:
    name: my-app-repo-credentials
```

### Step 3: Add to Kustomization

```yaml
# customer-service-catalog/helm/<cluster-name>/kustomization.yaml
resources:
  # ... existing resources ...
  - sources/my-app-repo.yaml
```

### Step 4: Commit and Push

```bash
git add .
git commit -m "feat: add my-app-repo git source"
git push
```

## Add a Helm Repository

### HTTPS Helm Repository

```yaml
apiVersion: source.toolkit.fluxcd.io/v1
kind: HelmRepository
metadata:
  name: bitnami
  namespace: flux-system
spec:
  interval: 1h
  url: https://charts.bitnami.com/bitnami
```

### OCI Helm Repository

```yaml
apiVersion: source.toolkit.fluxcd.io/v1
kind: HelmRepository
metadata:
  name: my-oci-registry
  namespace: flux-system
spec:
  interval: 1h
  url: oci://ghcr.io/my-org/charts
  type: oci
  secretRef:
    name: oci-registry-credentials
```

### Private Helm Repository

```yaml
apiVersion: source.toolkit.fluxcd.io/v1
kind: HelmRepository
metadata:
  name: private-charts
  namespace: flux-system
spec:
  interval: 1h
  url: https://charts.internal.example.com
  secretRef:
    name: helm-repo-credentials
---
apiVersion: v1
kind: Secret
metadata:
  name: helm-repo-credentials
  namespace: flux-system
type: Opaque
stringData:
  username: chart-reader
  password: your-password-here
```

## Using Repositories in config.yaml

You can also declare additional Helm repositories in `config.yaml`:

```yaml
fluxcd:
  helmRepositories:
    - name: bitnami
      url: https://charts.bitnami.com/bitnami
    - name: my-oci-registry
      url: oci://ghcr.io/my-org/charts
      type: oci
      secretRef: oci-registry-credentials
```

These will be generated automatically by `kubarax generate --helm` into the `helmrepositories.yaml` file.

## SSH Git Repository

For SSH-based Git access:

```bash
kubectl create secret generic my-ssh-repo-credentials \
  --from-file=identity=./deploy-key \
  --from-file=known_hosts=./known_hosts \
  -n flux-system
```

```yaml
apiVersion: source.toolkit.fluxcd.io/v1
kind: GitRepository
metadata:
  name: my-ssh-repo
  namespace: flux-system
spec:
  interval: 5m
  url: ssh://git@github.com/org/my-repo.git
  ref:
    branch: main
  secretRef:
    name: my-ssh-repo-credentials
```

## Comparison with kubara

| kubara (ArgoCD) | kubarax (FluxCD) |
|---|---|
| Repository credential in ArgoCD config | `GitRepository` / `HelmRepository` CR + `secretRef` |
| Repository scoped to AppProject | Repository in tenant namespace (multi-tenancy mode) |
| HTTPS + PAT auth | HTTPS + PAT via Kubernetes Secret |
| SSH auth | SSH key via Kubernetes Secret |
| OCI Helm repos | `HelmRepository` with `type: oci` |
