# union-api

HTTP service that exposes a service accounts management interface. It queries
Union permissions via the `uctl` CLI and GKE namespaces via the Connect Gateway,
and sits behind Entra ID (Azure AD) authentication.

## Local development

### Prerequisites

- Go 1.26+
- [`uctl`](https://www.union.ai/docs/v2/union/api-reference/uctl-cli/) installed and on `$PATH`
- [gcloud CLI](https://cloud.google.com/sdk/docs/install) with application-default credentials:
  ```sh
  gcloud auth application-default login
  ```

### Setup

1. **Copy the config template and fill in non-secret values:**
   ```sh
   cp config.example.yaml config.local.yaml
   ```
   Edit `config.local.yaml` with the correct endpoints, org, client IDs, etc.
   for your target environment. The file is gitignored.

2. **Copy the secrets template and fill in secret values:**
   ```sh
   cp .env.example .env
   ```
   Edit `.env` with the two client secrets and session secret. The file is gitignored.

   `UNION_CLIENT_SECRET` must be an environment variable (not just in the config
   file) because the `uctl` binary resolves it by name in its own process at
   runtime.

3. **Run the service:**
   ```sh
   make run
   ```
   The Makefile loads `.env` automatically and passes `config.local.yaml` to the
   service via `CONFIG_FILE`.

## Configuration

Configuration is loaded from a YAML file specified by the `CONFIG_FILE`
environment variable. All fields can be overridden by environment variables —
env vars take priority over the config file.

See [`config.example.yaml`](config.example.yaml) for the full list of available
fields and their default values.

### Secrets

| Secret | Config key | Environment variable |
|---|---|---|
| Entra ID client secret | `entra_id.client_secret` | `ENTRA_ID_CLIENT_SECRET` |
| Union client secret | _(env var name only)_ `union.client_secret_env_var` | `UNION_CLIENT_SECRET` |
| Session cookie signing key | `session_secret` | `SESSION_SECRET` |

Generate a session secret with: `openssl rand -base64 32`

In production, secrets are injected as environment variables by the cluster and
are never present in config files.

## Make targets

| Target | Description |
|---|---|
| `make run` | Run the service locally with `config.local.yaml` and `.env` |
| `make build` | Build binary to `bin/serviceaccounts` |
| `make test` | Run all tests |
