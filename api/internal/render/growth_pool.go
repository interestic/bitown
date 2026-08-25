package render

import (
	"sort"

	"github.com/interestic/bitown/internal/citycore"
)

// Growth-tier thresholds remapped from Cs.hx POP_* onto bitown's ~500 pop scale.
// See docs/map-building-growth.md.
const (
	DefaultBuildingTier = 1

	popTierPeon   = 40
	popTierNormal = 120
	popTierHuge   = 350

	// Outer lots (dist² from center): drop max tier by one; far rim caps at 1
	// so huge cities still keep house-scale outskirts (big_city, #49).
	// Scaled with map half-extent so 30×30 crops behave like the old 20×20.
)

func outerLotDist2() int {
	h := mapCols / 2
	return (h * h) / 2
}

// maxTierForPop returns the highest building tier allowed for a city pop.
func maxTierForPop(pop int) int {
	switch {
	case pop < popTierPeon:
		return 1
	case pop < popTierNormal:
		return 2
	default:
		return 3
	}
}

// landmarkMixPermille is the chance (0..1000) to draw from the landmark pool.
// peon/normal: 0; big: small; huge: higher; sec/com give a weak boost.
func landmarkMixPermille(pop, sec, com int) int {
	if pop < popTierNormal {
		return 0
	}
	chance := 15
	if pop >= popTierHuge {
		chance = 45
	}
	chance += sec / 50
	chance += com / 25
	if chance > 120 {
		chance = 120
	}
	return chance
}

func (a *Atlas) folderTier(folder string) int {
	if a == nil || a.TierByFolder == nil {
		return DefaultBuildingTier
	}
	if tier, ok := a.TierByFolder[folder]; ok {
		return tier
	}
	return DefaultBuildingTier
}

func (a *Atlas) frameTier(frameBase string) int {
	return a.folderTier(spriteFolderBase(frameBase))
}

func filterBasesByMaxTier(a *Atlas, bases []string, maxTier int) []string {
	out := make([]string, 0, len(bases))
	for _, base := range bases {
		if a.frameTier(base) <= maxTier {
			out = append(out, base)
		}
	}
	return out
}

// tierPickWeight prefers low tiers at low pop and higher tiers as the city grows.
func tierPickWeight(tier, pop int) int {
	if tier < 0 {
		tier = 0
	}
	if tier > 3 {
		tier = 3
	}
	switch {
	case pop < popTierPeon:
		return 4 - tier
	case pop < popTierNormal:
		w := 3 - tier
		if w < 1 {
			w = 1
		}
		return w
	case pop < popTierHuge:
		// Big: peak at mid-rise. Tier 3 is unlocked but must not dominate
		// the instant pop=120 threshold (outer lots also jump maxTier 1→2).
		if tier >= 3 {
			return 2
		}
		return tier + 1
	default:
		return tier*2 + 1
	}
}

// tierPickWeightForTag keeps houses common in residential zones after
// skyscrapers unlock. Commercial / industrial keep the generic curve.
func tierPickWeightForTag(tag string, tier, pop int) int {
	if tag != TagResidential {
		return tierPickWeight(tier, pop)
	}
	if tier < 0 {
		tier = 0
	}
	if tier > 3 {
		tier = 3
	}
	switch {
	case pop < popTierNormal:
		return tierPickWeight(tier, pop)
	case pop < popTierHuge:
		// Big: the residential pool has one tier-3 tower and no tier 2.
		// Generic weights would give that tower ~40% of residential lots.
		switch tier {
		case 0:
			return 4
		case 1:
			return 5
		case 2:
			return 2
		default:
			return 1
		}
	default:
		switch tier {
		case 0:
			return 2
		case 1:
			return 3
		case 2:
			return 4
		default:
			return 4
		}
	}
}

// weightedIndex returns the chosen index for positive weights, or -1 if none.
func weightedIndex(weights []int, seed uint32) int {
	sum := 0
	for _, w := range weights {
		if w > 0 {
			sum += w
		}
	}
	if sum <= 0 {
		return -1
	}
	r := int(seed % uint32(sum)) //#nosec G115
	acc := 0
	for i, w := range weights {
		if w <= 0 {
			continue
		}
		acc += w
		if r < acc {
			return i
		}
	}
	return len(weights) - 1
}

