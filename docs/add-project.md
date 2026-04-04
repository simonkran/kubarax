# Add Project

In kubara (ArgoCD), an `AppProject` provides logical tenancy — controlling who can deploy what and where. FluxCD does not have an equivalent CRD, so kubarax implements project-level tenancy using **Kubernetes namespaces, RBAC, and ServiceAccounts**.

## Concept

A "project" in kubarax is:
- A **namespace** on the target cluster that scopes a tenant's resources
- A **ServiceAccount** with restricted RBAC permissions
- A **Flux Kustomization** that reconciles the tenant's resources with the scoped ServiceAccount

## Add a Project via config.yaml (Recommended)

Add projects to your cluster definition in `config.yaml`:

```yaml
clusters:
  - name: my-cluster
    # ... existing fields ...
    projects:
      - name: team-alpha
      - name: team-beta
        clusterRole: edit    # optional, defaults to cluster-admin
```

Then regenerate manifests:

```bash
kubarax generate --helm
```

This generates:
- `customer-service-catalog/helm/<cluster-name>/tenants/tenants.yaml` — Namespace, ServiceAccount, and RoleBinding for each project
- `customer-service-catalog/helm/<cluster-name>/tenants/kustomizations.yaml` — Flux Kustomization for each project

The cluster's `kustomization.yaml` automatically includes the tenant resources when projects are defined.

Commit and push:

```bash
git add .
git commit -m "feat: add tenant projects"
git push
```

### Configuration Reference

| Field | Required | Default | Description |
|---|---|---|---|
| `name` | yes | — | Project/tenant name (used as namespace) |
| `clusterRole` | no | `cluster-admin` | ClusterRole to bind to the tenant ServiceAccount |

## Add a Project Manually (Advanced)

For advanced customization (e.g., Kyverno policies, ResourceQuota), you can create tenant manifests manually.

### Step 1: Create a Tenant Namespace Manifest

Add a tenant namespace definition to your customer catalog:

```bash
mkdir -p customer-service-catalog/helm/<cluster-name>/tenants/
```

Create `customer-service-catalog/helm/<cluster-name>/tenants/team-alpha.yaml`:

```yaml
apiVersion: v1
kind: Namespace
metadata:
  name: team-alpha
  labels:
    kubarax.io/tenant: team-alpha
    kubarax.io/project: team-alpha
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: flux-team-alpha
  namespace: team-alpha
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: flux-team-alpha
  namespace: team-alpha
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: cluster-admin
subjects:
  - kind: ServiceAccount
    name: flux-team-alpha
    namespace: team-alpha
```

### Step 2: Create a Flux Kustomization for the Tenant

Add a `Kustomization` in flux-system that reconciles the tenant's resources with the scoped ServiceAccount:

```yaml
# customer-service-catalog/helm/<cluster-name>/tenants/team-alpha-kustomization.yaml
apiVersion: kustomize.toolkit.fluxcd.io/v1
kind: Kustomization
metadata:
  name: tenant-team-alpha
  namespace: flux-system
spec:
  interval: 5m
  sourceRef:
    kind: GitRepository
    name: flux-system
  path: ./tenants/team-alpha
  prune: true
  serviceAccountName: flux-team-alpha
  targetNamespace: team-alpha
```

The `serviceAccountName` restricts what this Kustomization can do — it can only manage resources that the ServiceAccount has permissions for.

### Step 3: Restrict Source Repositories (Optional)

To limit which Git repositories a tenant can deploy from, use the Flux Operator's multi-tenancy features:

```yaml
# Enable multi-tenancy in config.yaml
fluxcd:
  cluster:
    multitenant: true  # defaults to false
```

With multi-tenancy enabled, each tenant's `Kustomization` and `HelmRelease` resources can only reference sources within their own namespace. Other `fluxcd.cluster` fields (`type`, `size`, `networkPolicy`) have sensible defaults and can be overridden here as well.

### Step 4: Add Kyverno Policies (Optional)

If Kyverno is enabled, add policies to enforce tenant constraints:

```yaml
# customer-service-catalog/helm/<cluster-name>/tenants/team-alpha-policies.yaml
apiVersion: kyverno.io/v1
kind: ClusterPolicy
metadata:
  name: restrict-team-alpha-registries
spec:
  validationFailureAction: Enforce
  rules:
    - name: restrict-image-registries
      match:
        resources:
          namespaces:
            - team-alpha
      validate:
        message: "Images must come from approved registries."
        pattern:
          spec:
            containers:
              - image: "ghcr.io/team-alpha/*"
```

### Step 5: Add to Kustomization

Add the tenant manifests to your cluster's `kustomization.yaml`:

```yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
  # ... existing resources ...
  - tenants/team-alpha.yaml
  - tenants/team-alpha-kustomization.yaml
```

### Step 6: Commit and Push

```bash
git add .
git commit -m "feat: add tenant project team-alpha"
git push
```

## Comparison with kubara's AppProject

| kubara AppProject feature | kubarax equivalent |
|---|---|
| Allowed source repos | `serviceAccountName` scoping + multi-tenancy mode |
| Allowed destination namespaces | `targetNamespace` on Kustomization |
| Allowed cluster resources | RBAC RoleBinding on ServiceAccount |
| RBAC within ArgoCD | Kubernetes-native RBAC |
| Resource quotas | `ResourceQuota` + `LimitRange` in namespace |
| Policy enforcement | Kyverno policies |
