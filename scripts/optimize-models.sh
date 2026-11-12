#!/usr/bin/env bash
# Спрощує моделі Poly Haven до "легкої" геометрії та пакує в .glb (client/assets/glb/).
set -e
cd "$(dirname "$0")/.."
SRC=client/assets/models; OUT=client/assets/glb; mkdir -p $OUT
opt() { # id ratio texsize
  if [ "$2" = "1" ]; then SIMP="--simplify false"; else SIMP="--simplify-ratio $2 --simplify-error ${4:-0.004}"; fi
  # рослини (ratio 1) не спрощуємо: листові пластинки схлопуються; їх проріджує клієнт
  npx gltf-transform optimize "$SRC/$1/$1.gltf" "$OUT/$1.glb" --compress false --texture-compress false \
    $SIMP --instance false --join true --palette false >/dev/null
  npx gltf-transform resize "$OUT/$1.glb" "$OUT/$1.glb" --width "${3:-512}" --height "${3:-512}" >/dev/null
  echo "$1 -> $(du -h "$OUT/$1.glb" | cut -f1)"
}
opt concrete_road_barrier 0.03 512
opt concrete_road_barrier_02 0.05 512 0.02
opt covered_car 0.05 512 0.04
opt old_tyre 0.15 256 0.02
opt street_lamp_01 0.06 512 0.02
opt old_military_crate 0.006 256 0.06
opt wooden_military_crate 0.02 256 0.05
opt ammo_box 0.3 256
opt metal_jerrycan_green 0.1 256 0.02
opt barrel_03 0.6 512
opt portable_generator 0.05 512 0.02
opt cement_bag 0.25 256 0.03
opt dead_tree_trunk_02 0.004 256 0.06
opt tree_stump_01 0.012 512 0.03
opt rock_07 0.02 512 0.03
opt stone_01 0.004 512 0.03
