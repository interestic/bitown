# Embed widget

Vanilla iframe widget for blog sidebars (Phase 1 embed).

## Snippet

```html
<div
  data-bitown-city="testcity"
  data-bitown-width="200"
  data-bitown-height="240"
  data-bitown-api="http://localhost:8080"
></div>
<script src="http://localhost:8080/embed/bitown.js" async></script>
```

Or iframe directly:

```html
<iframe
  src="http://localhost:8080/embed/widget.html?slug=testcity&api=http://localhost:8080"
  width="200"
  height="240"
  style="border:0;overflow:hidden"
  title="bitown"
></iframe>
```

## Local

1. `make up`（API on :8080）
2. API serves `/embed/*` from this directory
3. Open a page with the snippet, or hit `http://localhost:8080/embed/widget.html?slug=testcity`

## Behavior

- Shows `map.png` for the slug
- Click map → sector chooser → `POST /api/cities/{slug}/support`
- Same visitor × city × UTC day → “Already supported today”

## CORS / production

Prefer serving the widget and API from the **same origin** (e.g. `https://bitown.dev/embed/…` + `https://bitown.dev/api/…`).

If the city page or blog host differs from the API origin, set `CORS_ALLOW_ORIGIN` on the API to that host. `data-bitown-api` / `?api=` only accept the script origin or `localhost` / `127.0.0.1` (arbitrary remote API URLs are ignored).

