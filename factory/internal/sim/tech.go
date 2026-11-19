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
	{ID: "station", Name: "Вантажна справа", Desc: "Вантажна станція: три носії, що лише переносять вантажі.",
		Cost: []Stack{{IronOre, 30}, {Stone, 30}}, Unlocks: []Kind{KStation}},
	{ID: "chest", Name: "Стелажі", Desc: "Ратуша й склади вміщують удвічі більше: 600 і 400 предметів.",
		Cost: []Stack{{IronPlate, 40}, {Stone, 40}}, Req: []string{"furnace"}},
	{ID: "flamer", Name: "Вогнемет", Desc: "Турель ближнього бою. Палить вугілля, б'є по площі.",
		Cost: []Stack{{IronPlate, 30}, {Brick, 15}, {Coal, 20}}, Req: []string{"furnace"}, Unlocks: []Kind{KFlamer}},
	{ID: "assembler", Name: "Складання", Desc: "Збирач: шестерні й дріт — основа всього складнішого.",
		Cost: []Stack{{IronPlate, 40}, {CopperPlate, 25}}, Req: []string{"furnace"}, Unlocks: []Kind{KAssembler}},
	{ID: "turret", Name: "Кулеметна турель", Desc: "Далекобійна турель і рецепт набоїв у збирачі.",
		Cost: []Stack{{IronPlate, 60}, {Gear, 30}, {CopperPlate, 30}}, Req: []string{"assembler"}, Unlocks: []Kind{KTurret}},
	{ID: "steel", Name: "Сталеваріння", Desc: "Рецепт сталі в печі: 3 залізні пластини + вугілля.",
		Cost: []Stack{{IronPlate, 100}, {Coal, 50}}, Req: []string{"assembler"}},
	{ID: "belt1", Name: "Бруковані стежки", Desc: "Жителі й носії ходять на 25% швидше.",
		Cost: []Stack{{Stone, 120}, {IronPlate, 40}}, Req: []string{"furnace"}},
	{ID: "belt2", Name: "Візки", Desc: "Носії вантажних станцій беруть удвічі більше: 12 предметів.",
		Cost: []Stack{{Gear, 60}, {IronPlate, 60}}, Req: []string{"assembler", "station"}},
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
	{ID: "bigminer", Name: "Великий бур", Desc: "Бур 2×2: видобуває як п'ять звичайних.",
		Cost: []Stack{{Steel, 40}, {Gear, 60}}, Req: []string{"mine1", "steel"}, Unlocks: []Kind{KMinerBig}},
	{ID: "bigfurnace", Name: "Плавильня", Desc: "Піч 2×2: плавить утричі швидше.",
		Cost: []Stack{{Brick, 80}, {Steel, 30}}, Req: []string{"steel"}, Unlocks: []Kind{KFurnaceBig}},
	{ID: "bigassembler", Name: "Складальний цех", Desc: "Збирач 2×2: утричі швидший.",
		Cost: []Stack{{Steel, 40}, {Wire, 80}, {Gear, 40}}, Req: []string{"craft1", "steel"}, Unlocks: []Kind{KAssemblerBig}},
	{ID: "bigturret", Name: "Важка турель", Desc: "Турель 2×2: більша шкода й дальність.",
		Cost: []Stack{{Steel, 60}, {Gear, 40}, {Ammo, 50}}, Req: []string{"turret", "steel"}, Unlocks: []Kind{KTurretBig}},
	{ID: "repair", Name: "Ремонтна бригада", Desc: "Пошкоджені будівлі лагодяться вп'ятеро швидше.",
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

var (
	ErrResearched = errors.New("вже досліджено")
	ErrQueued     = errors.New("уже в черзі досліджень")
	ErrPrereq     = errors.New("спершу дослідь попередні вузли")
)

// TechTime is how long researchers work once all materials are delivered.
func TechTime(t *Tech) float64 {
	n := 0
	for _, c := range t.Cost {
		n += c.N
	}
	return 20 + float64(n)/3
}

// Queued reports the position of a tech in the research queue (-1 if absent).
func (w *World) Queued(id string) int {
	for i, q := range w.ResearchQueue {
		if q == id {
			return i
		}
	}
	return -1
}

// ActiveResearch is the tech being worked on now, or nil.
func (w *World) ActiveResearch() *Tech {
	if len(w.ResearchQueue) == 0 {
		return nil
	}
	return TechByID(w.ResearchQueue[0])
}

// ResearchProgress reports delivered materials (0..1) and work done (0..1)
// for the active research.
func (w *World) ResearchProgress() (delivered, work float64) {
	t := w.ActiveResearch()
	if t == nil {
		return 0, 0
	}
	total, left := 0, 0
	for _, c := range t.Cost {
		total += c.N
	}
	for _, c := range w.ResearchNeed {
		left += c.N
	}
	if total == 0 {
		return 1, float64(w.ResearchWork)
	}
	return float64(total-left) / float64(total), float64(w.ResearchWork)
}

// Research puts a tech into the research queue. Settlers carry its materials
// to a research station, where researchers then work on it.
func (w *World) Research(id string) error {
	t := TechByID(id)
	switch {
	case t == nil:
		return errors.New("невідоме дослідження")
	case w.Researched[id]:
		return ErrResearched
	case w.Queued(id) >= 0:
		return ErrQueued
	}
	for _, r := range t.Req {
		if !w.Researched[r] && w.Queued(r) < 0 {
			return ErrPrereq
		}
	}
	w.ResearchQueue = append(w.ResearchQueue, id)
	if len(w.ResearchQueue) == 1 {
		w.startResearch()
	}
	return nil
}

func (w *World) startResearch() {
	w.ResearchNeed, w.ResearchWork = nil, 0
	if t := w.ActiveResearch(); t != nil {
		w.ResearchNeed = append([]Stack(nil), t.Cost...)
	}
}

// CancelResearch removes a tech (and anything queued that depends on it)
// from the queue. Materials already delivered go back to the town hall.
func (w *World) CancelResearch(id string) {
	pos := w.Queued(id)
	if pos < 0 {
		return
	}
	if pos == 0 {
		t := w.ActiveResearch()
		for _, c := range t.Cost {
			left := 0
			for _, n := range w.ResearchNeed {
				if n.Item == c.Item {
					left = n.N
				}
			}
			w.Stock[c.Item] += c.N - left
		}
	}
	drop := map[string]bool{id: true}
	var keep []string
	for _, q := range w.ResearchQueue {
		dep := drop[q]
		for _, r := range TechByID(q).Req {
			if drop[r] {
				dep = true
			}
		}
		if dep {
			drop[q] = true
			continue
		}
		keep = append(keep, q)
	}
	w.ResearchQueue = keep
	if pos == 0 {
		w.startResearch()
	}
}

// researchLab is the station that receives research materials.
func (w *World) researchLab() *Building {
	for _, id := range w.Order() {
		if b := w.Buildings[id]; b.Kind == KLab && !b.Site && !b.Paused {
			return b
		}
	}
	return nil
}

// HasLab reports whether a finished research station exists.
func (w *World) HasLab() bool { return w.researchLab() != nil }

// stepLab advances the active research with one staffed station.
func (w *World) stepLab(b *Building) {
	t := w.ActiveResearch()
	if t == nil || len(w.ResearchNeed) > 0 {
		b.Working = false
		return
	}
	b.Working = true
	w.ResearchWork += float32(DT * w.workRate(b) / TechTime(t))
	if w.ResearchWork >= 1 {
		w.Researched[t.ID] = true
		w.ResearchQueue = w.ResearchQueue[1:]
		w.notice("Досліджено: %s", t.Name)
		w.startResearch()
	}
}

// deliverResearch hands materials to the active research; returns leftovers.
func (w *World) deliverResearch(st Stack) int {
	n := st.N
	for i := range w.ResearchNeed {
		if w.ResearchNeed[i].Item == st.Item {
			d := min(n, w.ResearchNeed[i].N)
			w.ResearchNeed[i].N -= d
			n -= d
			if w.ResearchNeed[i].N == 0 {
				w.ResearchNeed = append(w.ResearchNeed[:i], w.ResearchNeed[i+1:]...)
			}
			break
		}
	}
	return n
}

// UnlockAll researches everything (tests, demos).
func (w *World) UnlockAll() {
	for _, t := range Techs {
		w.Researched[t.ID] = true
	}
	w.ResearchQueue, w.ResearchNeed = nil, nil
}

// Multipliers from research.

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
