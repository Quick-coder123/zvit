# Промти для графіки «Потоку»

## Як генерувати

1. **Кожна картинка — окремий запит.** Не проси «все разом» або «спрайт-лист»: ChatGPT змішує стилі й розміри.
2. На початку чату один раз встав **«Загальний стиль»** (нижче) і попроси дотримуватись його в усіх наступних картинках. Потім надсилай промти з розділів по одному.
3. Якщо картинка вийшла в перспективі (видно бокові стінки, тінь «падає» вбік, ізометрія), відповідай: *«Redo strictly top-down orthographic, camera looking straight down, no visible sides»*.
4. Якщо фон не прозорий: *«Same image, transparent background PNG»*.
5. Розмір усім однаковий, квадрат 1024×1024. Під розмір клітинок я перемасштабую сам.
6. Назви файли так, як вказано в заголовку кожного промту (`belt.png`, `miner_base.png`…), і завантаж у `factory/assets/` на гілку `claude/amazing-galileo-2ucyyr` або прикріпи в чат.

Анімацію, повороти будівель за напрямком і підсвітку роботи робить код. Тому частини, що рухаються (свердло бура, ствол турелі, сопло вогнемета), винесені в окремі картинки.

---

## Загальний стиль (вставити першим повідомленням)

```
I am making a calm minimalist 2D factory game. Every image I ask for from now on must follow
this style exactly, so all assets match:

- Strictly TOP-DOWN ORTHOGRAPHIC view: camera looks straight down at the ground.
  No perspective, no isometric angle, no visible side walls.
- Flat vector illustration with very soft, subtle shading only. Clean geometric shapes,
  rounded corners, minimal detail. Must stay readable when shrunk to 32x32 pixels.
- Thin dark outline in slate #2E3440 around the main shapes.
- Palette: warm off-white #FBF9F4 (machine bodies), dark slate #2E3440 and #3B4252,
  teal #4FA3A5, copper #D08A5B, soft orange #F2A65A, steel blue-grey #7D8E9C,
  muted blue #6C8EBF, beige stone #B9AE9E, soft red #E07A5F.
- Square 1024x1024 canvas, TRANSPARENT background (PNG with alpha), subject centred.
- No text, no letters, no logos, no ground, no drop shadow outside the object, no frame.
- Directional objects always face RIGHT (east): their output/front side is on the right.
Reply "ok" and wait for my requests.
```

---

## 1. Ландшафт

Вже готово: `ground`, `ore_iron`, `ore_copper` (лежать у `assets/`).

### `ore_coal.png`: поклад вугілля
```
Seamless tileable top-down texture of an ore deposit on the off-white ground #F4F1EA:
scattered rounded stones of different sizes with darker outlines, about 45% of the area
covered, calm flat vector style. This time the stones are matte charcoal black #3B3F45 with tiny dark-grey
facets. Seamless tileable, top-down, fills the whole square, off-white ground #F4F1EA.
```

### `ore_stone.png`: поклад каменю
```
Same deposit texture style, but large flat beige-grey rocks #A89C8C with soft cracks.
Seamless tileable, top-down, fills the whole square, off-white ground #F4F1EA.
```

### `ore_sand.png`: кварцовий пісок
```
Same deposit texture style, but a soft golden sand patch #E2C66E with tiny bright quartz
specks #FFF6D8 and small ripples. Seamless tileable, top-down, fills the whole square.
```

---

## 2. Логістика (1 клітинка)

### `belt.png`: конвеєр
```
One straight conveyor belt segment seen from directly above, running left-to-right,
occupying the full width of the square and about 80% of its height. Light warm grey
rubber belt #D8D1C4 with thin darker side rails #C2B9A9 at top and bottom edges.
NO arrows or chevrons on it (they are animated by the game). Transparent background.
```

