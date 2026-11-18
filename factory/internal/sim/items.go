package sim

// Item identifies a kind of item that can be carried on belts.
type Item uint8

const (
	ItemNone Item = iota
	IronOre
	CopperOre
	Coal
	Stone
	Sand
	IronPlate
	CopperPlate
	Brick
	Glass
	Steel
	Gear
	Wire
	ItemCount
)

// Shape is a visual hint for how an item is drawn.
type Shape uint8

const (
	ShapeOre   Shape = iota // irregular lump
	ShapePlate              // square tile
	ShapeBar                // wide rectangle
	ShapeGem                // diamond
	ShapeGear               // ring
	ShapeCoil               // small coil
)

type ItemDef struct {
	Name  string
	Color uint32 // 0xRRGGBB
	Shape Shape
}

var Items = [ItemCount]ItemDef{
	ItemNone:    {"—", 0x000000, ShapeOre},
	IronOre:     {"Залізна руда", 0x7D8E9C, ShapeOre},
	CopperOre:   {"Мідна руда", 0xC97B4A, ShapeOre},
	Coal:        {"Вугілля", 0x3B3F45, ShapeOre},
	Stone:       {"Камінь", 0xA89C8C, ShapeOre},
	Sand:        {"Кварцовий пісок", 0xE2C66E, ShapeOre},
	IronPlate:   {"Залізна пластина", 0xB4C0CB, ShapePlate},
	CopperPlate: {"Мідна пластина", 0xE39A66, ShapePlate},
	Brick:       {"Цегла", 0xB5523B, ShapeBar},
	Glass:       {"Скло", 0x8FD0DA, ShapeGem},
	Steel:       {"Сталь", 0x5E6B78, ShapeBar},
	Gear:        {"Шестерня", 0x8C99A6, ShapeGear},
	Wire:        {"Дріт", 0xD9894F, ShapeCoil},
}

// Stack is an amount of one item.
type Stack struct {
	Item Item
	N    int
}
