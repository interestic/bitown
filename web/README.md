# web — placement object catalog (Storybook)

Local Storybook gallery for sprites-v1 placement objects
(`residential` / `industrial` / `commercial` / `road` / `tree` / `water` …).

```bash
cd web
npm install
npm run storybook
```

Open http://localhost:6006 — stories:

- **Lot Patterns / Overview** — Ventura で見える空き地パターン（緑ベタ、黄色畝、かぼちゃ 5、4 分割、ひし形内道路など）
- **Placement Objects / Overview** — 配置オブジェクト早見

Assets are served from `../assets/sprites-v1` via Storybook `staticDirs`.