### `belt_curve.png`: поворот конвеєра
```
A conveyor belt that turns 90 degrees, seen from directly above: it enters from the LEFT
edge and exits through the BOTTOM edge, following a smooth quarter-circle. Same width,
colours and rails as a straight belt (#D8D1C4 belt, #C2B9A9 rails). No arrows.
Transparent background.
```

### `tunnel_in.png`: вхід тунелю
```
Entrance of an underground conveyor tunnel, top-down. The left half is a short piece of
light grey conveyor belt #D8D1C4; the right half is a solid rounded slate block #5E6B78
with a dark opening #2E3440 facing left where the belt disappears inside. Small light
arrow on the block pointing right. Transparent background.
```

### `tunnel_out.png`: вихід тунелю
```
Exit of an underground conveyor tunnel, mirror of the entrance, top-down: the LEFT half is
the rounded slate block #5E6B78 with its dark opening facing right, the right half is a short
conveyor belt piece #D8D1C4 leaving it. Small light arrow pointing right. Transparent background.
```

### `splitter.png`: розгалужувач
```
Small square logistics machine, top-down, off-white body #FBF9F4 with lilac outline #8E7CC3,
sitting on a light grey belt pad. On top, three arrows from the centre: one up, one right,
one down, the side arrows lilac #8E7CC3, the right arrow dark. Clean and iconic.
Transparent background.
```

### `merger.png`: об'єднувач
```
Small square logistics machine, top-down, off-white body #FBF9F4 with soft green outline
#6FAE7B, on a light grey belt pad. Three green arrows pointing INTO the centre (from top,
left and bottom) and one dark arrow pointing out to the right. Transparent background.
```

### `sorter.png`: сортер
```
Small square logistics machine, top-down, off-white body #FBF9F4 with warm tan outline
#D4A373, on a light grey belt pad. Empty round slot in the centre (the game draws the
filtered item there), small tan arrows pointing up and down, a dark arrow pointing right.
Transparent background.
```

---

## 3. Виробництво

### `miner_base.png`: бур, корпус (2×2)
```
Mining drill machine, top-down, square footprint. Off-white rounded body #FBF9F4 with teal
outline #4FA3A5, four small bolts in the corners, an empty circular well in the centre
(the spinning drill head is a separate image), a small output chute on the right edge.
Transparent background.
```

### `miner_head.png`: бур, свердло (обертається)
```
Top-down drill head on its own: a circular teal #4FA3A5 rotor with three curved blades
and a dark slate hub #2E3440 in the centre. Perfectly centred and rotationally balanced
so it can spin. Transparent background.
```

### `furnace_off.png` і `furnace_on.png`: піч (2×2)
```
Smelting furnace, top-down, square footprint. Off-white rounded body #FBF9F4 with soft
orange outline #F2A65A, a short chimney cap at the top edge, a rounded dark firebox
opening #4C566A in the middle, output chute on the right edge. Transparent background.
```
Друга картинка:
```
Same furnace, identical framing and shapes, but working: the firebox glows warm
orange-yellow #F2A65A → #F7D774 with a soft glow. Transparent background.
```

### `assembler.png`: збирач (2×2)
```
Assembling machine, top-down, square footprint. Off-white rounded body #FBF9F4 with muted
blue outline #6C8EBF, a ring of eight small blue rollers around an empty circular work area
in the centre (the game shows the product there), output chute on the right edge.
Transparent background.
```

### `chest.png`: склад (1 клітинка)
```
Storage crate, top-down, square. Light wooden lid #E5D3B3 with a darker band #C8A97E across
the upper third and two small metal latches, soft rounded corners, a small output notch
on the right edge. Transparent background.
```

### `core.png`: ядро (3×3)
```
The heart of the base, top-down, large square footprint. Dark slate rounded block #3B4252
with a thin teal ring #4FA3A5 inscribed in it and a glowing teal diamond crystal in the
centre. Subtle panel seams, calm and important-looking. Leave a clear circular space
around the crystal for a small turret barrel drawn by the game. Transparent background.
```

---

## 4. Оборона

