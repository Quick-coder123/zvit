package sim

import "errors"

// Tech is a research node bought with resources from the core.
type Tech struct {
	ID      string
	Name    string
	Desc    string
	Cost    []Stack
	Req     []string
	Unlocks []Kind
}

// Techs is the research tree. IDs are stored in saves.
var Techs = []Tech{
	{ID: "furnace", Name: "Плавка", Desc: "Піч: руда → пластини, камінь → цегла, пісок → скло.",
		Cost: []Stack{{IronOre, 30}, {Stone, 20}}, Unlocks: []Kind{KFurnace}},
	{ID: "walls", Name: "Укріплення", Desc: "Кам'яні стіни. Вороги мусять їх прогризати.",
		Cost: []Stack{{Stone, 30}}, Unlocks: []Kind{KWall}},
	{ID: "chest", Name: "Склади", Desc: "Склад на 200 предметів — буфер для ліній і турелей.",
		Cost: []Stack{{IronPlate, 20}}, Req: []string{"furnace"}, Unlocks: []Kind{KChest}},
	{ID: "flamer", Name: "Вогнемет", Desc: "Турель ближнього бою. Палить вугілля, б'є по площі.",
		Cost: []Stack{{IronPlate, 30}, {Brick, 15}, {Coal, 20}}, Req: []string{"furnace"}, Unlocks: []Kind{KFlamer}},
	{ID: "assembler", Name: "Складання", Desc: "Збирач: шестерні й дріт — основа всього складнішого.",
		Cost: []Stack{{IronPlate, 40}, {CopperPlate, 25}}, Req: []string{"furnace"}, Unlocks: []Kind{KAssembler}},
	{ID: "turret", Name: "Кулеметна турель", Desc: "Далекобійна турель і рецепт набоїв у збирачі.",
		Cost: []Stack{{IronPlate, 60}, {Gear, 30}, {CopperPlate, 30}}, Req: []string{"assembler"}, Unlocks: []Kind{KTurret}},
	{ID: "tunnel", Name: "Тунелі", Desc: "Стрічки під іншими будівлями, до 5 клітинок.",
		Cost: []Stack{{IronPlate, 40}, {Gear, 15}}, Req: []string{"assembler"}, Unlocks: []Kind{KTunnel}},
	{ID: "split", Name: "Розподіл потоків", Desc: "Розгалужувач і об'єднувач.",
		Cost: []Stack{{IronPlate, 50}, {Gear, 25}, {Wire, 20}}, Req: []string{"assembler"}, Unlocks: []Kind{KSplitter, KMerger}},
	{ID: "sorter", Name: "Сортування", Desc: "Сортер: обраний предмет іде вбік.",
		Cost: []Stack{{Gear, 30}, {Wire, 40}}, Req: []string{"split"}, Unlocks: []Kind{KSorter}},
	{ID: "steel", Name: "Сталеваріння", Desc: "Рецепт сталі в печі: 3 залізні пластини + вугілля.",
		Cost: []Stack{{IronPlate, 100}, {Coal, 50}}, Req: []string{"assembler"}},
	{ID: "belt1", Name: "Змащені стрічки", Desc: "Конвеєри й тунелі на 33% швидші.",
		Cost: []Stack{{IronPlate, 60}, {Gear, 60}}, Req: []string{"assembler"}},
	{ID: "belt2", Name: "Посилені приводи", Desc: "Конвеєри ще на 50% швидші.",
		Cost: []Stack{{Steel, 40}, {Gear, 80}}, Req: []string{"belt1", "steel"}},
	{ID: "mine1", Name: "Загартовані бури", Desc: "Бури видобувають на 30% більше.",
		Cost: []Stack{{IronPlate, 80}, {Gear, 40}}, Req: []string{"assembler"}},
	{ID: "mine2", Name: "Сталеві свердла", Desc: "Бури видобувають ще на 40% більше.",
		Cost: []Stack{{Steel, 50}, {Gear, 60}}, Req: []string{"mine1", "steel"}},
	{ID: "craft1", Name: "Робочий ритм", Desc: "Печі й збирачі працюють на 30% швидше.",
		Cost: []Stack{{IronPlate, 100}, {Wire, 60}, {Brick, 40}}, Req: []string{"assembler"}},
	{ID: "craft2", Name: "Точна механіка", Desc: "Печі й збирачі ще на 40% швидші.",
		Cost: []Stack{{Steel, 60}, {Gear, 80}, {Wire, 80}}, Req: []string{"craft1", "steel"}},
	{ID: "dmg1", Name: "Посилені набої", Desc: "Турелі й вогнемети б'ють на 30% сильніше.",
		Cost: []Stack{{Steel, 30}, {CopperPlate, 80}}, Req: []string{"turret", "steel"}},
	{ID: "dmg2", Name: "Бронебійні набої", Desc: "Ще +50% шкоди, кулі ігнорують броню.",
		Cost: []Stack{{Steel, 80}, {Gear, 60}}, Req: []string{"dmg1"}},
	{ID: "range", Name: "Оптика", Desc: "Турелі бачать і стріляють на 3 клітинки далі.",
		Cost: []Stack{{Glass, 40}, {Gear, 30}}, Req: []string{"turret"}},
	{ID: "armor", Name: "Бронювання", Desc: "Усі будівлі на 60% міцніші.",
		Cost: []Stack{{Steel, 40}, {Brick, 60}}, Req: []string{"steel"}},
	{ID: "repair", Name: "Самовідновлення", Desc: "Пошкоджені будівлі лагодяться вп'ятеро швидше.",
		Cost: []Stack{{Gear, 60}, {Wire, 60}, {Glass, 20}}, Req: []string{"assembler"}},
}

