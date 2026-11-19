package sim

// Kind is a building type. Values are stored in saves: only append.
type Kind uint8

const (
	KNone Kind = iota
	KBelt      // legacy (removed with conveyor logistics)
	KMiner
	KFurnace
	KAssembler
	KChest // storehouse
	KCore
	KTunnel   // legacy
	KSplitter // legacy
	KMerger   // legacy
	KSorter   // legacy
	KWall
	KFlamer
	KTurret
	KMinerBig
	KFurnaceBig
	KAssemblerBig
	KTurretBig
	KHouse
	KFarm
	KStation
	KLab
	KLumber
	KForester
	KGatherer
	KSawmill
	KPump
	KDam
	KLevee
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

// Legacy kinds only exist in old saves and are removed when loading.
func (k Kind) Legacy() bool {
	return k == KBelt || k == KTunnel || k == KSplitter || k == KMerger || k == KSorter
}

type BuildDef struct {
	Name string
	Desc string
	Size int
	Cost []Stack
	HP   float32
	// Speed multiplies work rate (miners, crafters, farms); 0 means 1.
	Speed float64
	// Workers needed inside for the building to run.
	Workers int
	// Housing capacity (homes).
	Housing int
	// Player can build it from the toolbar.
	Buildable bool
}

var Builds = [KindCount]BuildDef{
	KBelt:     {Name: "Конвеєр (старий)", Size: 1, Cost: []Stack{{IronOre, 1}}},
	KTunnel:   {Name: "Тунель (старий)", Size: 1, Cost: []Stack{{IronPlate, 6}, {Gear, 2}}},
	KSplitter: {Name: "Розгалужувач (старий)", Size: 1, Cost: []Stack{{IronPlate, 4}, {Gear, 3}, {Wire, 2}}},
	KMerger:   {Name: "Об'єднувач (старий)", Size: 1, Cost: []Stack{{IronPlate, 4}, {Gear, 3}}},
	KSorter:   {Name: "Сортер (старий)", Size: 1, Cost: []Stack{{IronPlate, 4}, {Gear, 2}, {Wire, 4}}},

	KCore: {
		Name: "Ратуша", Size: 3, HP: 2500, Housing: 4,
		Desc: "Серце поселення: склад на 300 предметів, житло на 4 особи й слабка гармата.",
	},
	KHouse: {
		Name: "Будинок", Size: 1, Buildable: true, HP: 150, Housing: 3,
		Desc: "Житло на 3 особи. Де живуть хоча б двоє й є їжа, з часом народжується новий житель.",
		Cost: []Stack{{Wood, 6}, {Stone, 2}},
	},
	KFarm: {
		Name: "Ферма", Size: 1, Buildable: true, HP: 120, Workers: 1,
		Desc: "Вирощує їжу. Ставиться на чисту землю. Без їжі жителі слабшають і помирають.",
		Cost: []Stack{{Wood, 4}},
	},
	KChest: {
		Name: "Склад", Size: 1, Buildable: true, HP: 150,
		Desc: "Тримає 200 предметів. Поставлений біля виробництва, скорочує шлях носіям.",
		Cost: []Stack{{Wood, 8}, {Stone, 4}},
	},
	KStation: {
		Name: "Вантажна станція", Size: 1, Buildable: true, HP: 200, Workers: 3,
		Desc: "Три носії, які тільки переносять вантажі в радіусі 12 клітинок: вдвічі більше й швидше за вільних жителів.",
		Cost: []Stack{{Planks, 8}, {IronOre, 8}},
	},
	KLab: {
		Name: "Дослідницька станція", Size: 1, Buildable: true, HP: 200, Workers: 1,
		Desc: "Сюди жителі зносять матеріали для досліджень, а дослідник працює над ними. Кілька станцій досліджують швидше.",
		Cost: []Stack{{Planks, 6}, {Stone, 6}},
	},
	KLumber: {
		Name: "Хатина лісоруба", Size: 1, Buildable: true, HP: 120, Workers: 1,
		Desc: "Лісоруб рубає дорослі дерева в радіусі 6 клітинок і приносить деревину.",
		Cost: []Stack{{Stone, 4}},
	},
	KForester: {
		Name: "Хатина лісника", Size: 1, Buildable: true, HP: 120, Workers: 1,
		Desc: "Лісник садить дерева на вільній землі в радіусі 6 клітинок. Саджанець виростає за 4 хв.",
		Cost: []Stack{{Wood, 4}},
	},
	KGatherer: {
		Name: "Хатина збирача", Size: 1, Buildable: true, HP: 120, Workers: 1,
		Desc: "Збирач збирає ягоди з кущів у радіусі 6 клітинок — їжа без ферми. Ягоди відростають за 1.5 хв.",
		Cost: []Stack{{Wood, 4}},
	},
	KPump: {
		Name: "Насос", Size: 1, Buildable: true, HP: 150, Workers: 1,
		Desc: "Ставиться біля води. Працівник качає воду — жителі п'ють її раз на хвилину. Запасайте воду до посухи.",
		Cost: []Stack{{Wood, 6}, {Stone, 2}},
	},
	KDam: {
		Name: "Гребля", Size: 1, Buildable: true, HP: 300,
		Desc: "Тримає воду на 1 рівень вище, надлишок переливається. Перегородіть річку, щоб пережити посуху.",
		Cost: []Stack{{Wood, 4}, {Stone, 6}},
	},
	KLevee: {
		Name: "Насип", Size: 1, Buildable: true, HP: 400,
		Desc: "Висока земляна стіна: вода крізь неї не проходить. Захищає від затоплення.",
		Cost: []Stack{{Stone, 6}, {Wood, 2}},
	},
	KSawmill: {
		Name: "Тартак", Size: 1, Buildable: true, HP: 200, Workers: 1,
		Desc: "Розпилює деревину на дошки.",
		Cost: []Stack{{Wood, 8}, {Stone, 6}},
	},
	KMiner: {
		Name: "Бур", Size: 1, Buildable: true, HP: 150, Workers: 1,
		Desc: "Ставиться на клітинку покладу. Працівник видобуває сировину, носії відносять її на склад.",
		Cost: []Stack{{Wood, 6}, {IronOre, 6}},
	},
	KFurnace: {
		Name: "Піч", Size: 1, Buildable: true, HP: 200, Workers: 1,
		Desc: "Переплавляє руду. Рецепт обирає сама за першою привезеною сировиною.",
		Cost: []Stack{{Stone, 12}},
	},
	KAssembler: {
		Name: "Збирач", Size: 1, Buildable: true, HP: 200, Workers: 1,
		Desc: "Збирає деталі. Клацни по ньому, щоб обрати рецепт.",
		Cost: []Stack{{IronPlate, 15}, {CopperPlate, 10}},
	},
	KWall: {
		Name: "Стіна", Size: 1, Buildable: true, HP: 450,
		Desc: "Міцна перешкода. Вороги мусять її прогризти, поки турелі працюють.",
		Cost: []Stack{{Stone, 4}},
	},
	KFlamer: {
		Name: "Вогнемет", Size: 1, Buildable: true, HP: 220, Workers: 1,
		Desc: "Б'є вогнем по площі на короткій дистанції. Потрібні працівник і вугілля.",
		Cost: []Stack{{IronPlate, 15}, {Brick, 10}},
	},
	KTurret: {
		Name: "Кулеметна турель", Size: 1, Buildable: true, HP: 350, Workers: 1,
		Desc: "Стріляє далеко й точно. Потрібні працівник і набої (збирач: залізо + мідь).",
		Cost: []Stack{{IronPlate, 20}, {Gear, 10}, {CopperPlate, 6}},
	},
	KMinerBig: {
		Name: "Великий бур", Size: 2, Buildable: true, HP: 400, Speed: 5, Workers: 2,
		Desc: "Покращений бур 2×2: два працівники видобувають як п'ять звичайних бурів.",
		Cost: []Stack{{IronPlate, 30}, {Steel, 10}, {Gear, 10}},
	},
	KFurnaceBig: {
		Name: "Плавильня", Size: 2, Buildable: true, HP: 500, Speed: 3, Workers: 2,
		Desc: "Покращена піч 2×2: плавить утричі швидше.",
		Cost: []Stack{{Brick, 30}, {Steel, 10}},
	},
	KAssemblerBig: {
		Name: "Складальний цех", Size: 2, Buildable: true, HP: 500, Speed: 3, Workers: 2,
		Desc: "Покращений збирач 2×2: працює втричі швидше. Клацни, щоб обрати рецепт.",
		Cost: []Stack{{Steel, 15}, {Gear, 15}, {Wire, 20}},
	},
	KTurretBig: {
		Name: "Важка турель", Size: 2, Buildable: true, HP: 800, Workers: 2,
		Desc: "Покращена турель 2×2: б'є вдвічі з половиною сильніше й на 4 клітинки далі.",
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

// BuildTime is how long a builder works on a site after materials arrive.
func BuildTime(k Kind) float64 {
	s := float64(Builds[k].Size)
	return 3 * s * s
}

// IsCrafter reports kinds that run recipes.
func (k Kind) IsCrafter() bool {
	b := k.Base()
	return b == KFurnace || b == KAssembler || b == KSawmill
}

// IsStorage reports kinds that hold the settlement's stock.
func (k Kind) IsStorage() bool { return k == KCore || k == KChest }

// Blocks reports whether enemies collide with (and attack) this kind.
func (k Kind) Blocks() bool { return k != KNone }

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
	{In: []Stack{{IronOre, 1}}, Out: Stack{IronPlate, 1}, Time: 3, Machine: KFurnace},
	{In: []Stack{{CopperOre, 1}}, Out: Stack{CopperPlate, 1}, Time: 3, Machine: KFurnace},
	{In: []Stack{{Stone, 2}}, Out: Stack{Brick, 1}, Time: 4, Machine: KFurnace},
	{In: []Stack{{Sand, 2}}, Out: Stack{Glass, 1}, Time: 4, Machine: KFurnace},
	{In: []Stack{{IronPlate, 3}, {Coal, 1}}, Out: Stack{Steel, 1}, Time: 8, Machine: KFurnace, Tech: "steel"},
	{In: []Stack{{IronPlate, 2}}, Out: Stack{Gear, 1}, Time: 2, Machine: KAssembler},
	{In: []Stack{{CopperPlate, 1}}, Out: Stack{Wire, 2}, Time: 1.6, Machine: KAssembler},
	{In: []Stack{{IronPlate, 1}, {CopperPlate, 1}}, Out: Stack{Ammo, 1}, Time: 2.4, Machine: KAssembler, Tech: "turret"},
	{In: []Stack{{Wood, 2}}, Out: Stack{Planks, 1}, Time: 3, Machine: KSawmill},
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
	TPS       = 60
	DT        = 1.0 / TPS
	ChestCap  = 200
	CoreCap   = 300
	MinerTime = 4.0 // seconds per item for a 1×1 miner on ore
	FarmTime  = 8.0 // seconds per food
	OutCap    = 8
)
