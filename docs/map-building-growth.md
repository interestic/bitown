# Map building growth — Game.hx / Cs.hx → bitown 写像表

原作正本:

- [Game.hx](https://github.com/motion-twin/WebGamesArchives/blob/main/Miniville/client/src/Game.hx)
- [Cs.hx](https://github.com/motion-twin/WebGamesArchives/blob/main/Miniville/client/src/Cs.hx)

`client.swf` の AS dump は難読化照合用のみ。実装の正本にはしない。

---

## 1. 原作定数（Cs.hx）

| 定数 | 値 | 役割 |
|---|---:|---|
| `POP_PEON` | 3 | 極低密度: 簡易住宅枠（type id 12 寄り） |
| `POP_NORMAL` | 2 | ミニセル内の「普通以上」判定（`rep[i] > POP_NORMAL` で `getBatType`） |
| `POP_BIG` | 20 | 中規模: size フレーム解禁・ミニセル分岐 |
| `POP_HUGE` | 200 | 大規模: スクエア全体を大型枠、`updateLib` で最大 size 解禁 |
| `PROBA_SPECIAL` | 500 | `random(500)==0` で variant+1（約 0.2%） |

密度半径:

```text
getRayMax(n) = 1 + n^0.6 * 0.15
```

---

## 2. 原作の「ランク」相当（明示フィールドなし）

1. **size フレーム解禁**（`updateLib`）  
   `densityMax < POP_BIG` / `< POP_HUGE` のとき大きな `mcHouse` 枠をスキップ。
2. **セル密度閾値**（`genSquare` / `genMiniSquare`）  
   `POP_PEON` / `POP_NORMAL` / `POP_BIG` / `POP_HUGE` で建物 size・type を分岐。
3. **`getBatType` 重み**（下表）で type 0…4 を抽選。
4. **`PROBA_SPECIAL`** で稀に見た目 variant。

低密度空きセルは木（type 14）や畑寄り（type 15）。

### `getBatType` 重み（Game.hx）

セル中心からの正規化距離 `coef = dist / (getRayMax(pop) * SQUARE_SIDE)` を使い:

| index | 重み | 意味（原作） |
|---:|---|---|
| 0 | `pop` | 住宅寄り |
| 1 | `ind * (coef * 5)` | 産業（外周ほど強い） |
| 2 | `env * 2` | 環境 |
| 3 | `sec * 0.2` | 治安 |
| 4 | `com * 0.75` | 商業 |

weighted pick（累積和）で type を返す。

---

## 3. bitown への写像（M1 採用方針）

Flash の SIDE=30 全体（300×300 ミニセル）は API 向け PNG が過大なのでコピーしない。  
**`displaySide×SQUARE_SIDE`（既定 4×10 = 40×40）**。低 pop は中央の 4×4 `mcDalle` 台地のみ（Townzzy Caerphilly）。pop≥80 または tra>0 で全域＋幹線道路。建物は lot-column で切らず、足元の道路はみ出しだけ除去。

### 3.1 タグ写像（`getBatType` 近似）

厳密な 5 type → タグ 1:1 ではなく、**ゾーン契約を正**とし、`getBatType` は **同一ゾーン内の tier / プール絞り** に使う。

| 原作 type | bitown タグ | 備考 |
|---|---|---|
| 0（pop） | `residential` | 既定・低〜中 tier |
| 1（ind×距離） | `industrial` | 外周 2 セル + `ind>0`（セットバックで建てられる内側外周を含む） |
| 2（env） | `tree`（公園ロット＋空き/道路沿い散布） / 低 tier residential | `env` は専用公園数と空きロットへの木散布。建物プールには混ぜない。公園増は同 pop の建物枠を減らす |
| 3（sec） | （M1 では重みのみ） | 専用タグなし。`sec` は landmark 出現の弱いブースト |
| 4（com） | `commercial` | 既存: 中心 + `com>0` ゾーン契約を維持 |
| 特殊 / 高密度 | `landmark` | 高 pop かつ条件付きでプールに混入 |

**ゾーン契約（既存・壊さない）**

- 中心付近 + `com > 0` → `commercial`
- 外周 2 セル + `ind > 0` → `industrial`（幅 1 だと curb のみで倉庫が立たない）
- それ以外 → `residential`
- 公園ロット → `tree`（建物ではない。`env` 専用公園 + 空きロット散布。緑優先で建物数は減りうる）

### 3.2 建物 tier（カタログメタデータ）

| tier | 意味 | 典型 |
|---:|---|---|
| 0 | 小屋・畑・低層民家 | 低い bbox / 小さな residential |
| 1 | 普通住宅・低層店舗 | 中程度 residential / 小型 commercial |
| 2 | 中高層・倉庫・大きめ街区 | 高い bbox / industrial・大型 commercial |
| 3 | landmark / 目立つ高層 | `landmark` タグ、または `bbox_h >= 80` の突出した高さ |

未定義 tier は **フォールバック 1**（普通）。

割当は `scripts/generate_buildings_manifest.py`（tag + `max_bbox_height` ヒューリスティック）と、任意の `sprite_tag_overrides.json` → `tiers`。  
`counts.by_tier` と Storybook カタログで確認できる。

**低 pop（tier 0）ギャップメモ:** 現行プールで tier 0 は少数（住宅・小型 commercial 中心）。畑スケールの専用アートが足りない場合は、`exclude` からの昇格を手作業で検討する（UI 断片は昇格しない）。

### 3.3 pop スケールのリマップ

bitown の充填は `min(pop/500, 1)` ベース（`lotOccupancy`）。  
原作の `POP_*` はセル局所密度なので、**都市全体 pop** に次の閾値を当てる（近似）:

| 段階 | bitown `pop` | 許可 tier（目安） | 備考 |
|---|---:|---|---|
| peon | `< 40` | 0…1 | 民家・低層優先。landmark 禁止 |
| normal | `40 … 119` | 0…2 | 中層解禁。landmark は極稀 |
| big | `120 … 349` | 0…3 | landmark 条件付き解禁 |
| huge | `≥ 350` | 0…3 | landmark 混入率を上げる |

外周ロットは中心より max tier を 1 下げ、遠外周（dist² ≥ 2×outer）はさらに 1 にキャップ（原作 big_city の縁の民家帯）。**住宅ゾーン**は big 帯（pop 350 未満）で outer を半分にして民家帯を広くする。huge と工業外周はこのキャップを緩くして中心の高層・倉庫を残す。

住宅ゾーンの tier 抽選は big / huge でも低層（0–1）を厚くする。カタログに高層が 1 本しかないと、汎用重みではその塔が住宅ロットの過半を占めてしまうため。

閾値は原作 `3 / 20 / 200` を 500 飽和スケールへ伸ばした近似（×約 1.7〜2）。  
厳密写しではなく、**低/高で建物種が目に見えて違う**ことを優先する。
### 3.4 ロットごとのプール選択（#41 実装契約）

1. `zoneTag` でベースタグを決定（既存契約）。
2. 都市 `pop` とロット距離から **許可 max tier** を決める（中心ほど高い tier を許しやすい。住宅は民家帯が広い）。
3. 許可 max tier 内で **tier を先に重み抽選**（同一 tier は重み 1 枠。tall カタログ増で中層が溺れない。住宅は低層寄り）。
4. 選ばれた tier のフォルダから均等抽選し、フォルダ内フレームを決定論 seed で選ぶ（多フレームクリップの洪水対策）。
5. 高密度段階かつ重み条件を満たすときだけ `landmark` を候補に混ぜる。
6. 空プールは既存どおり `residential` → 矩形フォールバック。
7. マップ全体をラスター順で確定するとき、**tier≥2 は既に置いた上下左右の同一フォルダを避ける**（民家の隣接は許容）。

PNG / ETag の決定論は維持（同一 slug・同一パラメータで同一出力）。

### 3.5 `sec` / `tra`（オープンクエスチョンへの暫定方針）

| 入力 | M1 |
|---|---|
| `sec` | `getBatType` 重み index 3 として **landmark ブーストにのみ使用**（専用ゾーンなし） |
| `tra` | **未使用**（道路係数。道路レイアウトは slug ハッシュのまま） |
| `PROBA_SPECIAL` | M1 では省略可（見た目 variant は既存 `_v00` 固定で十分） |

厳密写しが必要になったら Phase 後続で見直す。

---

## 4. DEBUG 比較手順

`DEBUG_MODE=true` のとき:

```bash
# 低: 民家・低 tier 寄り。peon は dalle 1 枚（緑の芝上辺）に建物 1 つ
open "http://localhost:8080/api/cities/testcity/map.png?pop=8&ind=0&com=0&env=0"
open "http://localhost:8080/api/cities/testcity/map.png?pop=20&ind=0&com=0&env=0"

# 中: 住宅ゾーンに民家、中心は中高層。同一高層の 2×2 密集は避ける
open "http://localhost:8080/api/cities/testcity/map.png?pop=300&ind=10&com=5&env=100"

# 高: 中高層・landmark 混在
open "http://localhost:8080/api/cities/testcity/map.png?pop=500&ind=10&com=5&env=400"
```

受け入れ: 密度差だけでなく **建物種の構成**が目に見えて違うこと。

---

## 5. 実装メモ（完了済み）

| 項目 | 内容 |
|---|---|
| catalog tier | `buildings.json` に tier |
| pool selection | 本写像表どおりのプール選択 |
| regression | 回帰テスト + README |

---

## 6. 非ゴール

- Flash ピクセル一致 / マルチタイル正式占有 / visites 経済の完全移植
