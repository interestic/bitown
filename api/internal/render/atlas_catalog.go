package render

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strings"
)

type loadedCatalog struct {
	buildingBases   []string
	basesByTag      map[string][]string
	tierByFolder    map[string]int
	unlockByFolder  map[string]FolderUnlock
	libraryByFolder map[string]LibraryRef
}

func loadBuildingsCatalog(root *os.Root, frameBases []string) (loadedCatalog, error) {
	data, err := readRootFile(root, "buildings.json")
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return loadedCatalog{}, fmt.Errorf("%w: run scripts/generate_buildings_manifest.py", ErrBuildingsManifestMissing)
		}
		return loadedCatalog{}, fmt.Errorf("read buildings manifest: %w", err)
	}

	var manifest buildingsManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return loadedCatalog{}, fmt.Errorf("parse buildings manifest: %w", err)
	}
	if manifest.Version < 2 {
		return loadedCatalog{}, fmt.Errorf("buildings.json version must be >= 2")
	}
	if len(manifest.BuildingBases) == 0 {
		return loadedCatalog{}, ErrBuildingsManifestEmpty
	}
	if len(manifest.Entries) == 0 {
		return loadedCatalog{}, fmt.Errorf("buildings.json has no catalog entries")
	}

	tagByFolder := make(map[string]string, len(manifest.Entries))
	tierByFolder := make(map[string]int, len(manifest.Entries))
	unlockByFolder := make(map[string]FolderUnlock, len(manifest.Entries))
	libraryByFolder := make(map[string]LibraryRef, len(manifest.Entries))
	for _, entry := range manifest.Entries {
		if entry.Base == "" || entry.Tag == "" {
			continue
		}
		tagByFolder[entry.Base] = entry.Tag
		if entry.Tier != nil {
			tier := *entry.Tier
			if tier < 0 {
				tier = 0
			}
			if tier > 3 {
				tier = 3
			}
			tierByFolder[entry.Base] = tier
		}
		if entry.Unlock != nil {
			unlockByFolder[entry.Base] = *entry.Unlock
		}
		if entry.LibraryRef != nil && entry.LibraryRef.LibraryID > 0 {
			libraryByFolder[entry.Base] = *entry.LibraryRef
		}
	}

	allowed := make(map[string]struct{}, len(manifest.BuildingBases))
	for _, folderBase := range manifest.BuildingBases {
		if deniedBuildingFolder(folderBase) {
			continue
		}
		if tag, ok := tagByFolder[folderBase]; ok {
			if _, building := mapBuildingTags[tag]; !building {
				continue
			}
		} else {
			continue
		}
		allowed[folderBase] = struct{}{}
	}

	filtered := make([]string, 0, len(frameBases))
	for _, frameBase := range frameBases {
		folder := spriteFolderBase(frameBase)
		if _, ok := allowed[folder]; !ok {
			continue
		}
		if deniedBuildingFolder(folder) {
			continue
		}
		filtered = append(filtered, frameBase)
	}
	if len(filtered) == 0 {
		return loadedCatalog{}, fmt.Errorf("no atlas frames matched buildings manifest")
	}

	if len(tagByFolder) == 0 {
		for tag, folders := range manifest.BasesByTag {
			for _, folder := range folders {
				tagByFolder[folder] = tag
			}
		}
	}

	byTag := make(map[string][]string)
	for _, frameBase := range frameBases {
		folder := spriteFolderBase(frameBase)
		tag, ok := tagByFolder[folder]
		if !ok || tag == "" {
			continue
		}
		if deniedBuildingFolder(folder) {
			if _, building := mapBuildingTags[tag]; building {
				continue
			}
		}
		if _, building := mapBuildingTags[tag]; building {
			if _, ok := allowed[folder]; !ok {
				continue
			}
		}
		byTag[tag] = append(byTag[tag], frameBase)
	}

	return loadedCatalog{
		buildingBases:   filtered,
		basesByTag:      byTag,
		tierByFolder:    tierByFolder,
		unlockByFolder:  unlockByFolder,
		libraryByFolder: libraryByFolder,
	}, nil
}

// maxSingleLotBuildingW caps atlas frame width for 1-lot placement.
// Native FFDec sizes (post-normalize, no 96px downscale) let library primaries
// overhang several cells — mcHouse3 is ~238px wide. Keep a soft ceiling so
// accidental full-stage / UI sheets stay out of the pool.
const maxSingleLotBuildingW = 280

func filterOversizedSingleLotBuildings(catalog loadedCatalog, frames map[string]frameRect) loadedCatalog {
	buildingBases := filterNarrowFrameBases(catalog.buildingBases, frames)
	byTag := make(map[string][]string, len(catalog.basesByTag))
	for tag, bases := range catalog.basesByTag {
		if _, building := mapBuildingTags[tag]; building {
			byTag[tag] = filterNarrowFrameBases(bases, frames)
			continue
		}
		byTag[tag] = append([]string(nil), bases...)
	}
	return loadedCatalog{
		buildingBases:   buildingBases,
		basesByTag:      byTag,
		tierByFolder:    catalog.tierByFolder,
		unlockByFolder:  catalog.unlockByFolder,
		libraryByFolder: catalog.libraryByFolder,
	}
}

func filterNarrowFrameBases(bases []string, frames map[string]frameRect) []string {
	out := make([]string, 0, len(bases))
	for _, frameBase := range bases {
		if frameBaseWiderThan(frameBase, frames, maxSingleLotBuildingW) {
			continue
		}
		out = append(out, frameBase)
	}
	return out
}

func frameBaseWiderThan(frameBase string, frames map[string]frameRect, maxW int) bool {
	for _, suf := range []string{"_v00.png", "_v01.png", "_v02.png", "_v03.png"} {
		if fr, ok := frames[frameBase+suf]; ok && fr.W > maxW {
			return true
		}
	}
	folder := spriteFolderBase(frameBase)
	prefix := folder + "/"
	for key, fr := range frames {
		if strings.HasPrefix(key, prefix) && fr.W > maxW {
			return true
		}
	}
	return false
}

func deniedBuildingFolder(folder string) bool {
	lower := strings.ToLower(folder)
	for _, token := range buildingDenySubstr {
		if strings.Contains(lower, strings.ToLower(token)) {
			return true
		}
	}
	return false
}
