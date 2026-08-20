package render

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strings"

	"github.com/interestic/bitown/internal/citycore"
)

const mapRendererVersion = "map-v19"

var (
	ErrAtlasNotFound            = errors.New("sprites-v1 atlas directory not found")
	ErrBuildingsManifestMissing = errors.New("buildings.json missing")
	ErrBuildingsManifestEmpty   = errors.New("buildings.json has no building_bases")
)

// MapEntityTag returns a strong ETag for map.png derived from city state and atlas revision.
// It does not render the PNG.
func MapEntityTag(city *citycore.City) (string, error) {
	mode, revision, err := mapRenderIdentity()
	if err != nil {
		return "", err
	}

	sum := sha256.New()
	_, _ = fmt.Fprintf(sum, "%s/%s/%s/%d/%d/%d/%d/%d/%d/%d", mapRendererVersion, mode, revision, city.Pop, city.Ind, city.Tra, city.Sec, city.Env, city.Com, len(city.Slug))
	_, _ = sum.Write([]byte(city.Slug))
	return `"` + hex.EncodeToString(sum.Sum(nil)[:16]) + `"`, nil
}

func mapRenderIdentity() (mode, revision string, err error) {
	atlas, atlasErr := loadAtlas()
	if atlasErr != nil {
		if atlasRequired() {
			return "", "", fmt.Errorf("atlas required: %w", atlasErr)
		}
		return "fallback", fallbackRenderRevision(atlasErr), nil
	}
	return "atlas", atlas.revision, nil
}

func fallbackRenderRevision(atlasErr error) string {
	sum := sha256.New()
	_, _ = fmt.Fprintf(sum, "fallback/%s", atlasFallbackReason(atlasErr))
	return hex.EncodeToString(sum.Sum(nil)[:8])
}

func atlasFallbackReason(err error) string {
	switch {
	case errors.Is(err, ErrAtlasNotFound):
		return "atlas_not_found"
	case errors.Is(err, ErrBuildingsManifestMissing):
		return "buildings_manifest_missing"
	case errors.Is(err, ErrBuildingsManifestEmpty):
		return "buildings_manifest_empty"
	default:
		msg := err.Error()
		if strings.Contains(msg, "no atlas frames matched") {
			return "buildings_manifest_mismatch"
		}
		if strings.Contains(msg, "buildings manifest") {
			return "buildings_manifest_invalid"
		}
		return "atlas_load_error"
	}
}

func wrapAtlasError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, ErrAtlasNotFound) || errors.Is(err, ErrBuildingsManifestMissing) || errors.Is(err, ErrBuildingsManifestEmpty) {
		return err
	}
	msg := err.Error()
	switch {
	case strings.Contains(msg, "sprites-v1 atlas not found"):
		return fmt.Errorf("%w: %s", ErrAtlasNotFound, msg)
	case strings.Contains(msg, "read buildings manifest"):
		if errors.Is(err, fs.ErrNotExist) || missingFileMessage(msg) {
			return fmt.Errorf("%w: %s", ErrBuildingsManifestMissing, msg)
		}
		return err
	case strings.Contains(msg, "buildings manifest is empty") || strings.Contains(msg, "has no building_bases"):
		return fmt.Errorf("%w: %s", ErrBuildingsManifestEmpty, msg)
	default:
		return err
	}
}

func missingFileMessage(msg string) bool {
	return strings.Contains(msg, "no such file") ||
		strings.Contains(msg, "not found") ||
		strings.Contains(msg, "file does not exist")
}

// MatchIfNoneMatch reports whether an If-None-Match header matches etag.
// It accepts a single tag, a comma-separated list, weak validators, and "*".
func MatchIfNoneMatch(header, etag string) bool {
	if header == "" || etag == "" {
		return false
	}
	want := stripWeakETag(etag)
	for _, part := range strings.Split(header, ",") {
		token := strings.TrimSpace(part)
		if token == "*" {
			return true
		}
		if stripWeakETag(token) == want {
			return true
		}
	}
	return false
}

func stripWeakETag(v string) string {
	v = strings.TrimSpace(v)
	if len(v) >= 2 && (v[0] == 'W' || v[0] == 'w') && v[1] == '/' {
		return strings.TrimSpace(v[2:])
	}
	return v
}

// ResetAtlasCacheForTest clears the in-process atlas cache between tests.
func ResetAtlasCacheForTest() {
	atlasMu.Lock()
	atlasInst = nil
	atlasMu.Unlock()
}

func rootFileRevision(root *os.Root, name string) (string, error) {
	info, err := root.Stat(name)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s:%d:%d", name, info.ModTime().UnixNano(), info.Size()), nil
}

func atlasDiskRevision(root *os.Root) (string, error) {
	parts := make([]string, 0, 3)
	for _, name := range []string{
		"atlas/sprites_v1_atlas.json",
		"atlas/sprites_v1_atlas.png",
		"buildings.json",
	} {
		part, err := rootFileRevision(root, name)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) && name == "buildings.json" {
				return "", fmt.Errorf("read buildings manifest: %w", fs.ErrNotExist)
			}
			return "", err
		}
		parts = append(parts, part)
	}
	return strings.Join(parts, "|"), nil
}