func (a *Atlas) pickBuildingFrameForTagAvoiding(city *citycore.City, tag string, maxTier, pop, densityMax int, seed uint32, avoid map[string]struct{}) string {
	if a == nil {
		return ""
	}
	bases := filterBasesByMaxTier(a, a.BasesForTag(tag), maxTier)
	bases = filterBasesByUnlock(a, bases, city)
	bases = filterBasesByUpdateLib(a, bases, densityMax, pop)
	if len(bases) == 0 {
		return ""
	}
	// Folder first (issue #49): multi-frame clips must not flood weights.
	// Tier first within that: one weight per tier so a huge tall catalog
	// cannot drown mid-rise after bbox_h>=80 → tier 3 (#47/#48 review).
	folders := filterFoldersAvoiding(uniqueSpriteFolders(bases), avoid)
	folders = filterFoldersByUnlock(a, folders, city)
	folders = filterFoldersByUpdateLib(a, folders, densityMax, pop)
	folder := pickFolderByTierThenUniform(a, folders, pop, tag, seed)
	if folder == "" {
		return ""
	}
	var inFolder []string
	for _, base := range bases {
		if spriteFolderBase(base) == folder {
			inFolder = append(inFolder, base)
		}
	}
	if len(inFolder) == 0 {
		return ""
	}
	picked := inFolder[int((seed>>8)%uint32(len(inFolder)))] //#nosec G115
	return picked + "_v00.png"
}

func filterBasesByUpdateLib(a *Atlas, bases []string, densityMax, cityPop int) []string {
	if a == nil || len(a.LibraryByFolder) == 0 {
		return bases
	}
	out := make([]string, 0, len(bases))
	for _, base := range bases {
		folder := spriteFolderBase(base)
		ref, ok := a.LibraryByFolder[folder]
		if !ok || updateLibHouseUnlocked(ref.LibraryID, densityMax, cityPop) {
			out = append(out, base)
		}
	}
	return out
}

func filterFoldersByUpdateLib(a *Atlas, folders []string, densityMax, cityPop int) []string {
	if a == nil || len(a.LibraryByFolder) == 0 {
		return folders
	}
	out := make([]string, 0, len(folders))
	for _, folder := range folders {
		ref, ok := a.LibraryByFolder[folder]
		if !ok || updateLibHouseUnlocked(ref.LibraryID, densityMax, cityPop) {
			out = append(out, folder)
		}
	}
	return out
}

func filterFoldersAvoiding(folders []string, avoid map[string]struct{}) []string {
	if len(avoid) == 0 {
		return folders
	}
	out := make([]string, 0, len(folders))
	for _, folder := range folders {
		if _, skip := avoid[folder]; skip {
			continue
		}
		out = append(out, folder)
	}
	if len(out) == 0 {
		return folders
	}
	return out
}

// pickFolderByTierThenUniform weights each distinct tier once, then picks a
// folder uniformly inside the chosen tier.
func pickFolderByTierThenUniform(a *Atlas, folders []string, pop int, tag string, seed uint32) string {
	if len(folders) == 0 {
		return ""
	}
	byTier := make(map[int][]string, 4)
	for _, folder := range folders {
		t := a.folderTier(folder)
		byTier[t] = append(byTier[t], folder)
	}
	tiers := make([]int, 0, len(byTier))
	for t := range byTier {
		tiers = append(tiers, t)
	}
	sort.Ints(tiers)
	weights := make([]int, len(tiers))
	for i, t := range tiers {
		weights[i] = tierPickWeightForTag(tag, t, pop)
	}
	ti := weightedIndex(weights, seed)
	if ti < 0 {
		return folders[int(seed%uint32(len(folders)))] //#nosec G115
	}
	cands := byTier[tiers[ti]]
	return cands[int((seed^0x9e3779b9)%uint32(len(cands)))] //#nosec G115
}

func uniqueSpriteFolders(bases []string) []string {
	seen := make(map[string]struct{}, len(bases))
	out := make([]string, 0, len(bases))
	for _, base := range bases {
		folder := spriteFolderBase(base)
		if folder == "" {
			continue
		}
		if _, ok := seen[folder]; ok {
			continue
		}
		seen[folder] = struct{}{}
		out = append(out, folder)
	}
	return out
}

