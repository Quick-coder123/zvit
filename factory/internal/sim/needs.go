package sim

import "math"

// Foods settlers can eat. Eating different kinds in a row satisfies the
// need for a varied diet.
var Foods = []Item{Food, Berries, Bread}

// IsFood reports items settlers eat.
func IsFood(it Item) bool {
	for _, f := range Foods {
		if f == it {
			return true
		}
	}
	return false
}

// Need is one thing that makes settlers content.
type Need int

const (
	NeedFood Need = iota
	NeedWater
	NeedVariety
	NeedLeisure
	NeedBeauty
	NeedCount
)

var NeedNames = [NeedCount]string{"Їжа", "Вода", "Різноманітна їжа", "Відпочинок", "Краса"}

const (
	LeisureRadius = 7.0 // campfire reach around homes
	BeautyRadius  = 5.0 // flowerbeds; statues reach further
	StatueRadius  = 8.0
)

// needScores reports how well each need of a settler is met (0..1).
func (w *World) needScores(s *Settler) [NeedCount]float64 {
	var n [NeedCount]float64
	if s.Starving == 0 {
		n[NeedFood] = 1
	}
	if s.Thirst == 0 {
		n[NeedWater] = 1
	}
	if s.LastFood != ItemNone && s.PrevFood != ItemNone && s.LastFood != s.PrevFood {
		n[NeedVariety] = 1
	}
	home := w.Buildings[s.Home]
	if home == nil {
		return n
	}
	hx, hy := home.center()
	beauty := 0.0
	for _, b := range w.Buildings {
		if b.Site {
			continue
		}
		bx, by := b.center()
		d := math.Hypot(bx-hx, by-hy)
		switch b.Kind {
		case KCampfire:
			if d <= LeisureRadius {
				n[NeedLeisure] = 1
			}
		case KFlowerbed:
			if d <= BeautyRadius {
				beauty += 0.34
			}
		case KStatue:
			if d <= StatueRadius {
				beauty += 1
			}
		}
	}
	n[NeedBeauty] = math.Min(1, beauty)
	return n
}

// updateWellbeing recomputes the town's wellbeing and its productivity bonus.
func (w *World) updateWellbeing() {
	if len(w.Settlers) == 0 {
		w.wellbeing, w.needShare = 0, [NeedCount]float64{}
		return
	}
	var sum [NeedCount]float64
	for _, s := range w.Settlers {
		n := w.needScores(s)
		for i := range n {
			sum[i] += n[i]
		}
	}
	total := 0.0
	for i := range sum {
		sum[i] /= float64(len(w.Settlers))
		total += sum[i]
	}
	w.wellbeing, w.needShare = total, sum
}

// Wellbeing is the average number of needs met (0..5).
func (w *World) Wellbeing() float64 { return w.wellbeing }

// NeedShare is the share of settlers whose need is met (0..1).
func (w *World) NeedShare(n Need) float64 { return w.needShare[n] }

// Productivity scales all work by wellbeing: x0.8 when only survival needs
// are met badly, x1.3 when every need is met.
func (w *World) Productivity() float64 { return 0.8 + 0.1*w.wellbeing }

// FoodAvailable counts all foods in storage and waiting at producers.
func (w *World) FoodAvailable() int {
	n := 0
	for _, f := range Foods {
		n += w.TotalStock(f)
	}
	for _, b := range w.Buildings {
		if IsFood(b.OutItem) {
			n += b.OutN
		}
	}
	return n
}

// eat takes one food, preferring a kind the settler didn't just eat, from
// storage or straight from a producer so full warehouses never starve
// the town.
func (w *World) eat(s *Settler) bool {
	order := make([]Item, 0, len(Foods))
	for _, f := range Foods {
		if f != s.LastFood {
			order = append(order, f)
		}
	}
	order = append(order, s.LastFood)
	took := ItemNone
	for _, f := range order {
		if f != ItemNone && w.TakeStock(f, 1) {
			took = f
			break
		}
	}
	if took == ItemNone {
		for _, id := range w.Order() {
			if b := w.Buildings[id]; IsFood(b.OutItem) && b.OutN > b.res[b.OutItem] {
				b.OutN--
				took = b.OutItem
				break
			}
		}
	}
	if took == ItemNone {
		return false
	}
	s.PrevFood, s.LastFood = s.LastFood, took
	return true
}

// HomeNeeds summarises the needs of the people living in a building.
func (w *World) HomeNeeds(id int32) string {
	var met [NeedCount]int
	n := 0
	for _, s := range w.Settlers {
		if s.Home != id {
			continue
		}
		n++
		sc := w.needScores(s)
		for i := range sc {
			if sc[i] >= 0.99 {
				met[i]++
			}
		}
	}
	if n == 0 {
		return ""
	}
	out := ""
	for i := Need(0); i < NeedCount; i++ {
		mark := "ні"
		if met[i] == n {
			mark = "так"
		} else if met[i] > 0 {
			mark = "частково"
		}
		if i > 0 {
			out += "\n"
		}
		out += NeedNames[i] + ": " + mark
	}
	return out
}
