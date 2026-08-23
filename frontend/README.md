# bitown frontend (Next.js)

City page + rankings home for Phase 1.

## Run

```bash
# API
make up

# Frontend
cd frontend
cp .env.example .env.local   # optional
npm ci
npm run dev
# → http://localhost:3000
# → http://localhost:3000/cities/testcity
```

## Env

| Variable | Default | Meaning |
|---|---|---|
| `NEXT_PUBLIC_BITOWN_API_URL` | `http://localhost:8080` | API origin |

## Notes

- Map is served as `<img src="…/map.png">` (no PixiJS yet)
- Support buttons call `POST /api/cities/{slug}/support`
- Indicators come from Core.hx metrics on city JSON
- Browser calls go cross-origin to `NEXT_PUBLIC_BITOWN_API_URL`; for production set API `CORS_ALLOW_ORIGIN` to the frontend origin

