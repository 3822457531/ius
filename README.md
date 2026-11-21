# ius

API (`api/`) + Web (`web/`) monorepo.

## Setup

1. Copy `api/config.sample.toml` → `api/config.toml` and fill in local secrets (never commit `config.toml`).
2. Copy `web/.env.production.example` → `web/.env.production` for production builds.
3. Development env: `web/.env.development` (localhost).

```bash
# API
cd api && go run .

# Web
cd web && npm install && npm run serve
```
