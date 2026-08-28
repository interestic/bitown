package render

// Stamp metadata in buildings.json describes logical footprint + foot contract
// per sprite folder (docs/map-building-placement.md phase B).
const (
	StampKindMiniFoot       = "mini_foot"
	StampKindArterialYard   = "arterial_yard"
	StampKindLandmarkCenter = "landmark_center"
	StampKindPackedMini     = "packed_mini"
)

const (
	NudgeProfileDefault       = "default"
	NudgeProfileLandmark      = "landmark"
	NudgeProfileArterialYard  = "arterial_yard"
	NudgeProfileMidriseTower  = "midrise_tower"
	NudgeProfileOverlapYard   = "overlap_yard"
	NudgeProfilePackedNoCross = "packed_no_cross"
	NudgeProfilePackedCross   = "packed_cross"
	NudgeProfileArterialDown  = "arterial_down"
)

// Screen +Y fallbacks when foot_extra_y is unset (legacy nudge_profile only).
const StampPackedNoCrossDownY = 0
const StampOverlapYardDownY = 0
const StampLandmarkUpY = -10
const StampMidriseTowerUpY = -10
const StampArterialDownY = 0

// stampFootExtraY returns explicit per-folder Y only (lab tuning). Unset → 0 so
// sandbox stage 21 uses core kind nudges + atlas anchors without profile extras.
func stampFootExtraY(meta StampMeta) int {
	if meta.FootExtraY != nil {
		return *meta.FootExtraY
	}
	return 0
}

// StampMeta is the per-folder stamp contract from buildings.json.
type StampMeta struct {
	Kind           string `json:"kind"`
	FootprintMinis int    `json:"footprint_minis"`
	CrossReserve   bool   `json:"cross_reserve"`
	NudgeProfile   string `json:"nudge_profile,omitempty"`
	// FootExtraX/Y are sandbox-tuned screen offsets per folder (phase B1+).
	// When set, they override nudge_profile shared constants so tuning one
	// sprite family does not move others that share a profile name.
	FootExtraX *int `json:"foot_extra_x,omitempty"`
	FootExtraY *int `json:"foot_extra_y,omitempty"`
}

// StampForKey returns catalog stamp metadata for a sprite frame key.
func (a *Atlas) StampForKey(key string) (StampMeta, bool) {
	if a == nil || key == "" {
		return StampMeta{}, false
	}
	folder := spriteFolderBase(key)
	if meta, ok := a.StampByFolder[folder]; ok && meta.Kind != "" {
		return meta, true
	}
	tier := DefaultBuildingTier
	if t, ok := a.TierByFolder[folder]; ok {
		tier = t
	}
	if tier >= 3 {
		return StampMeta{
			Kind:            StampKindLandmarkCenter,
			FootprintMinis:  16,
			CrossReserve:    true,
			NudgeProfile:    NudgeProfileLandmark,
		}, true
	}
	if tier <= 1 {
		return StampMeta{
			Kind:            StampKindMiniFoot,
			FootprintMinis:  1,
			CrossReserve:    false,
			NudgeProfile:    NudgeProfileDefault,
		}, true
	}
	return StampMeta{
		Kind:            StampKindArterialYard,
		FootprintMinis:  1,
		CrossReserve:    true,
		NudgeProfile:    NudgeProfileArterialYard,
	}, true
}

// buildingStampFoot is overlayFoot + stamp-kind nudges for a density lot building.
func buildingStampFoot(atlas *Atlas, key string, lotX, lotY int, roadless bool) (footX, footY int) {
	footX, footY = overlayFoot(lotX, lotY, overlayLift(roadless))
	if atlas == nil || key == "" {
		return footX, footY
	}
	stamp, _ := atlas.StampForKey(key)
	return applyBuildingStampNudges(footX, footY, lotX, lotY, roadless, stamp)
}