// PickBuildingKeyForLot picks a building frame using zone tag, local square
// density (Game.hx) + periphery caps, updateLib house-library gates, and
// optional landmark mix. Falls back to residential within the same tier cap;
// empty means callers draw a rectangle (do not bypass via PickBuildingKeyForTag).
func (a *Atlas) PickBuildingKeyForLot(city *citycore.City, tag string, x, y int, seed uint32) string {
	local, densityMax := 0, 0
	if city != nil {
		dens := genMapPop(city.Pop.Int(), newMapRNG(city.Slug.String()))
		local = localDensityAt(dens, x, y)
		densityMax = dens.max
	}
	return a.pickBuildingKeyForLot(city, tag, x, y, seed, nil, local, densityMax)
}

func (a *Atlas) pickBuildingKeyForLot(city *citycore.City, tag string, x, y int, seed uint32, avoid map[string]struct{}, localDensity, densityMax int) string {
	if a == nil {
		return ""
	}
	if city == nil {
		return a.PickBuildingKeyForTag(tag, seed)
	}

	pop := city.Pop.Int()
	maxTier := maxTierForLotWithLocal(localDensity, pop, city.Ind.Int(), city.Com.Int(), x, y, tag)
	const sectorMid = 50
	if tag == TagIndustrial && city.Ind.Int() < sectorMid && maxTier > 1 {
		maxTier = 1
	}
	if tag == TagCommercial && city.Com.Int() < sectorMid && maxTier > 1 {
		maxTier = 1
	}

	if maxTier >= 3 {
		chance := landmarkMixPermille(pop, city.Sec.Int(), city.Com.Int())
		// Fold high and low bits — plain seed%1000 is biased for hashCell coords.
		roll := int(((seed >> 16) ^ (seed >> 8) ^ seed) % 1000) //#nosec G115
		if chance > 0 && roll < chance {
			if key := a.pickBuildingFrameForTagAvoiding(city, TagLandmark, maxTier, pop, densityMax, seed^0x9e3779b9, avoid); key != "" {
				return key
			}
		}
	}

	key := a.pickBuildingFrameForTagAvoiding(city, tag, maxTier, pop, densityMax, seed, avoid)
	if key == "" && tag != TagResidential {
		key = a.pickBuildingFrameForTagAvoiding(city, TagResidential, maxTier, pop, densityMax, seed, avoid)
	}
	return key
}

var cardinalDirs = [4][2]int{{0, -1}, {0, 1}, {-1, 0}, {1, 0}}

// assignBuildingKeys picks frames in raster order, skipping high-tier folders
// already used on a cardinal neighbor. Low-tier houses may still clump.
func assignBuildingKeys(atlas *Atlas, city *citycore.City, occ map[[2]int]lotCell, densityMax int) map[[2]int]string {
	if atlas == nil || city == nil {
		return nil
	}
	type xy struct{ x, y int }
	lots := make([]xy, 0, len(occ))
	for pos, lot := range occ {
		if lot.use == lotBuilding {
			lots = append(lots, xy{pos[0], pos[1]}) //#nosec G602 -- map key is [2]int
		}
	}
	sort.Slice(lots, func(i, j int) bool {
		if lots[i].y != lots[j].y {
			return lots[i].y < lots[j].y
		}
		return lots[i].x < lots[j].x
	})
	keys := make(map[[2]int]string, len(lots))
	slug := city.Slug.String()
	for _, p := range lots {
		avoid := make(map[string]struct{}, 4)
		for _, d := range cardinalDirs {
			nb := keys[[2]int{p.x + d[0], p.y + d[1]}]
			if nb == "" || !avoidRepeatNeighbor(atlas, nb) {
				continue
			}
			avoid[spriteFolderBase(nb)] = struct{}{}
		}
		lot := occ[[2]int{p.x, p.y}]
		seed := hashCell(slug, p.x, p.y)
		keys[[2]int{p.x, p.y}] = atlas.pickBuildingKeyForLot(city, lot.tag, p.x, p.y, seed, avoid, lot.density, densityMax)
	}
	return keys
}

func avoidRepeatNeighbor(a *Atlas, key string) bool {
	if a == nil || key == "" {
		return false
	}
	return a.folderTier(spriteFolderBase(key)) >= 2
}
