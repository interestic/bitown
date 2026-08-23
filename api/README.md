# api/

Go サーバーモジュール（`module github.com/interestic/bitown`）。  
Go server module for the bitown backend.

---

## パッケージ構成 / Package layout

```text
api/
├── cmd/            # エントリーポイント / Entry points
├── internal/
│   ├── citycore/   # ドメイン層 — stdlib only, no I/O drivers
│   ├── city/       # HTTP ハンドラー + DTO / HTTP handlers + DTOs
│   ├── store/      # PostgreSQL / Valkey アダプター / adapters
│   ├── render/     # PNG マップ描画 / PNG map rendering
│   └── middleware/ # CORS・ClientIP など / CORS, ClientIP, etc.
└── migrations/     # DB マイグレーション / DB migrations
```

---

## 依存方向ルール / Dependency direction

Clean Architecture の依存関係の向きに従います。  
Follows Clean Architecture dependency inversion:

| CA レイヤー | 現在の置き場 |
|---|---|
| Entities / domain | `internal/citycore` |
| Use cases | handlers の薄いヘルパー + `citycore` 内ロジック |
| Interface adapters | `internal/city`（handlers, DTO） |
| Frameworks / drivers | PostgreSQL · Valkey · `net/http` · PNG render |

### 許可 / Allowed

- `internal/city` → `internal/citycore`
- `internal/render` → `internal/citycore`

### 禁止 / Forbidden

- `internal/citycore` → 外側パッケージ（`city`, `store`, `render`, `middleware`, `cmd`）
- `internal/citycore` → `net/http`, `github.com/jackc/pgx`, `go-redis`

`citycore` は **stdlib のみ**。HTTP・DBドライバ・外部パッケージへの依存は持ちません。  
`citycore` is **stdlib-only** — no HTTP, no DB drivers, no external imports.

### 非目標 / Non-goals

- エンティティ/ユースケースの大規模なディレクトリ再編
- DI フレームワーク（wire, fx など）の導入

`citycore` は HTTP / DB ドライバに依存しないドメイン層として保ちます。
