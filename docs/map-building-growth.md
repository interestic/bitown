# Map building growth — Game.hx / Cs.hx → bitown 写像表

用語は [Glossary](https://github.com/interestic/bitown/wiki/Glossary) / [用語集](https://github.com/interestic/bitown/wiki/Glossary-ja)（population, square density, roadless town, hut foot, …）。

Epic: #38  
Phase 0: #39

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
**`displaySide×SQUARE_SIDE`（上限 25×10 = 250×250 ミニセル）**。密度は仮想 `Cs.SIDE=30` の `genMapPop` を中央 crop。低 pop は道路なし＋中央の live island（`flashDisplaySide(pop)`、pop=1 → 6×6 `mcDalle`）のみ描画（Townzzy 初期タウン / Game.hx）。PNG は島を `max(769, islandWidth)` の正方形に letterbox。**`pop ≥ 3`** で畑を解禁するが、全 plate カーペットにはしない。Game.hx `genSquare` に合わせ、密度 0 スクエアは四隣密度合計 `sidePop` が `2 ≤ sidePop < 50` かつ道路なしのときだけ全面畑（`DefineSprite_401`、`auth_big_champs_gfx` / `size=0 type=15`）。遠い空きは dalle（または森 type 14）。密度あり（`< POP_HUGE`）スクエアの空きミニは 4 セル畑（`DefineSprite_521` / `503`、`auth_champs_gfx` / `size=1 type=2`）。建物ミニ／道路には置かない。**畑と木は排他**（Game.hx: 全面畑 type 15 と森 type 14 は同じ空きスクエアで両立しない。ミニ畑 type 2 上にも木は置かない）。bitown は畑カバーを `lotFarm` で先に確保し、`env` の木・公園は残った空き芝だけに生やす。peon のスプライトは dalle 土縁 ~20px 分上げて芝上面に載せる。pop≥80 または tra>0 で全域＋幹線道路。建物は lot-column で切らず、足元の道路はみ出しだけ除去。

### 3.1 タグ写像（`getBatType` 近似）

厳密な 5 type → タグ 1:1 ではなく、**ゾーン契約を正**とし、`getBatType` は **同一ゾーン内の tier / プール絞り** に使う。

| 原作 type | bitown タグ | 備考 |
|---|---|---|
| 0（pop） | `residential` | 既定・低〜中 tier |
| 1（ind×距離） | `industrial` | 外周 2 セル + `ind>0`（セットバックで建てられる内側外周を含む） |
| 2（env） | `tree`（公園ロット＋空き/道路沿い散布） / 低 tier residential | `env` は専用公園数と**畑でない**空きロットへの木散布。建物プールには混ぜない。公園増は同 pop の建物枠を減らす |
| 3（sec） | （M1 では重みのみ） | 専用タグなし。`sec` は landmark 出現の弱いブースト |
| 4（com） | `commercial` | 既存: 中心 + `com>0` ゾーン契約を維持 |
| 特殊 / 高密度 | `landmark` | 高 pop かつ条件付きでプールに混入 |

**ゾーン契約（既存・壊さない）**

- 中心付近 + `com > 0` → `commercial`
- 外周 2 セル + `ind > 0` → `industrial`（幅 1 だと curb のみで倉庫が立たない）
- それ以外 → `residential`
- 公園ロット → `tree`（建物ではない。`env` 専用公園 + 空きロット散布。緑優先で建物数は減りうる。**畑カバー `lotFarm` 上には置かない**）

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

### 3.3 密度場（Game.hx `genMapPop`）と pop スケール

**正本:** [Game.hx](https://github.com/motion-twin/WebGamesArchives/blob/main/Miniville/client/src/Game.hx) の `genMapPop` → `genSquare` → `genMiniSquare` / `updateLib`。

1. 仮想 `Cs.SIDE=30` 格子に pop を極座標で積み上げ（`getRayMax(n) = 1 + n^0.6 * 0.15`）。
2. `BlurFilter(2,2)` 相当の 3-tap box blur でなます。
3. 中央 `displaySide×displaySide`（上限 25×25 スクエア = 250×250 ミニセル）を crop（API フィールド上限。SIDE=30 全体は描かない）。live island は `flashDisplaySide(pop)` で Game.hx `displayMargin` に揃え、PNG は島を正方形 letterbox（`max(769, islandWidth)`）。Game.hx は `Std.int` で pop=1 → displaySide=6。
4. `flashDisplaySide(pop)` でアクティブなスクエア数を決め、その範囲だけ `genMiniSquare` 配置（cap 25）。
5. **roadless town** も Game.hx と同じ quadrant 分割を使う: 台座 10×10 を 4 quadrant に分け、密度あり quadrant は 2×2 hut foot に最大 4 軒。中央 1 セルへスナップしない（#116）。**小ダイア内側閉じ（#118）:** roadless では Game.hx のミニ内 `(0,0)/(0,2)/(2,0)/(2,2)` が象限／プレートつなぎ目に足が乗るため、1 セル SE に寄せて `(1,1)/(1,3)/(3,1)/(3,3)` に置く（見た目の象限内閉じを座標互換より優先）。arterial は Game.hx 足のまま（道路ファサード契約）。家は `grassTopCell` の grass top のみ（soil rim は置かない）。中層クローン重なりは roadless で chebyshev≤2 の同一フォルダ避け（#114）。
6. **畑床:** `farmsEnabled = pop ≥ 3`（Townzzy 描画ゲート。Game.hx 自体には無い）。発生はスクエア密度（`bmpPop`）:
   - 密度 `> 0` かつ `< POP_HUGE` → 4 ミニに割る。ミニ密度 `== 0` なら 4 セル畑（足元はミニ SE = origin+`(3,3)`、描画は連続ブロック菱形にクリップ）。roadless では家足が小ダイア内側なので、空きミニの畑は隣象限に家があってもスタンプする（パンチアウトしない）。arterial は Game.hx 縁足のままなので、同一スクエアに建物があるときはミニ畑スタンプを出さない。ミニ密度 `> 0` は建物（`densityHut` / Cs.POP_PEON は小屋閾値であり畑解禁ではない）。
   - 密度 `== 0` → 四隣 `sidePop` が `2 ≤ sidePop < 50` かつ道路なしなら全面畑。それ以外は plate（遠い空きは畑にしない。Game.hx では森 type 14）。
   - **排他:** occupancy で畑カバーを `lotFarm` として先にマークし、専用公園 / `env` 散布の木は `lotEmpty` の芝だけ（畑の上に木を乗せない）。配置後に畑 chebyshev≤1 の公園は落とす。デコ入り／方眼入り畑クリップ（521/5, 401/1, 401/2, 401/6）は stamp プールから除外（#113 / #116）。畑・道路・木・建物は roadless / arterial とも `plateGrassLift`（20px、catalog plateLip）で grass top に載せる。はみ出しは square / grass クリップで切る（リフトを短くしない）。芝マスクは全島セルを `-plateGrassLift` で grass top に合わせ、ピクセル単位で判定（列の `py<=maxY` だと北キャンバスへ漏れる）。畑スタンプは島の最外縁セルを避ける。arterial 道路（702/705）の足元はスクエア SE。
   整数 3-tap blur が堆積を 0 に潰す場合、`pop ≥ 3` だけ生の堆積を残す（pop=1 の空タウンは維持）。
   plate の後・オブジェクトの前。決定論は slug + 座標。
7. `updateLib`: `densityMax`（と city pop フォールバック）で mcHouse1/2/3（`library_ref`）を解禁。

都市全体 population の building band（unlock / landmark）:

| building band | bitown `pop` | 許可 tier（天井） | 備考 |
|---|---:|---|---|
| hut band | `< 40` | 0…1 | landmark 禁止 |
| house band | `40 … 119` | 0…2 | mcHouse2 解禁 |
| city band | `120 … 349` | 0…3 | landmark 条件付き |
| city band (huge) | `≥ 350` | 0…3 | mcHouse3 解禁 |

`getBatType` は住宅ロットの多様化に使い、ind/com の空間帯は既存 `zoneTag` 契約を維持（島相対の距離。フィールド全体 250×250 ではなく live island）。
### 3.4 ロットごとのプール選択（#41 実装契約）

1. `zoneTag` でベースタグを決定（既存契約）。
2. 都市 `pop` とロット距離から **許可 max tier** を決める（中心ほど高い tier を許しやすい。住宅は民家帯が広い）。
3. 許可 max tier 内で **tier を先に重み抽選**（同一 tier は重み 1 枠。tall カタログ増で中層が溺れない。住宅は低層寄り）。
4. 選ばれた tier のフォルダから均等抽選し、フォルダ内フレームを決定論 seed で選ぶ（多フレームクリップの洪水対策）。
5. 高密度段階かつ重み条件を満たすときだけ `landmark` を候補に混ぜる。
6. 空プールは既存どおり `residential` → 矩形フォールバック。
7. マップ全体をラスター順で確定するとき、**既に置いた近傍の同一フォルダを避ける**（tier 問わず。arterial は chebyshev≤1、roadless は≤2 — 象限足が密なため #114）。roadless の家の足は genMiniSquare の象限＋2×2 を **1 セル内側**に置き、プレート中央へ寄せない（#116 / #118）。arterial は Game.hx 2×2 のまま。

PNG / ETag の決定論は維持（同一 slug・同一パラメータで同一出力）。

### 3.5 `sec` / `tra`（オープンクエスチョンへの暫定方針）

| 入力 | M1 |
|---|---|
| `sec` | `getBatType` 重み index 3 として **landmark ブーストにのみ使用**（専用ゾーンなし） |
| `tra` | **幹線道路の解禁**（`pop≥80` または `tra>0` で `arterialsEnabled`）。建物 tier / ゾーン選択には未使用 |
| `PROBA_SPECIAL` | M1 では省略可（見た目 variant は既存 `_v00` 固定で十分） |

厳密写しが必要になったら Phase 後続で見直す。

### 3.6 オブジェクト解禁表（#79）

tier 上限に加え、カタログ各 base に **sector 最小値**（`unlock`）を付与する。  
抽選前に `pop / ind / com / env / sec / tra` を満たす候補だけ残す（未設定 = 制限なし）。

| 条件 | ヒューリスティック（`generate_buildings_manifest.py`） |
|---|---|
| tier 1 建物 | `min_pop` 40 |
| tier 2 建物 | `min_pop` 120 |
| tier 3 建物 | `min_pop` 350 |
| industrial tier 1 | `min_ind` 1 |
| industrial tier 2+ | `min_ind` 50 |
| commercial tier 1 | `min_com` 1 |
| commercial tier 2+ | `min_com` 50 |
| landmark | `min_pop` 120, `min_sec` 300 |
| tree（4 種） | `min_env` 0 / 80 / 200 / 400（種類ごと） |

手動調整は `sprite_tag_overrides.json` の `unlocks` で上書き。  
実装: `api/internal/render/unlock.go` → `PickBuildingKeyForLot` / `PickKeyForTagUnlocked`。

### 3.7 SWF キャラクタグラフ（needed / dependent / spawn role）

FFDec `swf2xml` から `assets/sprites-v1/swf_character_graph.json` を生成し、`buildings.json` 各 entry に載せる。

| フィールド | JPEXS / SWF 相当 | 用途 |
|---|---|---|
| `needed_characters` | Needed Characters | 合成ツリー（子クリップ。出現率そのものではない） |
| `dependent_characters` | Dependent Characters | このクリップを使う親 |
| `placeable_hint` | （派生） | `mc*` export は true。それ以外は dependent が空のとき true |
| `role` | mcHouse ShowFrame 列 | `library_primary` / `building_module` / `deco_subpart` / `spawn_library` … |
| `library_ref` | mcHouse1/2/3 frame | `{library_id, library_name, frame}` — 原作 `updateLib` の size 枠 |
| `pool_eligible` | （派生） | **抽選プール**に載せる主物（mcHouse library frame のみ） |
| `building_class` | （派生） | module を `exclude` に落とすとき元の building tag を保持 |

**抽選プール:** `building_bases` は `pool_eligible=true` の **15 件**（mcHouse1/2/3 の library frame 主物）。  
249 / 257 など `building_module` は `exclude` へ降格し、主物プールから外す（#82）。畑・床クリップ（`DefineSprite_401` / `521`）は `ground` 上書きでプールから外し、畑スタンプ専用に残す（#89）。

再生成:

```bash
python3 scripts/extract_swf_character_graph.py
python3 scripts/generate_buildings_manifest.py
make assets-check
```

---

## 4. DEBUG 比較手順

`DEBUG_MODE=true` のとき（#37）:

```bash
# 低: 民家・低 tier 寄り。roadless town は plate island（grass top）に建物
open "http://localhost:8080/api/cities/testcity1/map.png?pop=8&ind=0&com=0&env=0"
open "http://localhost:8080/api/cities/testcity1/map.png?pop=20&ind=0&com=0&env=0"

# 中: 住宅ゾーンに民家、中心は中高層。同一高層の 2×2 密集は避ける
open "http://localhost:8080/api/cities/testcity1/map.png?pop=300&ind=10&com=5&env=100"

# 高: 中高層・landmark 混在
open "http://localhost:8080/api/cities/testcity1/map.png?pop=500&ind=10&com=5&env=400"
```

受け入れ: 密度差だけでなく **建物種の構成**が目に見えて違うこと。

---

## 5. 後続 issue

| Issue | 内容 |
|---|---|
| #40 | `buildings.json` に tier |
| #41 | 本写像表どおりのプール選択 |
| #42 | 回帰テスト + README |
| #79 | オブジェクト解禁表 + 抽選フィルタ |
| #81 | 合成スプライト composition（depth/matrix） |
| #82 | 装飾サブオブジェクト / 249 を主物から外す |

---

## 6. 非ゴール

- Flash ピクセル一致 / マルチタイル正式占有 / visites 経済の完全移植
