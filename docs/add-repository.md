# Add Repository

To deploy applications with FluxCD, you need to register Git repositories and Helm repositories as Flux source objects.

## Recommended: Declarative via config.yaml

The simplest way to add repositories is to declare them in `config.yaml` and run `kubarax generate --helm`.

### Git Repositories

Add Git repository sources to your cluster configuration:

```yaml
clusters:
  - name: my-cluster
    # ... existing config ...
    fluxcd:
      # ... existing fluxcd config ...
      gitRepositories:
        - name: my-app-repo
          url: https://github.com/org/my-app-repo
          branch: main
          secretRef: my-app-repo-credentials
        - name: team-configs
          url: https://github.com/org/team-configs
          branch: develop
          interval: 10m
```

**Fields:**
| Field | Required | Default | Description |
|-------|----------|---------|-------------|
| `name` | yes | — | Resource name for the GitRepository CR |
| `url` | yes | — | Git repository HTTPS URL |
| `branch` | no | `main` | Branch to track |
| `secretRef` | no | — | Name of K8s Secret with credentials |
| `interval` | no | `5m` | Reconciliation interval |

Then generate:

```bash
kubarax generate --helm
```

This creates `GitRepository` CRs in `customer-service-catalog/helm/<cluster-name>/flux-system/gitrepositories.yaml` and adds them to the kustomization automatically.

### Helm Repositories

Helm repositories are also declared in `config.yaml`:

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

### Credentials

For repositories requiring authentication, create a Kubernetes Secret before or alongside your repository declaration:

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

## Advanced: Manual Repository Creation

For cases not covered by the declarative approach, you can manually create repository manifests.

### Git Repository

Add to `customer-service-catalog/helm/<cluster-name>/sources/`:

```yaml
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

Then add to `customer-service-catalog/helm/<cluster-name>/kustomization.yaml`:

```yaml
resources:
  # ... existing resources ...
  - sources/my-app-repo.yaml
```

### SSH Git Repository

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
