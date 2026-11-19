package sim

// Kind is a building type. Values are stored in saves: only append.
type Kind uint8

const (
	KNone Kind = iota
	KBelt
	KMiner
	KFurnace
	KAssembler
	KChest
	KCore
	KTunnel
	KSplitter
	KMerger
	KSorter
	KWall
	KFlamer
	KTurret
	KMinerBig
	KFurnaceBig
	KAssemblerBig
	KTurretBig
	KindCount
)

// bigOf maps a base machine to its larger upgraded variant.
var bigOf = map[Kind]Kind{KMiner: KMinerBig, KFurnace: KFurnaceBig, KAssembler: KAssemblerBig, KTurret: KTurretBig}

// Base returns the basic kind of an upgraded variant (or k itself).
func (k Kind) Base() Kind {
	for b, big := range bigOf {
		if big == k {
			return b
		}
	}
	return k
}

// Big returns the upgraded variant of a base kind, or KNone.
func (k Kind) Big() Kind { return bigOf[k] }

type BuildDef struct {
	Name string
	Desc string
	Size int
	Cost []Stack
	HP   float32
	// Speed multiplies work rate (miners, crafters); 0 means 1.
	Speed float64
	// Player can build it from the toolbar.
	Buildable bool
}

var Builds = [KindCount]BuildDef{
	KBelt: {
		Name: "Конвеєр", Size: 1, Buildable: true, HP: 30,
		Desc: "Везе предмети. Протягни мишею, щоб прокласти лінію. Вороги по ньому просто ходять.",
		Cost: []Stack{{IronOre, 1}},
	},
	KMiner: {
		Name: "Бур", Size: 1, Buildable: true, HP: 150,
		Desc: "Ставиться на клітинку покладу й видобуває сировину. Випускає вперед.",
		Cost: []Stack{{IronOre, 15}},
	},
	KFurnace: {
		Name: "Піч", Size: 1, Buildable: true, HP: 200,
		Desc: "Переплавляє руду. Рецепт обирає сама за першим предметом.",
		Cost: []Stack{{Stone, 12}},
	},
	KAssembler: {
		Name: "Збирач", Size: 1, Buildable: true, HP: 200,
		Desc: "Збирає деталі. Клацни по ньому, щоб обрати рецепт.",
		Cost: []Stack{{IronPlate, 15}, {CopperPlate, 10}},
	},
	KChest: {
		Name: "Склад", Size: 1, Buildable: true, HP: 150,
		Desc: "Накопичує до 200 предметів і віддає їх уперед.",
		Cost: []Stack{{IronPlate, 8}},
	},
	KCore: {
		Name: "Ядро", Size: 3, HP: 2500,
		Desc: "Серце бази. Все, що сюди привезено, йде в загальний запас. Має власну слабку гармату.",
	},
	KTunnel: {
		Name: "Тунель", Size: 1, Buildable: true, HP: 80,
		Desc: "Проводить стрічку під іншими будівлями. Постав вхід, потім вихід у тому ж напрямку — до 5 клітинок між ними.",
		Cost: []Stack{{IronPlate, 6}, {Gear, 2}},
	},
	KSplitter: {
		Name: "Розгалужувач", Size: 1, Buildable: true, HP: 80,
		Desc: "Бере предмети ззаду й по черзі роздає вліво, вперед і вправо — туди, де є куди класти. Якщо один вихід забився, все йде в інші.",
		Cost: []Stack{{IronPlate, 4}, {Gear, 3}, {Wire, 2}},
	},
	KMerger: {
		Name: "Об'єднувач", Size: 1, Buildable: true, HP: 80,
		Desc: "Збирає потоки зліва, ззаду й справа та чесно по черзі випускає вперед.",
		Cost: []Stack{{IronPlate, 4}, {Gear, 3}},
	},
	KSorter: {
		Name: "Сортер", Size: 1, Buildable: true, HP: 80,
		Desc: "Обраний предмет відправляє вбік (вліво/вправо), решту — прямо. Клацни, щоб обрати фільтр.",
		Cost: []Stack{{IronPlate, 4}, {Gear, 2}, {Wire, 4}},
	},
	KWall: {
		Name: "Стіна", Size: 1, Buildable: true, HP: 450,
		Desc: "Міцна перешкода. Вороги мусять її прогризти, поки турелі працюють.",
		Cost: []Stack{{Stone, 4}},
	},
	KFlamer: {
		Name: "Вогнемет", Size: 1, Buildable: true, HP: 220,
		Desc: "Б'є вогнем по площі на короткій дистанції. Палить вугілля — підвези його конвеєром.",
		Cost: []Stack{{IronPlate, 15}, {Brick, 10}},
	},
	KTurret: {
		Name: "Кулеметна турель", Size: 1, Buildable: true, HP: 350,
		Desc: "Стріляє далеко й точно. Потрібні набої (збирач: залізо + мідь) — підвези конвеєром.",
		Cost: []Stack{{IronPlate, 20}, {Gear, 10}, {CopperPlate, 6}},
	},
	KMinerBig: {
		Name: "Великий бур", Size: 2, Buildable: true, HP: 400, Speed: 5,
		Desc: "Покращений бур 2×2: на повному покладі видобуває як п'ять звичайних, а вихід один.",
		Cost: []Stack{{IronPlate, 30}, {Steel, 10}, {Gear, 10}},
	},
	KFurnaceBig: {
		Name: "Плавильня", Size: 2, Buildable: true, HP: 500, Speed: 3,
		Desc: "Покращена піч 2×2: плавить утричі швидше.",
		Cost: []Stack{{Brick, 30}, {Steel, 10}},
	},
	KAssemblerBig: {
		Name: "Складальний цех", Size: 2, Buildable: true, HP: 500, Speed: 3,
		Desc: "Покращений збирач 2×2: працює втричі швидше. Клацни, щоб обрати рецепт.",
		Cost: []Stack{{Steel, 15}, {Gear, 15}, {Wire, 20}},
	},
	KTurretBig: {
		Name: "Важка турель", Size: 2, Buildable: true, HP: 800,
		Desc: "Покращена турель 2×2: б'є вдвічі з половиною сильніше й на 4 клітинки далі. Потрібні набої.",
		Cost: []Stack{{Steel, 25}, {Gear, 20}, {CopperPlate, 20}},
	},
}