### `wall.png`: стіна (1 клітинка)
```
Stone wall block, top-down, filling the whole square: neat beige stone bricks #B9AE9E in
a running bond pattern with darker mortar lines #8C8173, slightly rounded outer corners.
Seamless enough that walls placed side by side look continuous. Transparent background.
```

### `flamer_base.png` і `flamer_nozzle.png`: вогнемет
```
Flamethrower turret base, top-down, square 1-tile footprint: off-white body #FBF9F4 with soft
orange outline #F2A65A, a round dark slate swivel plate #4C566A in the centre, small fuel
tank on one side. No nozzle (separate image). Transparent background.
```
```
Flamethrower nozzle on its own, top-down: a short thick dark slate barrel #3B4252 pointing
RIGHT from the exact centre of the canvas, ending with an orange tip #F2A65A. The pivot
point is the canvas centre (barrel starts there). Transparent background.
```

### `turret_base.png` і `turret_barrel.png`: кулеметна турель (2×2)
```
Machine-gun turret base, top-down, square footprint: off-white body #FBF9F4 with slate
outline #5E6B78, four corner bolts, a large round dark slate turntable #4C566A in the centre.
No gun barrel (separate image). Transparent background.
```
```
Machine-gun turret head on its own, top-down: round dark slate cap #3B4252 at the exact centre
of the canvas with a twin barrel pointing RIGHT, reaching about 45% of the canvas width,
small brass detail #C9A227. Pivot = canvas centre. Transparent background.
```

---

## 5. Вороги (вигляд згори, обличчям вправо)

### `enemy_crawler.png`: повзун
```
Small cute-but-hostile bug creature seen from directly above, facing RIGHT: round body in
soft red #D9694F with a darker outline, six short legs (three per side), two big white eyes
with dark pupils near the right edge looking right. Simple, friendly-minimal style, not gory.
Transparent background.
```

### `enemy_swarmer.png`: мошка
```
Tiny fast insect seen from directly above, facing RIGHT: small round orange body #E8A04F,
two translucent wings, two dot eyes near the right edge. Simple minimal style.
Transparent background.
```

### `enemy_brute.png`: громила
```
Big armoured beetle-like creature seen from directly above, facing RIGHT: heavy round body
in dark plum #8C4A5E with a lighter armour shell ring and plates, six thick legs, small
angry eyes near the right edge. Minimal style, not gory. Transparent background.
```

---

## 6. Предмети (іконки, що їздять конвеєром)

Для всіх: *«Single item icon, top-down, centred, fills 70% of the canvas, transparent background, same style.»* Додай до кожного:

| Файл | Опис |
|---|---|
| `item_iron_ore.png` | `An irregular chunk of blue-grey iron ore #7D8E9C with facets` |
| `item_copper_ore.png` | `An irregular chunk of copper ore #C97B4A with a few green patina spots` |
| `item_coal.png` | `A lump of matte black coal #3B3F45 with subtle facets` |
| `item_stone.png` | `A rough beige-grey stone #A89C8C` |
| `item_sand.png` | `A small heap of golden quartz sand #E2C66E with bright specks` |
| `item_iron_plate.png` | `A square light steel plate #B4C0CB with rounded corners and a subtle bevel` |
| `item_copper_plate.png` | `A square copper plate #E39A66 with rounded corners and a subtle bevel` |
| `item_brick.png` | `A single red clay brick #B5523B seen from above` |
| `item_glass.png` | `A small pane of light cyan glass #8FD0DA, diamond-shaped, with a highlight` |
| `item_steel.png` | `A dark steel ingot #5E6B78 seen from above` |
| `item_gear.png` | `A grey metal gear #8C99A6 with 8 teeth and a hole in the centre` |
| `item_wire.png` | `A small coil of copper wire #D9894F` |
| `item_ammo.png` | `Two brass cartridges #C9A227 side by side, bullets pointing up` |

---

## 7. Ефекти

