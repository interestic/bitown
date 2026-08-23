package render

import (
	"encoding/json"
	"fmt"
	"image"
	_ "image/png"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

type frameRect struct {
	X       int `json:"x"`
	Y       int `json:"y"`
	W       int `json:"w"`
	H       int `json:"h"`
	AnchorX int `json:"anchor_x"`
	AnchorY int `json:"anchor_y"`
}

type atlasMeta struct {
	Image  string               `json:"image"`
	Count  int                  `json:"count"`
	Frames map[string]frameRect `json:"frames"`
}

type buildingsManifest struct {
	Version       int                 `json:"version"`
	BuildingBases []string            `json:"building_bases"`
	BasesByTag    map[string][]string `json:"bases_by_tag"`
	Entries       []catalogEntry      `json:"entries"`
}

type catalogEntry struct {
	Base  string `json:"base"`
	Group string `json:"group"`
	Tag   string `json:"tag"`
	Tier  *int   `json:"tier,omitempty"`
}

const (
	TagResidential = "residential"
	TagIndustrial  = "industrial"
	TagCommercial  = "commercial"
	// TagLandmark is catalogued for growth placement (high pop mix); unused as a zone tag.
	TagLandmark = "landmark"
	TagRoad     = "road"
	TagTree     = "tree"
	TagWater    = "water"
	// TagGround is stamped as 4×4 mcDalle plates (peon / Caerphilly field).
	TagGround = "ground"
	// TagPark is catalogued but intentionally unused in M1 (count may be 0).
	// Park lots are drawn with TagTree sprites instead.
	TagPark    = "park"
	TagExclude = "exclude"
)

var mapBuildingTags = map[string]struct{}{
	TagResidential: {},
	TagIndustrial:  {},
	TagCommercial:  {},
	TagLandmark:    {},
}

// Tokens that must never enter the map building pool, even if the manifest is stale.
var buildingDenySubstr = []string{
	"mcLoading", "mcAnti", "mcAnalog", "mcStat", "mcCompt", "mcObs",
	"mcTest", "mcBg", "mcDalle", "mcRoad", "brushWood", "StatPanel", "StatusBar",
}

// Atlas holds the sprites v1 texture atlas and lookup tables for rendering.
type Atlas struct {
	Image         image.Image
	Frames        map[string]frameRect
	BuildingBases []string
	BasesByTag    map[string][]string
	// TierByFolder maps sprite folder base (e.g. sprites/Foo) to growth tier 0..3.
	// Missing keys fall back to DefaultBuildingTier (1) in pool selection.
	TierByFolder  map[string]int
	VariantSuffix func(seed uint32) string
	sourceDir     string
	revision      string
}

var (
	atlasMu   sync.RWMutex
	atlasInst *Atlas
)

func loadAtlas() (*Atlas, error) {
	base, err := resolveSpritesV1Dir()
	if err != nil {
		return nil, wrapAtlasError(err)
	}

	root, err := os.OpenRoot(base)
	if err != nil {
		return nil, wrapAtlasError(fmt.Errorf("open sprites-v1 root: %w", err))
	}
	defer func() { _ = root.Close() }()

	revision, err := atlasDiskRevision(root)
	if err != nil {
		return nil, wrapAtlasError(err)
	}

	atlasMu.RLock()
	if atlasInst != nil && atlasInst.sourceDir == base && atlasInst.revision == revision {
		cached := atlasInst
		atlasMu.RUnlock()
		return cached, nil
	}
	atlasMu.RUnlock()

	atlasMu.Lock()
	defer atlasMu.Unlock()
	if atlasInst != nil && atlasInst.sourceDir == base && atlasInst.revision == revision {
		return atlasInst, nil
	}

	loaded, err := loadAtlasFromRoot(base, root, revision)
	if err != nil {
		return nil, wrapAtlasError(err)
	}
	atlasInst = loaded
	return atlasInst, nil
}

func loadAtlasFromRoot(base string, root *os.Root, revision string) (*Atlas, error) {
	metaBytes, err := readRootFile(root, "atlas/sprites_v1_atlas.json")
	if err != nil {
		return nil, fmt.Errorf("read atlas json: %w", err)
	}

	var meta atlasMeta
	if err := json.Unmarshal(metaBytes, &meta); err != nil {
		return nil, fmt.Errorf("parse atlas json: %w", err)
	}

	imgName := filepath.Base(meta.Image)
	if imgName == "." || imgName == string(filepath.Separator) || strings.Contains(imgName, "..") {
		return nil, fmt.Errorf("invalid atlas image name")
	}

	imgFile, err := root.Open(filepath.Join("atlas", imgName))
	if err != nil {
		return nil, fmt.Errorf("open atlas png: %w", err)
	}
	defer func() { _ = imgFile.Close() }()

	img, _, err := image.Decode(imgFile)
	if err != nil {
		return nil, fmt.Errorf("decode atlas png: %w", err)
	}

	for key, rect := range meta.Frames {
		if rect.W <= 0 || rect.H <= 0 {
			return nil, fmt.Errorf("atlas frame %s has invalid size", key)
		}
		bounds := img.Bounds()
		if rect.X < bounds.Min.X || rect.Y < bounds.Min.Y ||
			rect.X+rect.W > bounds.Max.X || rect.Y+rect.H > bounds.Max.Y {
			return nil, fmt.Errorf("atlas frame %s exceeds image bounds", key)
		}
	}

	bases := make([]string, 0, len(meta.Frames))
	for key := range meta.Frames {
		if strings.HasSuffix(key, "_v00.png") && strings.HasPrefix(key, "sprites/") {
			bases = append(bases, strings.TrimSuffix(key, "_v00.png"))
		}
	}
	sort.Strings(bases)
	if len(bases) == 0 {
		return nil, fmt.Errorf("atlas has no v00 sprite frames")
	}

	catalog, err := loadBuildingsCatalog(root, bases)
	if err != nil {
		return nil, err
	}
	catalog = filterOversizedSingleLotBuildings(catalog, meta.Frames)

	return &Atlas{
		Image:         img,
		Frames:        meta.Frames,
		BuildingBases: catalog.buildingBases,
		BasesByTag:    catalog.basesByTag,
		TierByFolder:  catalog.tierByFolder,
		sourceDir:     base,
		revision:      revision,
		VariantSuffix: func(seed uint32) string {
			return fmt.Sprintf("_v%02d.png", seed%4)
		},
	}, nil
}

func resolveSpritesV1Dir() (string, error) {
	for _, dir := range spritesV1Candidates() {
		if atlasRootExists(dir) {
			return dir, nil
		}
	}
	if strings.TrimSpace(os.Getenv("BITOWN_ASSETS_DIR")) != "" {
		return "", fmt.Errorf("sprites-v1 atlas not found under BITOWN_ASSETS_DIR")
	}
	return "", fmt.Errorf("sprites-v1 atlas not found (set BITOWN_ASSETS_DIR)")
}

func spritesV1Candidates() []string {
	if env := strings.TrimSpace(os.Getenv("BITOWN_ASSETS_DIR")); env != "" {
		dir := filepath.Join(filepath.Clean(env), "sprites-v1")
		if abs, err := filepath.Abs(dir); err == nil {
			return []string{abs}
		}
		return []string{dir}
	}
	dirs := []string{
		filepath.Join("..", "assets", "sprites-v1"),
		filepath.Join("..", "..", "assets", "sprites-v1"),
		filepath.Join("..", "..", "..", "assets", "sprites-v1"),
		filepath.Join("/assets", "sprites-v1"),
	}
	out := make([]string, 0, len(dirs))
	for _, dir := range dirs {
		if abs, err := filepath.Abs(dir); err == nil {
			out = append(out, abs)
		} else {
			out = append(out, dir)
		}
	}
	return out
}

func atlasRootExists(dir string) bool {
	root, err := os.OpenRoot(dir)
	if err != nil {
		return false
	}
	defer func() { _ = root.Close() }()
	f, err := root.Open("atlas/sprites_v1_atlas.json")
	if err != nil {
		return false
	}
	_ = f.Close()
	return true
}

// BasesForTag returns atlas folder keys for a catalog tag (road, tree, ...).
func (a *Atlas) BasesForTag(tag string) []string {
	if a == nil || len(a.BasesByTag) == 0 {
		return nil
	}
	return a.BasesByTag[tag]
}

const roadSpriteBase = "sprites/DefineSprite_702_mcRoad"

// PickRoadKey chooses an mcRoad autotile frame from grid neighbor connectivity.
// Frame roles (DefineSprite_702_mcRoad): 3=EW, 6=NS, 1/2/4/5=corner stubs.
// Crossings and T-junctions keep an EW/NS spine tile; v00 is used for stability.
func (a *Atlas) PickRoadKey(n, e, s, w bool, x, y int) string {
	if a == nil {
		return ""
	}
	idx := roadFrameIndex(n, e, s, w, x, y)
	key := fmt.Sprintf("%s/%d_v00.png", roadSpriteBase, idx)
	if _, ok := a.Frames[key]; ok {
		return key
	}
	fallback := roadSpriteBase + "/3_v00.png"
	if _, ok := a.Frames[fallback]; ok {
		return fallback
	}
	return ""
}

func roadFrameIndex(n, e, s, w bool, x, y int) int {
	ew := e && w
	ns := n && s
	switch {
	case ew && ns:
		// Full cross: alternate spine orientation for visual variety.
		if (x+y)%2 == 0 {
			return 3
		}
		return 6
	case ew && (n || s):
		// T with east-west bar.
		return 3
	case ns && (e || w):
		// T with north-south bar.
		return 6
	case ew:
		return 3
	case ns:
		return 6
	case n && e:
		return 1
	case e && s:
		return 2
	case s && w:
		return 4
	case w && n:
		return 5
	case e || w:
		return 3
	case n || s:
		return 6
	default:
		return 3
	}
}

// PickKeyForTag chooses a deterministic frame for the tag.
// Empty catalogs return "" (callers must fall back explicitly).
func (a *Atlas) PickKeyForTag(tag string, seed uint32) string {
	bases := a.BasesForTag(tag)
	if len(bases) == 0 {
		return ""
	}
	idx := seed % uint32(len(bases)) //#nosec G115 -- catalog length is bounded
	base := bases[idx]
	if _, building := mapBuildingTags[tag]; building {
		return base + "_v00.png"
	}
	key := a.frameKey(base, a.VariantSuffix(seed))
	if _, ok := a.Frames[key]; ok {
		return key
	}
	return base + "_v00.png"
}

// PickBuildingKeyForTag picks a building frame for a zone tag.
// Contract: try the zone tag, then residential; still empty means callers
// draw a rectangle fallback (not the full BuildingBases pool).
func (a *Atlas) PickBuildingKeyForTag(tag string, seed uint32) string {
	key := a.PickKeyForTag(tag, seed)
	if key == "" {
		key = a.PickKeyForTag(TagResidential, seed)
	}
	return key
}

func readRootFile(root *os.Root, name string) ([]byte, error) {
	f, err := root.Open(name)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	return io.ReadAll(f)
}

func spriteFolderBase(frameBase string) string {
	parts := strings.Split(frameBase, "/")
	if len(parts) >= 2 {
		return parts[0] + "/" + parts[1]
	}
	return frameBase
}

func (a *Atlas) frameKey(base, variantSuffix string) string {
	return base + variantSuffix
}
