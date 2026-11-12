// Завантажує CC0-ресурси з Poly Haven (https://polyhaven.com, ліцензія CC0) у client/assets/.
// Запуск: node scripts/fetch-assets.mjs   (потрібен інтернет лише один раз)
import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const ROOT = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..', 'client', 'assets');
const RES = '1k';

export const HDRIS = {
  clear: 'kloofendal_48d_partly_cloudy_puresky',
  overcast: 'kloofendal_overcast_puresky',
  misty: 'kloofendal_misty_morning_puresky',
  winter: 'snow_field_puresky',
};
export const TEXTURES = [
  'leafy_grass', 'withered_grass', 'brown_mud_02', 'muddy_tracks', 'snow_02', 'dry_ground_01',
  'asphalt_02', 'rubble', 'concrete_wall_004', 'rough_plaster_broken', 'plaster_brick_01',
  'green_metal_rust', 'rusty_metal_02', 'denim_fabric', 'bark_brown_02',
];
export const MODELS = [
  'dead_tree_trunk_02', 'rock_07', 'stone_01', 'tree_stump_01',
  'concrete_road_barrier', 'concrete_road_barrier_02', 'covered_car', 'old_tyre', 'street_lamp_01',
  'old_military_crate', 'wooden_military_crate', 'ammo_box', 'metal_jerrycan_green', 'barrel_03',
  'portable_generator', 'cement_bag',
];

const DRY = process.argv.includes('--dry');
let total = 0;
async function j(url) { const r = await fetch(url); if (!r.ok) throw new Error(url + ' ' + r.status); return r.json(); }
async function dl(url, dest, size) {
  total += size || 0;
  if (DRY) return;
  if (fs.existsSync(dest)) return;
  fs.mkdirSync(path.dirname(dest), { recursive: true });
  const r = await fetch(url);
  if (!r.ok) throw new Error(url + ' ' + r.status);
  fs.writeFileSync(dest, Buffer.from(await r.arrayBuffer()));
}

for (const [key, id] of Object.entries(HDRIS)) {
  const f = await j(`https://api.polyhaven.com/files/${id}`);
  const h = f.hdri[RES].hdr;
  await dl(h.url, path.join(ROOT, 'hdri', `${key}.hdr`), h.size);
}
for (const id of TEXTURES) {
  const f = await j(`https://api.polyhaven.com/files/${id}`);
  const maps = { diff: f.Diffuse, nor: f.nor_gl, rough: f.Rough };
  for (const [k, m] of Object.entries(maps)) {
    if (!m) continue;
    const e = m[RES].jpg || m[RES].png;
    await dl(e.url, path.join(ROOT, 'tex', id, `${k}.jpg`), e.size);
  }
}
for (const id of MODELS) {
  const f = await j(`https://api.polyhaven.com/files/${id}`);
  const g = f.gltf[RES].gltf;
  await dl(g.url, path.join(ROOT, 'models', id, `${id}.gltf`), g.size);
  for (const [rel, inc] of Object.entries(g.include || {})) await dl(inc.url, path.join(ROOT, 'models', id, rel), inc.size);
}
console.log(`${DRY ? 'Буде завантажено' : 'Готово'}: ${(total / 1048576).toFixed(1)} МБ`);
