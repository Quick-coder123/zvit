// Завантаження фотоскановaних ресурсів (Poly Haven, CC0) з прогресом.
import * as THREE from 'three';
import { GLTFLoader } from 'three/addons/loaders/GLTFLoader.js';
import { RGBELoader } from 'three/addons/loaders/RGBELoader.js';

const TEX = ['leafy_grass', 'withered_grass', 'brown_mud_02', 'muddy_tracks', 'snow_02', 'dry_ground_01', 'asphalt_02', 'rubble',
  'concrete_wall_004', 'rough_plaster_broken', 'plaster_brick_01', 'green_metal_rust', 'rusty_metal_02', 'denim_fabric', 'bark_brown_02'];
const GLB = ['dead_tree_trunk_02', 'rock_07', 'stone_01', 'tree_stump_01',
  'concrete_road_barrier', 'concrete_road_barrier_02', 'covered_car', 'old_tyre', 'street_lamp_01', 'old_military_crate',
  'wooden_military_crate', 'ammo_box', 'metal_jerrycan_green', 'barrel_03', 'portable_generator', 'cement_bag'];
const HDR = ['clear', 'overcast', 'misty', 'winter'];

export const A = { tex: {}, glb: {}, hdr: {}, alpha: {} };

export async function loadAssets(renderer, onProgress) {
  const manager = new THREE.LoadingManager();
  manager.onProgress = (_u, done, total) => onProgress && onProgress(done / total);
  const tl = new THREE.TextureLoader(manager);
  const gl = new GLTFLoader(manager);
  const hl = new RGBELoader(manager);
  const aniso = Math.min(8, renderer.capabilities.getMaxAnisotropy());
  const jobs = [];
  for (const id of TEX) {
    A.tex[id] = {};
    for (const k of ['diff', 'nor', 'rough']) {
      jobs.push(tl.loadAsync(`assets/tex/${id}/${k}.jpg`).then((t) => {
        t.wrapS = t.wrapT = THREE.RepeatWrapping;
        t.anisotropy = aniso;
        if (k === 'diff') t.colorSpace = THREE.SRGBColorSpace;
        A.tex[id][k] = t;
      }).catch(() => {}));
    }
  }
  for (const id of GLB) {
    jobs.push(gl.loadAsync(`assets/glb/${id}.glb`).then((g) => {
      // одна сітка + матеріал на модель (моделі зібрані в один меш)
      let mesh = null;
      g.scene.updateMatrixWorld(true);
      g.scene.traverse((o) => { if (o.isMesh && !mesh) mesh = o; });
      const geo = mesh.geometry.clone().applyMatrix4(mesh.matrixWorld);
      geo.computeBoundingBox();
      const mat = mesh.material;
      if (mat.map) mat.map.anisotropy = aniso;
      A.glb[id] = { geo, mat, box: geo.boundingBox };
    }).catch((e) => console.warn('glb', id, e)));
  }
  for (const id of HDR) {
    jobs.push(hl.loadAsync(`assets/hdri/${id}.hdr`).then((t) => {
      t.mapping = THREE.EquirectangularReflectionMapping;
      A.hdr[id] = t;
    }).catch(() => {}));
  }
  await Promise.all(jobs);
  // мішки з піском замість цементних
  if (A.glb.cement_bag) {
    const m = A.glb.cement_bag.mat;
    m.map = A.tex.denim_fabric?.diff ? null : m.map;
    m.color.setHex(0x6e6448);
    m.roughness = 1;
  }
  return A;
}

// Клон текстури з власним повтором
export function texRep(id, kind, rx, ry = rx) {
  const t = A.tex[id]?.[kind];
  if (!t) return null;
  const c = t.clone();
  c.repeat.set(rx, ry);
  c.needsUpdate = true;
  return c;
}
