# bitown

> Embeddable isometric city that grows with visitor clicks — a modern revival of MyMiniCity

bitown lets you place a tiny city in your blog's sidebar. Every visitor who clicks once helps the city grow: residents move in, factories are built, roads are paved.

Inspired by [MyMiniCity / Miniville](http://miniville.fr/) by Motion Twin (2007–2022), rebuilt for the modern web without Flash.

---

## Status

🚧 **Phase 1 — prototype** (in development)

| | |
|---|---|
| License (code) | MIT |
| License (assets/sprites-v1) | CC BY-NC-SA 4.0 |
| Stack | Go · PostgreSQL 16 · Valkey · Next.js · PixiJS |
| Infra | AWS Lightsail $5/mo + CloudFront + S3 |

---

## Quick Start (local)

```bash
git clone https://github.com/interestic/bitown.git
cd bitown

cp .env.example .env
# Edit .env — set POSTGRES_PASSWORD and DAILY_SALT_SEED at minimum
make up
```

API: http://localhost:8080  
Health: http://localhost:8080/api/health

See [Local Dev wiki](https://github.com/interestic/bitown/wiki/Local-Dev) for full setup instructions.

### Debug mode (optional)

Set `DEBUG_MODE=true` (in `.env` or the shell) to enable a debug endpoint for
inspecting a specific city. Debug mode is ignored when `ENV=production`.

```bash
DEBUG_MODE=true make up
curl -s http://localhost:8080/api/debug/cities/<slug> | jq .
```

This endpoint returns:
- current city snapshot
- unlocked sectors
- today's support counts by sector (UTC)
- latest 20 support logs (visitor hash is shortened)

### City map image

You can fetch a generated city map PNG per slug:

```bash
curl -o map.png http://localhost:8080/api/cities/<slug>/map.png
```

The renderer uses the sprites-v1 atlas when `assets/sprites-v1/` is available
(`BITOWN_ASSETS_DIR` in Docker). If the atlas cannot be loaded and
`BITOWN_ATLAS_REQUIRED` is unset (and `ENV` is not `production`), it falls
back to a deterministic rectangle map.

- layout hashed from the city slug
- 20×20 logical grid drawn in 2:1 isometric projection
- building density linked to `pop` (center-out fill); `ind` / `com` / `env` shift zones and parks
- sprites are tagged in `assets/sprites-v1/buildings.json` (`counts.by_tag`); the map building pool excludes UI, roads, and empty clips
- responses send a strong ETag and `Cache-Control: public, max-age=300`

---

## Architecture

See [wiki/Architecture](https://github.com/interestic/bitown/wiki/Architecture).

### Art pipeline (Phase 1)

For the Flash-extract + recolor pipeline plan, see:

- `docs/art-pipeline/01_inventory.md` ... `08_ops_runbook.md`

---

## Contributing

See [CONTRIBUTING.md](./CONTRIBUTING.md).  
Security issues: see [SECURITY.md](./SECURITY.md) — please do not use public issues.

---

## License

### Code — MIT

Copyright (c) 2026 Hiroyuki Yokoshima / Interestic. See [LICENSE](./LICENSE).

### Assets (`assets/sprites-v1/`) — CC BY-NC-SA 4.0

The sprites in `assets/sprites-v1/` are **derived from**
[motion-twin/WebGamesArchives](https://github.com/motion-twin/WebGamesArchives)
and are licensed under
[CC BY-NC-SA 4.0](https://creativecommons.org/licenses/by-nc-sa/4.0/)
(Attribution · **NonCommercial** · ShareAlike).

> ⚠️ **This means the Phase 1 build of bitown as a whole is non-commercial.**  
> You may run, fork, and modify it for non-commercial purposes, but you
> **cannot** use it in a commercial product while `assets/sprites-v1/` is
> included. See [assets/sprites-v1/LICENSE](./assets/sprites-v1/LICENSE).

Phase 3 will replace all Phase 1 assets with original artwork under a
permissive license, enabling commercial use.

bitown is inspired by MyMiniCity / Miniville by Motion Twin.  
bitown is not affiliated with or endorsed by Motion Twin.
