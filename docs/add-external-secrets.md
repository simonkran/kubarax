# Add External Secrets & ClusterSecretStore

This guide covers how to set up the External Secrets Operator and configure a `ClusterSecretStore` to sync secrets from an external provider (Vault, AWS Secrets Manager, etc.) into your Kubernetes clusters.

## Prerequisites

- External Secrets enabled in `config.yaml`:

```yaml
services:
  externalSecrets:
    status: enabled
```

- A secret backend (Vault, AWS Secrets Manager, GCP Secret Manager, etc.)

## Setting Up a ClusterSecretStore

A `ClusterSecretStore` is a cluster-scoped resource that tells the External Secrets Operator how to connect to your secret backend. There are two ways to configure it:

### Option A: During Bootstrap

Use the `--with-es-css-file` flag to apply a ClusterSecretStore manifest during cluster bootstrap. The file supports go-template + sprig syntax with access to your cluster config fields (`.name`, `.stage`, `.dnsName`, `.services`, etc.).

Create a manifest file:

```yaml
apiVersion: external-secrets.io/v1beta1
kind: ClusterSecretStore
metadata:
  name: {{ .name }}-vault
spec:
  provider:
    vault:
      server: "https://vault.example.com"
      path: "secret"
      auth:
        kubernetes:
          mountPath: "kubernetes"
          role: "{{ .name }}-external-secrets"
```

Then bootstrap with:

```bash
kubarax bootstrap my-cluster --with-es --with-es-css-file ./cluster-secret-store.yaml
```

The manifest is rendered with your cluster configuration and applied after the External Secrets Operator is installed.

### Option B: Via GitOps (Recommended for Ongoing Management)

Define ClusterSecretStores declaratively in your customer overlay values. After running `kubarax generate --helm`, edit the generated values file at:

```
customer-service-catalog/helm/<cluster-name>/external-secrets/values.yaml
```

Add your ClusterSecretStore configuration:

```yaml
clusterSecretStores:
  controlplane-vault:
    provider:
      vault:
        server: "https://vault.example.com"
        path: "secret"
        auth:
          kubernetes:
            mountPath: "kubernetes"
            role: "external-secrets"
    refreshInterval: 30

  aws-secrets:
    provider:
      aws:
        service: SecretsManager
        region: eu-central-1
        auth:
          jwt:
            serviceAccountRef:
              name: external-secrets-sa
```

Each key under `clusterSecretStores` becomes a `ClusterSecretStore` resource managed by FluxCD. Changes are automatically reconciled.

## Creating ExternalSecrets

Once a `ClusterSecretStore` exists, create `ExternalSecret` resources to sync individual secrets:

```yaml
apiVersion: external-secrets.io/v1beta1
kind: ExternalSecret
metadata:
  name: my-app-credentials
  namespace: my-app
spec:
  refreshInterval: 1h
  secretStoreRef:
    kind: ClusterSecretStore
    name: controlplane-vault
  target:
    name: my-app-credentials
  data:
    - secretKey: username
      remoteRef:
        key: apps/my-app
        property: username
    - secretKey: password
      remoteRef:
        key: apps/my-app
        property: password
```

## Common Use Cases

| Use Case | Guide |
|----------|-------|
| Worker cluster kubeconfigs | [Add Worker Cluster](add-worker-cluster.md) (Option B) |
| OAuth2 Proxy credentials | [Add SSO](add-sso.md) |
| Git repository credentials | [Add Repository](add-repository.md) |
