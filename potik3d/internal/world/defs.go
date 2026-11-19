package world

// Item is a kind of goods. Values are stored in saves: only append.
type Item uint8

const (
	ItemNone Item = iota
	Log
	Plank
	Berries
	Crop
	Water
	ItemCount
)

type ItemDef struct {
	Name  string
	Color uint32
}

var Items = [ItemCount]ItemDef{
	ItemNone: {"—", 0},
	Log:      {"Колоди", 0x9C6B45},
	Plank:    {"Дошки", 0xD9B27C},
	Berries:  {"Ягоди", 0xC0504D},
	Crop:     {"Овочі", 0xD9B44A},
	Water:    {"Вода", 0x4F9FD0},
}

// IsFood reports items settlers eat.
func IsFood(it Item) bool { return it == Berries || it == Crop }

// Stack is an amount of one item.
type Stack struct {
	Item Item
	N    int
}

// Kind is a building type. Values are stored in saves: only append.
type Kind uint8

const (
	KNone Kind = iota
	KHall
	KHouse
	KWarehouse
	KTank
	KLumber
	KForester
	KGatherer
	KFarm
	KPump
	KSawmill
	KDam
	KCampfire
	KPlatform
	KStairs
	KBridge
	KindCount
)

// Category groups buildings in the build menu.
type Category uint8

const (
	CatHome Category = iota
	CatFood
	CatWood
	CatWater
	CatBuild
	CatCount
)

var CategoryNames = [CatCount]string{"Поселення", "Їжа", "Ліс", "Вода", "Будова"}

type BuildDef struct {
	Name     string
	Desc     string
	Size     int
	Cost     []Stack
	Workers  int
	Housing  int
	Cap      int  // storage capacity
	WaterOK  bool // may be built in water
	Radius   int  // work radius for outdoor workers
	Cat      Category
	Hidden   bool // not in the build menu
	BuildSec float64
}

var Builds = [KindCount]BuildDef{
	KHall: {Name: "Ратуша", Size: 2, Housing: 6, Cap: 120, Hidden: true,
		Desc: "Серце поселення: склад на 120 речей і житло на 6 жителів."},
	KHouse: {Name: "Будинок", Size: 1, Housing: 3, Cat: CatHome, Cost: []Stack{{Log, 8}},
		Desc: "Житло на 3 жителів. Де живуть двоє і є їжа з водою — народжуються діти."},
	KWarehouse: {Name: "Склад", Size: 2, Cap: 160, Cat: CatHome, Cost: []Stack{{Log, 16}, {Plank, 4}},
		Desc: "Зберігає 160 речей (крім води). Ставте ближче до роботи — менше ходити."},
	KCampfire: {Name: "Вогнище", Size: 1, Cat: CatHome, Cost: []Stack{{Log, 6}},
		Desc: "Місце відпочинку: жителі, що живуть поруч, задоволеніші."},
	KGatherer: {Name: "Збирач ягід", Size: 1, Workers: 1, Radius: 8, Cat: CatFood, Cost: []Stack{{Log, 6}},
		Desc: "Збирає ягоди з кущів поблизу. Кущі родять лише на вологому ґрунті."},
	KFarm: {Name: "Поле", Size: 1, Workers: 1, Cat: CatFood, Cost: []Stack{{Log, 4}},
		Desc: "Вирощує овочі. Лише на вологому ґрунті біля води."},
	KLumber: {Name: "Лісоруб", Size: 1, Workers: 1, Radius: 8, Cat: CatWood, Cost: []Stack{{Log, 6}},
		Desc: "Рубає дорослі дерева поблизу: дерево — 2 колоди."},
	KForester: {Name: "Лісник", Size: 1, Workers: 1, Radius: 6, Cat: CatWood, Cost: []Stack{{Log, 6}},
		Desc: "Садить дерева на вологій землі поблизу. Саджанець росте 4 хвилини."},
	KSawmill: {Name: "Тартак", Size: 1, Workers: 1, Cat: CatWood, Cost: []Stack{{Log, 10}},
		Desc: "Розпилює колоди на дошки."},
	KPump: {Name: "Насос", Size: 1, Workers: 1, Cat: CatWater, Cost: []Stack{{Log, 6}},
		Desc: "Ставиться біля води й качає питну воду."},
	KTank: {Name: "Бак для води", Size: 1, Cap: 60, Cat: CatWater, Cost: []Stack{{Log, 6}, {Plank, 4}},
		Desc: "Зберігає 60 води — запас на посуху."},
	KDam: {Name: "Гребля", Size: 1, WaterOK: true, Cat: CatWater, Cost: []Stack{{Log, 4}},
		Desc: "Тримає воду на 1 рівень вище. Перегородіть річку, щоб пережити посуху."},
	KPlatform: {Name: "Помост", Size: 1, Cat: CatBuild, Cost: []Stack{{Log, 2}, {Plank, 3}},
		Desc: "Піднімає землю на 1 рівень; ставте ще раз зверху — до 3 рівнів. На помості можна будувати."},
	KStairs: {Name: "Сходи", Size: 1, Cat: CatBuild, Cost: []Stack{{Log, 2}, {Plank, 2}},
		Desc: "Ведуть на 1–2 рівні вгору до сусідньої клітинки. Самі розвертаються до уступу."},
	KBridge: {Name: "Міст", Size: 1, WaterOK: true, Cat: CatBuild, Cost: []Stack{{Log, 3}, {Plank, 2}},
		Desc: "Прохід над водою чи проваллям. Висота сама підлаштовується під берег; вода тече під ним."},
}

// IsSupport reports structures that carry a walkable surface (and maybe a
// building on top) instead of occupying the tile's top layer.
func (k Kind) IsSupport() bool { return k == KPlatform || k == KBridge }

// MaxStack is the highest a platform or bridge can raise a surface.
const MaxStack = 3

func (k Kind) Def() *BuildDef { return &Builds[k] }

// Accepts reports whether a storage building takes an item.
func (k Kind) Accepts(it Item) bool {
	switch k {
	case KHall:
		return true
	case KWarehouse:
		return it != Water
	case KTank:
		return it == Water
	}
	return false
}

// IsStorage reports buildings that hold the settlement's stock.
func (k Kind) IsStorage() bool { return Builds[k].Cap > 0 }

// IsField reports buildings whose workers go out onto the land.
func (k Kind) IsField() bool { return k == KLumber || k == KForester || k == KGatherer }

// BuildTime is how long a builder works on a site after materials arrive.
func (k Kind) BuildTime() float64 {
	if s := Builds[k].BuildSec; s > 0 {
		return s
	}
	n := float64(Builds[k].Size)
	return 4 * n * n
}

// Tuning.
const (
	PumpTime       = 3.0 // seconds per water
	FarmTime       = 9.0 // seconds per crop
	SawTime        = 4.0 // seconds per plank (1 log)
	ChopTime       = 5.0 // seconds to fell a tree
	PlantTime      = 3.0 // seconds to plant a sapling
	PickTime       = 3.0 // seconds to pick a bush
	TreeGrow       = 240 // seconds for a sapling to grow
	BushRegrow     = 90  // seconds for berries to grow back
	OutCap         = 6   // items a producer keeps before stopping
	WoodPerTree    = 2
	BerriesPerBush = 2
)