### `fx_muzzle.png`: спалах пострілу
```
Small muzzle flash seen from above, pointing RIGHT from the centre: a bright pale-yellow
#F7D774 star-shaped burst with an orange #F2A65A edge. Transparent background.
```

### `fx_flame.png`: язик полум'я
```
A soft blob of flame seen from above, roundish, orange #F2A65A outside, yellow #F7D774
inside, semi-transparent edges. Transparent background.
```

### `fx_splat.png`: слід від знищеного ворога
```
Small flat splat mark on the ground seen from above, muted dark red-brown #8C4A5E at 60%
opacity, irregular round shape with a few droplets. Not gory, minimal. Transparent background.
```

### `fx_ruin.png`: руїни будівлі
```
Top-down view of rubble left after a small building was destroyed: a few broken off-white
panels #FBF9F4 and grey debris #9AA0AC scattered within a square area, soft red #E07A5F
accent on some pieces. Transparent background.
```

---

## 8. Інтерфейс і заставки

### `icon.png`: іконка програми
```
App icon for a calm minimalist factory-building game called "Potik" (means "Flow").
Flat vector style, no text. A single stylized conveyor belt curving into a smooth S-shape
flow, carrying three small geometric items (a square, a circle, a triangle). Off-white
background #F4F1EA (NOT transparent), dark slate #2E3440, teal #4FA3A5, copper #D08A5B,
soft orange #F2A65A. Thick clean shapes, rounded corners, readable at 32x32.
```

### `menu.png`: фон головного меню (1920×1080)
```
Wide minimalist landscape illustration for the main menu, flat vector art, 2D side view
(this one is NOT top-down). A quiet plain at dawn with a small tidy factory: simple block
buildings, thin conveyor lines, two small turrets and a low stone wall guarding it, a few
tiny bug-like silhouettes far on the horizon. Lots of empty sky on the left third for menu
buttons. No text. Palette: off-white #F4F1EA, pale sky #DCE6E8, slate #2E3440, teal #4FA3A5,
copper #D08A5B, orange #F2A65A. Calm, slightly tense, hopeful. 1920x1080, not transparent.
```

### `wave.png`: заставка «Хвиля наближається» (необов'язково)
```
Wide minimalist banner illustration, flat vector, 2D side view: a dark line of small
bug-like creatures appearing over a hill at dusk, a tiny factory with turrets in the
foreground holding its ground. Soft red #E07A5F accents in the sky, slate #2E3440 silhouettes.
No text. 1920x600, not transparent.
```

### `research.png`: іконка досліджень
```
Small UI icon: a laboratory flask with teal liquid #4FA3A5, slate outline #2E3440, flat
vector, centred, transparent background.
```

---

## Перелік файлів

**Ландшафт:** `ore_coal`, `ore_stone`, `ore_sand`

**Логістика:** `belt`, `belt_curve`, `tunnel_in`, `tunnel_out`, `splitter`, `merger`, `sorter`

**Виробництво:** `miner_base`, `miner_head`, `furnace_off`, `furnace_on`, `assembler`, `chest`, `core`

**Оборона:** `wall`, `flamer_base`, `flamer_nozzle`, `turret_base`, `turret_barrel`

**Вороги:** `enemy_crawler`, `enemy_swarmer`, `enemy_brute`

**Предмети:** `item_iron_ore`, `item_copper_ore`, `item_coal`, `item_stone`, `item_sand`, `item_iron_plate`, `item_copper_plate`, `item_brick`, `item_glass`, `item_steel`, `item_gear`, `item_wire`, `item_ammo`

**Ефекти:** `fx_muzzle`, `fx_flame`, `fx_splat`, `fx_ruin`

**Інтерфейс:** `icon`, `menu`, `wave` (необов'язково), `research`

Лишилось 44 картинки. Не обов'язково робити всі одразу: гра підхоплюватиме ті, що є, а для решти лишатиметься нинішня графіка з коду.
