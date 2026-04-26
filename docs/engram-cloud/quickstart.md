[← Back to Engram Cloud](./README.md)

# Engram Cloud Quickstart

**Fastest working setup:** run the compose smoke profile, enroll one project, and sync explicitly.

This page gives one recommended path first. Advanced/authenticated mode follows after.

---

## Recommended Path: Local Smoke (Docker Compose)

### 1) Start cloud runtime + Postgres

```bash
docker compose -f docker-compose.cloud.yml up -d
```

`docker-compose.cloud.yml` defaults on this branch:
- `ENGRAM_CLOUD_INSECURE_NO_AUTH=1`
- `ENGRAM_CLOUD_ALLOWED_PROJECTS=smoke-project`
- cloud endpoint published at `http://127.0.0.1:18080`

### 2) Configure CLI cloud endpoint

```bash
engram cloud config --server http://127.0.0.1:18080
```

### 3) Enroll explicit project

```bash
engram cloud enroll smoke-project
```

### 4) Sync explicitly in cloud mode

```bash
engram sync --cloud --project smoke-project
engram sync --cloud --status --project smoke-project
```

### 5) Verify browser dashboard

Open:
- `http://127.0.0.1:18080/dashboard`

In compose smoke mode, `/dashboard/login` redirects to `/dashboard/` (no bearer login needed).

---

## Existing Project Upgrade Path (recommended)

Use this sequence before first bootstrap for established local projects:

```bash
engram cloud upgrade doctor --project smoke-project
engram cloud upgrade repair --project smoke-project --dry-run
engram cloud upgrade repair --project smoke-project --apply
engram cloud upgrade bootstrap --project smoke-project --resume
engram cloud upgrade status --project smoke-project
```

`rollback` is only available before bootstrap reaches `bootstrap_verified`.

---

## Deploy with Official GHCR Image (Dokploy/Coolify/Portainer/VPS)

Do not build from source for production deploys. Use the published image:

- `ghcr.io/gentleman-programming/engram:latest`

Reference compose file:
- [docker-compose.ghcr.yml](./docker-compose.ghcr.yml)

Required runtime env vars:
- `ENGRAM_DATABASE_URL`
- `ENGRAM_CLOUD_TOKEN`
- `ENGRAM_CLOUD_ADMIN`
- `ENGRAM_JWT_SECRET`
- `ENGRAM_CLOUD_ALLOWED_PROJECTS`
- `ENGRAM_CLOUD_HOST=0.0.0.0`
- `ENGRAM_PORT=18080`

Dokploy guidance:
1. Create a managed Postgres service.
2. Create an app from image `ghcr.io/gentleman-programming/engram:latest`.
3. Configure the env vars above (with strong secrets).
4. Expose container port `18080`.
5. Avoid build-from-source mode unless you are actively developing Engram itself.

> `ENGRAM_CLOUD_INSECURE_NO_AUTH=1` is for local/dev smoke only. Never use it in production.

---

## Common Failure Reasons

| Reason code | Meaning |
|---|---|
| `blocked_unenrolled` | Project is not enrolled for cloud replication |
| `auth_required` | Authenticated runtime requires valid token/session |
| `cloud_config_error` | Cloud endpoint config is missing/invalid |
| `policy_forbidden` | Project blocked by cloud policy |
| `paused` | Project sync paused in cloud control plane |
| `transport_failed` | Cloud transport/network operation failed |

---

<details>
<summary><strong>Advanced: Authenticated Source-Run Mode</strong></summary>

Use this when you are running `engram cloud serve` directly (no insecure compose smoke mode):

```bash
ENGRAM_DATABASE_URL="postgres://engram:engram_dev@127.0.0.1:5433/engram_cloud?sslmode=disable" \
ENGRAM_JWT_SECRET="replace-with-32+-byte-random-secret" \
ENGRAM_CLOUD_TOKEN="your-token" \
ENGRAM_CLOUD_ALLOWED_PROJECTS="my-project" \
engram cloud serve
```

Then configure client endpoint + token:

```bash
engram cloud config --server http://127.0.0.1:8080
export ENGRAM_CLOUD_TOKEN="your-token"
engram cloud enroll my-project
engram sync --cloud --project my-project
```

Rules that matter:
- `ENGRAM_CLOUD_INSECURE_NO_AUTH=1` cannot be combined with `ENGRAM_CLOUD_TOKEN`
- `ENGRAM_CLOUD_ALLOWED_PROJECTS` is required server-side in both modes
- authenticated mode requires explicit non-default `ENGRAM_JWT_SECRET`
- `ENGRAM_CLOUD_INSECURE_NO_AUTH=1` remains local/dev only (never production)

</details>

---

## Next Steps

- Deep runtime/env reference: [DOCS.md — Cloud CLI](../../DOCS.md#cloud-cli-opt-in)
- Background sync mode: [DOCS.md — Cloud Autosync](../../DOCS.md#cloud-autosync)
- Branding assets and usage: [Branding](./branding.md)