var (
	techIndex = map[string]int{}
	kindTech  [KindCount]string
)

func init() {
	for i, t := range Techs {
		techIndex[t.ID] = i
		for _, k := range t.Unlocks {
			kindTech[k] = t.ID
		}
	}
}

// TechFor names the research that unlocks a building ("" if none needed).
func TechFor(k Kind) string { return kindTech[k] }

// TechByID returns a research node.
func TechByID(id string) *Tech {
	if i, ok := techIndex[id]; ok {
		return &Techs[i]
	}
	return nil
}

// Has reports whether a research is done.
func (w *World) Has(id string) bool { return id == "" || w.Researched[id] }

// Unlocked reports whether a building may be built.
func (w *World) Unlocked(k Kind) bool { return w.Has(kindTech[k]) }

// RecipeUnlocked reports whether a recipe may be used.
func (w *World) RecipeUnlocked(r int) bool { return w.Has(Recipes[r].Tech) }

// TechAvailable reports whether all prerequisites are researched.
func (w *World) TechAvailable(t *Tech) bool {
	for _, r := range t.Req {
		if !w.Researched[r] {
			return false
		}
	}
	return true
}

func (w *World) affordable(cost []Stack) bool {
	for _, c := range cost {
		if w.Stock[c.Item] < c.N {
			return false
		}
	}
	return true
}

var (
	ErrResearched = errors.New("вже досліджено")
	ErrPrereq     = errors.New("спершу дослідь попередні вузли")
)

// Research buys a tech with resources from the core.
func (w *World) Research(id string) error {
	t := TechByID(id)
	switch {
	case t == nil:
		return errors.New("невідоме дослідження")
	case w.Researched[id]:
		return ErrResearched
	case !w.TechAvailable(t):
		return ErrPrereq
	case !w.affordable(t.Cost):
		return ErrNoFunds
	}
	for _, c := range t.Cost {
		w.Stock[c.Item] -= c.N
	}
	w.Researched[id] = true
	return nil
}

// UnlockAll researches everything (tests, demos).
func (w *World) UnlockAll() {
	for _, t := range Techs {
		w.Researched[t.ID] = true
	}
}

// Multipliers from research.

func (w *World) BeltSpeed() float64 {
	s := BeltSpeed
	if w.Researched["belt1"] {
		s *= 4.0 / 3.0
	}
	if w.Researched["belt2"] {
		s *= 1.5
	}
	return s
}

func (w *World) MineMul() float64 {
	m := 1.0
	if w.Researched["mine1"] {
		m *= 1.3
	}
	if w.Researched["mine2"] {
		m *= 1.4
	}
	return m
}

func (w *World) CraftMul() float64 {
	m := 1.0
	if w.Researched["craft1"] {
		m *= 1.3
	}
	if w.Researched["craft2"] {
		m *= 1.4
	}
	return m
}

func (w *World) DamageMul() float32 {
	m := float32(1)
	if w.Researched["dmg1"] {
		m *= 1.3
	}
	if w.Researched["dmg2"] {
		m *= 1.5
	}
	return m
}

func (w *World) RangeBonus() float64 {
	if w.Researched["range"] {
		return 3
	}
	return 0
}

// MaxHP of a building including research bonuses.
func (w *World) MaxHP(b *Building) float32 {
	hp := Builds[b.Kind].HP
	if w.Researched["armor"] {
		hp *= 1.6
	}
	return hp
}
