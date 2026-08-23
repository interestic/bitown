import type {
  AtlasMeta,
  BuildingsManifest,
  CatalogEntry,
  FrameRect,
  SpriteObject,
} from "./types";
import { PLACEMENT_TAGS } from "./types";

const SPRITES_BASE = "/sprites-v1";

/** `sprites/Foo/3_v01.png` → base + frameId + color index */
const ATLAS_KEY_RE = /^(sprites\/[^/]+)\/(\d+)_v(\d+)\.png$/i;

export function atlasImageUrl(): string {
  return `${SPRITES_BASE}/atlas/sprites_v1_atlas.png`;
}

async function fetchJson<T>(url: string): Promise<T> {
  const res = await fetch(url);
  if (!res.ok) {
    throw new Error(`Failed to load ${url}: ${res.status} ${res.statusText}`);
  }
  return (await res.json()) as T;
}

function entryByBase(entries: CatalogEntry[]): Map<string, CatalogEntry> {
  const map = new Map<string, CatalogEntry>();
  for (const entry of entries) {
    if (!map.has(entry.base)) {
      map.set(entry.base, entry);
    }
  }
  return map;
}

function pickPreviewKey(colorVariants: string[]): string | null {
  if (colorVariants.length === 0) return null;
  const v00 = colorVariants.find((key) => /_v00\.png$/i.test(key));
  return v00 ?? colorVariants[0];
}

type FrameBucket = {
  base: string;
  frameId: string;
  colorVariants: string[];
};

function framesByBaseAndId(
  frames: Record<string, FrameRect>,
): Map<string, FrameBucket> {
  const buckets = new Map<string, FrameBucket>();
  for (const key of Object.keys(frames)) {
    const match = ATLAS_KEY_RE.exec(key);
    if (!match) continue;
    const [, base, frameId] = match;
    const id = `${base}#${frameId}`;
    let bucket = buckets.get(id);
    if (!bucket) {
      bucket = { base, frameId, colorVariants: [] };
      buckets.set(id, bucket);
    }
    bucket.colorVariants.push(key);
  }
  for (const bucket of buckets.values()) {
    bucket.colorVariants.sort((a, b) => {
      const av = Number(/_v(\d+)\.png$/i.exec(a)?.[1] ?? 0);
      const bv = Number(/_v(\d+)\.png$/i.exec(b)?.[1] ?? 0);
      return av - bv;
    });
  }
  return buckets;
}

export type PlacementCatalog = {
  manifest: BuildingsManifest;
  atlas: AtlasMeta;
  objects: SpriteObject[];
  byTag: Record<string, SpriteObject[]>;
};

export async function loadPlacementCatalog(): Promise<PlacementCatalog> {
  const [manifest, atlas] = await Promise.all([
    fetchJson<BuildingsManifest>(`${SPRITES_BASE}/buildings.json`),
    fetchJson<AtlasMeta>(`${SPRITES_BASE}/atlas/sprites_v1_atlas.json`),
  ]);

  const buildingPool = new Set(manifest.building_bases);
  const entries = entryByBase(manifest.entries);
  const frameBuckets = framesByBaseAndId(atlas.frames);
  const objects: SpriteObject[] = [];
  const seen = new Set<string>();

  const tagOrder = [
    ...PLACEMENT_TAGS,
    ...Object.keys(manifest.bases_by_tag).filter(
      (tag) => !PLACEMENT_TAGS.includes(tag as (typeof PLACEMENT_TAGS)[number]),
    ),
  ];

  const bucketsByBase = new Map<string, FrameBucket[]>();
  for (const bucket of frameBuckets.values()) {
    const list = bucketsByBase.get(bucket.base) ?? [];
    list.push(bucket);
    bucketsByBase.set(bucket.base, list);
  }
  for (const list of bucketsByBase.values()) {
    list.sort((a, b) => Number(a.frameId) - Number(b.frameId));
  }

  for (const tag of tagOrder) {
    const bases = manifest.bases_by_tag[tag] ?? [];
    for (const base of bases) {
      const bucketsForBase = bucketsByBase.get(base) ?? [];
      if (bucketsForBase.length === 0) continue;
      const meta = entries.get(base);
      for (const bucket of bucketsForBase) {
        const id = `${bucket.base}#${bucket.frameId}`;
        if (seen.has(id)) continue;
        const previewKey = pickPreviewKey(bucket.colorVariants);
        if (!previewKey) continue;
        const frame = atlas.frames[previewKey];
        if (!frame) continue;
        seen.add(id);
        objects.push({
          base: bucket.base,
          frameId: bucket.frameId,
          tag,
          group: meta?.group ?? "other",
          tier: meta?.tier,
          colorVariants: bucket.colorVariants,
          previewKey,
          frame,
          inBuildingPool: buildingPool.has(bucket.base),
        });
      }
    }
  }

  const byTag: Record<string, SpriteObject[]> = {};
  for (const obj of objects) {
    (byTag[obj.tag] ??= []).push(obj);
  }

  return { manifest, atlas, objects, byTag };
}

export function shortBaseName(base: string): string {
  const parts = base.split("/");
  return parts[parts.length - 1] ?? base;
}

export function objectLabel(obj: SpriteObject): string {
  return `${shortBaseName(obj.base)}/${obj.frameId}`;
}

export function objectKey(obj: SpriteObject): string {
  return `${obj.base}#${obj.frameId}`;
}