// SpeedOf returns the work-rate multiplier of a kind.
func SpeedOf(k Kind) float64 {
	if s := Builds[k].Speed; s > 0 {
		return s
	}
	return 1
}

// IsCrafter reports kinds that run recipes.
func (k Kind) IsCrafter() bool { b := k.Base(); return b == KFurnace || b == KAssembler }

// Blocks reports whether enemies collide with (and attack) this kind.
func (k Kind) Blocks() bool { return k != KBelt && k != KNone }

// Recipe converts inputs to one output stack in a given machine.
type Recipe struct {
	In      []Stack
	Out     Stack
	Time    float64 // seconds
	Machine Kind
	Tech    string // research needed, "" if always available
}

// Recipe indices are stored in saves: only append.
var Recipes = []Recipe{
	{In: []Stack{{IronOre, 1}}, Out: Stack{IronPlate, 1}, Time: 1.6, Machine: KFurnace},
	{In: []Stack{{CopperOre, 1}}, Out: Stack{CopperPlate, 1}, Time: 1.6, Machine: KFurnace},
	{In: []Stack{{Stone, 2}}, Out: Stack{Brick, 1}, Time: 2.4, Machine: KFurnace},
	{In: []Stack{{Sand, 2}}, Out: Stack{Glass, 1}, Time: 2.4, Machine: KFurnace},
	{In: []Stack{{IronPlate, 3}, {Coal, 1}}, Out: Stack{Steel, 1}, Time: 6, Machine: KFurnace, Tech: "steel"},
	{In: []Stack{{IronPlate, 2}}, Out: Stack{Gear, 1}, Time: 1.0, Machine: KAssembler},
	{In: []Stack{{CopperPlate, 1}}, Out: Stack{Wire, 2}, Time: 0.8, Machine: KAssembler},
	{In: []Stack{{IronPlate, 1}, {CopperPlate, 1}}, Out: Stack{Ammo, 1}, Time: 1.2, Machine: KAssembler, Tech: "turret"},
}

// RecipesFor lists recipe indices usable in a machine kind (locked included).
func RecipesFor(k Kind) []int {
	var out []int
	for i, r := range Recipes {
		if r.Machine == k.Base() {
			out = append(out, i)
		}
	}
	return out
}

const (
	TPS         = 60
	DT          = 1.0 / TPS
	BeltSpeed   = 1.5  // base tiles per second
	BeltSpacing = 0.25 // minimum distance between items on a belt
	ChestCap    = 200
	MinerTime   = 1.5 // seconds per item with full ore coverage
	OutCap      = 8
	TunnelMax   = 5 // tiles between entrance and exit
	RouterCap   = 2
)