// applyBuildingStampNudges applies mini / arterial nudges per stamp.kind.
func applyBuildingStampNudges(footX, footY, lotX, lotY int, roadless bool, stamp StampMeta) (int, int) {
	switch stamp.Kind {
	case StampKindMiniFoot:
		footX = applyWestMiniStampNudge(footX, lotX, lotY)
		footX = applyNorthMiniStampNudgeX(footX, lotX, lotY)
		footY = applyNorthMiniStampNudge(footY, lotX, lotY)
		if roadless {
			footY = applySEMiniStampNudge(footY, lotX, lotY)
			footY = applyEWMiniStampNudge(footY, lotX, lotY)
		}
	case StampKindArterialYard:
		footX = applyWestMiniStampNudge(footX, lotX, lotY)
		footX = applyNorthMiniStampNudgeX(footX, lotX, lotY)
		footX = applyEastMiniStampNudge(footX, lotX, lotY)
		footY = applyNorthMiniStampNudge(footY, lotX, lotY)
		footY = applySEMiniStampNudge(footY, lotX, lotY)
		footY = applyEWMiniStampNudge(footY, lotX, lotY)
		if !roadless {
			footX, footY = applyArterialYardStampNudge(footX, footY, lotX, lotY)
		}
	case StampKindPackedMini:
		footX = applyWestMiniStampNudge(footX, lotX, lotY)
		footX = applyNorthMiniStampNudgeX(footX, lotX, lotY)
		footX = applyEastMiniStampNudge(footX, lotX, lotY)
		footY = applyNorthMiniStampNudge(footY, lotX, lotY)
		footY = applySEMiniStampNudge(footY, lotX, lotY)
		footY = applyEWMiniStampNudge(footY, lotX, lotY)
		if !roadless {
			footX, footY = applyArterialYardStampNudge(footX, footY, lotX, lotY)
		}
	case StampKindLandmarkCenter:
		// Regular lots should not pick landmarks; keep grass-top foot only.
	default:
		if roadless {
			return applyBuildingStampNudges(footX, footY, lotX, lotY, roadless, StampMeta{Kind: StampKindMiniFoot})
		}
		return applyBuildingStampNudges(footX, footY, lotX, lotY, roadless, StampMeta{Kind: StampKindArterialYard})
	}
	return footX, footY
}

// applyMapBaseStampNudges applies sandbox map-base mini-index nudges per stamp.kind.
// packsFullMini forces packed_mini contract (lab ¼ draw over mini_foot catalog entry).
func applyMapBaseStampNudges(footX, footY, mi int, kind string, crossArm bool) (int, int) {
	switch kind {
	case StampKindPackedMini:
		// Packed ¼ is a regular grid inside each mini. No plate-tip or cross-arm
		// nudges — those target single-foot arterial yards around CROSS (705).
		// CROSS is road layer; all 16 building feet stay occupied.
	case StampKindMiniFoot:
		if mi == miniSW {
			footX += westMiniStampNudgeX
		}
		if mi == miniNW {
			footX += northMiniStampNudgeX
			footY += northMiniStampNudgeY
		}
	case StampKindArterialYard:
		footX, footY = applyMiniNudgesByMiniIndex(footX, footY, mi, crossArm)
		footX, footY = applyArterialYardStampNudgeForMini(footX, footY, mi)
	case StampKindLandmarkCenter:
		// landmark feet use landmarkStampFoot; lot huts should not land here.
	default:
		footX, footY = applyMiniNudgesByMiniIndex(footX, footY, mi, crossArm)
		footX, footY = applyArterialYardStampNudgeForMini(footX, footY, mi)
	}
	return footX, footY
}

func applyMiniNudgesByMiniIndex(footX, footY, mi int, crossArm bool) (int, int) {
	if mi == miniSW {
		footX += westMiniStampNudgeX
	}
	if mi == miniNW {
		footX += northMiniStampNudgeX
		footY += northMiniStampNudgeY
	}
	if crossArm {
		footX, footY = applyCrossArmMiniNudges(footX, footY, mi)
	}
	return footX, footY
}

func applyCrossArmMiniNudges(footX, footY, mi int) (int, int) {
	if mi == miniSE {
		footY += seMiniStampNudgeY
	}
	if mi == miniSW || mi == miniNE {
		footY += ewMiniStampNudgeY
	}
	if mi == miniNE {
		footX += eastMiniStampNudgeX
	}
	return footX, footY
}
