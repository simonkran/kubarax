# kubarax

A framework and bootstrapping tool for building and operating a production-grade Kubernetes platform with **FluxCD**.

Inspired by [kubara](https://github.com/kubara-io/kubara) (which uses ArgoCD), kubarax translates the same platform engineering concepts to a FluxCD-native approach.

## Architecture

kubarax follows a **hub-and-spoke** model:

- **Control plane cluster (hub)**: Runs FluxCD, manages all platform components and spoke clusters
- **Worker clusters (spokes)**: Managed remotely via FluxCD's `spec.kubeConfig.secretRef`

### FluxCD Mapping (vs ArgoCD in kubara)

| kubara (ArgoCD) | kubarax (FluxCD) |
|---|---|
| `Application` | `HelmRelease` + `Kustomization` |
| `ApplicationSet` | Per-cluster `Kustomization` with overlays |
| `AppProject` | Namespace RBAC + ServiceAccount |
| ArgoCD cluster secrets | `spec.kubeConfig.secretRef` on resources |
| ArgoCD repo credentials | `GitRepository` / `HelmRepository` + `secretRef` |
| ArgoCD bootstrap | FluxCD bootstrap (install controllers + initial sources) |
| Sync waves | `dependsOn` chains |
| ArgoCD UI | Weave GitOps UI |

## Quick Start

### 1. Initialize

```bash
# Generate .gitignore and example .env
kubarax init --prep

# Fill in the .env file with your values
vim .env

# Generate config.yaml from environment
kubarax init
```

### 2. Generate

```bash
# Generate all FluxCD manifests (Helm charts + Kustomizations)
kubarax generate --helm

# Or preview without writing
kubarax generate --helm --dry-run
```

### 3. Bootstrap

```bash
# Bootstrap FluxCD onto your control plane cluster
kubarax bootstrap my-cluster --with-es-crds --with-prometheus-crds
```

## Commands

| Command | Description |
|---|---|
| `kubarax init --prep` | Generate `.gitignore` and example `.env` |
| `kubarax init` | Create `config.yaml` from environment variables |
| `kubarax generate` | Generate FluxCD manifests from templates |
| `kubarax bootstrap <cluster>` | Bootstrap FluxCD onto a cluster |
| `kubarax schema` | Generate JSON schema for `config.yaml` validation |

## Generated Structure

```
managed-service-catalog/
  helm/
    cert-manager/        # Upstream Helm chart wrappers
    traefik/
    external-secrets/
    kube-prometheus-stack/
    loki/
    ...

customer-service-catalog/
  helm/
    <cluster-name>/
      flux-system/
        gitrepository.yaml     # FluxCD GitRepository source
        kustomization.yaml     # FluxCD root Kustomization
        helmrepositories.yaml  # Upstream Helm repos
      helmreleases/
        cert-manager.yaml      # HelmRelease per service
        traefik.yaml
        ...
      kustomization.yaml       # Kustomize aggregator
```

## Platform Components

- **Ingress**: Traefik
- **Certificates**: cert-manager
- **DNS**: ExternalDNS
- **Secrets**: External Secrets Operator
- **Monitoring**: kube-prometheus-stack, Metrics Server
- **Logging**: Grafana Loki
- **Policy**: Kyverno
- **Auth**: OAuth2 Proxy
- **Storage**: Longhorn
- **Load Balancer**: MetalLB
- **GitOps UI**: Weave GitOps
- **Dashboard**: Homer
- **Git Service**: Forgejo

## Configuration

See `config.yaml` structure:

```yaml
clusters:
  - name: my-platform
    stage: prod
    type: controlplane
    dnsName: platform.example.com
    fluxcd:
      gitRepository:
        url: https://github.com/org/platform-repo
        branch: main
      interval: 5m
    services:
      traefik:
        status: enabled
      certManager:
        status: enabled
      # ... more services
```

## Building

```bash
make build    # Build binary
make test     # Run tests
make lint     # Run linter
```

## License

Apache 2.0
