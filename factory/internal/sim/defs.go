package sim

// Kind is a building type.
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
	KindCount
)

type BuildDef struct {
	Name string
	Desc string
	Size int
	Cost []Stack
	// Player can build it from the toolbar.
	Buildable bool
}

var Builds = [KindCount]BuildDef{
	KBelt: {
		Name: "Конвеєр", Size: 1, Buildable: true,
		Desc: "Везе предмети. Протягни мишею, щоб прокласти лінію.",
		Cost: []Stack{{IronPlate, 1}},
	},
	KMiner: {
		Name: "Бур", Size: 2, Buildable: true,
		Desc: "Ставиться на поклад і видобуває сировину. Випускає вперед.",
		Cost: []Stack{{IronPlate, 8}, {Gear, 4}},
	},
	KFurnace: {
		Name: "Піч", Size: 2, Buildable: true,
		Desc: "Переплавляє руду. Рецепт обирає сама за першим предметом.",
		Cost: []Stack{{Stone, 10}},
	},
	KAssembler: {
		Name: "Збирач", Size: 2, Buildable: true,
		Desc: "Збирає деталі. Клацни по ньому, щоб обрати рецепт.",
		Cost: []Stack{{IronPlate, 10}, {Gear, 5}, {Wire, 6}},
	},
	KChest: {
		Name: "Склад", Size: 1, Buildable: true,
		Desc: "Накопичує до 200 предметів і віддає їх уперед.",
		Cost: []Stack{{IronPlate, 6}},
	},
	KCore: {
		Name: "Ядро", Size: 3,
		Desc: "Серце бази. Все, що сюди привезено, йде в загальний запас.",
	},
	KTunnel: {
		Name: "Тунель", Size: 1, Buildable: true,
		Desc: "Проводить стрічку під іншими будівлями. Постав вхід, потім вихід у тому ж напрямку — до 5 клітинок між ними.",
		Cost: []Stack{{IronPlate, 5}, {Gear, 1}},
	},
	KSplitter: {
		Name: "Розгалужувач", Size: 1, Buildable: true,
		Desc: "Бере предмети ззаду й по черзі роздає вліво, вперед і вправо — туди, де є куди класти. Якщо один вихід забився, все йде в інші.",
		Cost: []Stack{{IronPlate, 4}, {Gear, 2}, {Wire, 2}},
	},
	KMerger: {
		Name: "Об'єднувач", Size: 1, Buildable: true,
		Desc: "Збирає потоки зліва, ззаду й справа та чесно по черзі випускає вперед.",
		Cost: []Stack{{IronPlate, 4}, {Gear, 2}},
	},
	KSorter: {
		Name: "Сортер", Size: 1, Buildable: true,
		Desc: "Обраний предмет відправляє вбік (вліво/вправо), решту — прямо. Клацни, щоб обрати фільтр.",
		Cost: []Stack{{IronPlate, 4}, {Gear, 2}, {Wire, 4}},
	},
}

// Recipe converts inputs to one output stack in a given machine.
type Recipe struct {
	In      []Stack
	Out     Stack
	Time    float64 // seconds
	Machine Kind
}

var Recipes = []Recipe{
	{In: []Stack{{IronOre, 1}}, Out: Stack{IronPlate, 1}, Time: 1.6, Machine: KFurnace},
	{In: []Stack{{CopperOre, 1}}, Out: Stack{CopperPlate, 1}, Time: 1.6, Machine: KFurnace},
	{In: []Stack{{Stone, 2}}, Out: Stack{Brick, 1}, Time: 2.4, Machine: KFurnace},
	{In: []Stack{{Sand, 2}}, Out: Stack{Glass, 1}, Time: 2.4, Machine: KFurnace},
	{In: []Stack{{IronPlate, 3}, {Coal, 1}}, Out: Stack{Steel, 1}, Time: 6, Machine: KFurnace},
	{In: []Stack{{IronPlate, 2}}, Out: Stack{Gear, 1}, Time: 1.0, Machine: KAssembler},
	{In: []Stack{{CopperPlate, 1}}, Out: Stack{Wire, 2}, Time: 0.8, Machine: KAssembler},
}

// RecipesFor lists recipe indices usable in a machine kind.
func RecipesFor(k Kind) []int {
	var out []int
	for i, r := range Recipes {
		if r.Machine == k {
			out = append(out, i)
		}
	}
	return out
}

const (
	TPS         = 60
	DT          = 1.0 / TPS
	BeltSpeed   = 1.5  // tiles per second
	BeltSpacing = 0.25 // minimum distance between items on a belt
	ChestCap    = 200
	MinerTime   = 1.5 // seconds per item with full ore coverage
	OutCap      = 8
	TunnelMax   = 5 // tiles between entrance and exit
	RouterCap   = 2
)
