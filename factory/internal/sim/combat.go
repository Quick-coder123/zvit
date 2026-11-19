package sim

import (
	"fmt"
	"math"
)

// EnemyKind is a type of attacker. Stored in saves: only append.
type EnemyKind uint8

const (
	Crawler EnemyKind = iota
	Swarmer
	Brute
	EnemyKindCount
)

type EnemyDef struct {
	Name   string
	HP     float32
	Speed  float64 // tiles per second
	Damage float32 // per bite
	Rate   float32 // seconds between bites
	Radius float64
	Armor  float32 // flat reduction of bullet damage
	Color  uint32
}

var EnemyDefs = [EnemyKindCount]EnemyDef{
	Crawler: {Name: "Повзун", HP: 30, Speed: 1.1, Damage: 5, Rate: 1.0, Radius: 0.28, Color: 0xD9694F},
	Swarmer: {Name: "Мошка", HP: 12, Speed: 2.2, Damage: 2.5, Rate: 0.6, Radius: 0.18, Color: 0xE8A04F},
	Brute:   {Name: "Громила", HP: 240, Speed: 0.55, Damage: 22, Rate: 1.4, Radius: 0.45, Armor: 3, Color: 0x8C4A5E},
}

// Enemy is one attacker on the map.
type Enemy struct {
	Kind     EnemyKind
	X, Y     float64 // tile coordinates (centre)
	HP       float32
	Cool     float32
	Target   int32
	Retarget float32
	Phase    float32 // animation offset
	Burn     float32 // seconds of fire effect left (visual)
}

// Shot is a transient weapon effect for rendering.
type Shot struct {
	X0, Y0, X1, Y1 float32
	TTL            float32
	Flame          bool
}

const (
	FirstWaveDelay = 600.0 // seconds before the first wave
	AmmoShots      = 10    // shots per Ammo item
	CoalFuel       = 40    // flamer fuel units per Coal
	TurretAmmoCap  = 200
	FlamerFuelCap  = 200
	aggroRadius    = 7
	regenDelay     = 5 * TPS
)

// Weapon stats.
const (
	turretRange  = 9.0
	turretDamage = 7
	turretCool   = 0.18
	flamerRange  = 4.5
	flamerDPS    = 32
	flamerSplash = 1.4
	coreRange    = 7.0
	coreDamage   = 6
	coreCool     = 0.5
)

func (w *World) rollWaveAngle() float64 {
	return unit(hash3(w.Seed, w.Wave, 0, 991)) * 2 * math.Pi
}

// WaveInterval is the pause after wave n has spawned.
func WaveInterval(n int) float64 { return math.Max(150, 240-6*float64(n)) }

// WaveBudget is the strength of wave n (1-based) in crawler-equivalents.
func WaveBudget(n int) float64 { return 3 + 2.2*float64(n) + 0.12*float64(n*n) }

func (b *Building) center() (float64, float64) {
	h := float64(b.Size()) / 2
	return float64(b.Pos.X) + h, float64(b.Pos.Y) + h
}

// Core returns the core building.
func (w *World) Core() *Building {
	for _, id := range w.Order() {
		if b := w.Buildings[id]; b.Kind == KCore {
			return b
		}
	}
	return nil
}

func (w *World) notice(format string, a ...any) {
	if len(w.Notices) < 20 {
		w.Notices = append(w.Notices, fmt.Sprintf(format, a...))
	}
}

// WaveSpawnPoint is where the next wave will appear.
func (w *World) WaveSpawnPoint() (float64, float64) {
	cx, cy := 0.5, 0.5
	if c := w.Core(); c != nil {
		cx, cy = c.center()
	}
	far := 0.0
	for _, b := range w.Buildings {
		x, y := b.center()
		far = math.Max(far, math.Hypot(x-cx, y-cy))
	}
	d := math.Min(120, math.Max(32, far+16))
	return cx + math.Cos(w.WaveAngle)*d, cy + math.Sin(w.WaveAngle)*d
}

// SpawnWave creates the next wave immediately.
func (w *World) SpawnWave() {
	w.Wave++
	n := w.Wave
	sx, sy := w.WaveSpawnPoint()
	budget := WaveBudget(n)
	r := hash3(w.Seed, n, 1, 4242)
	rnd := func() float64 {
		r = splitmix(r)
		return unit(r)
	}
	add := func(k EnemyKind) {
		w.Enemies = append(w.Enemies, &Enemy{
			Kind: k, X: sx + (rnd()-0.5)*6, Y: sy + (rnd()-0.5)*6,
			HP: EnemyDefs[k].HP, Phase: float32(rnd() * 6.28),
		})
	}
	for budget > 0 {
		switch {
		case n >= 4 && budget >= 6 && rnd() < 0.2:
			add(Brute)
			budget -= 6
		case n >= 2 && rnd() < 0.35:
			add(Swarmer)
			add(Swarmer)
			budget -= 1
		default:
			add(Crawler)
			budget--
		}
	}
	w.NextWave = WaveInterval(n)
	w.WaveAngle = w.rollWaveAngle()
	w.notice("Хвиля %d атакує! Ворогів: %d", n, len(w.Enemies))
}

// updateCombat runs waves, enemies, weapons and repairs for one tick.
func (w *World) updateCombat() {
	if !w.Peaceful {
		w.NextWave -= DT
		if w.NextWave <= 0 {
			w.SpawnWave()
		}
	} else if len(w.Enemies) > 0 {
		w.Enemies = nil
	}
	w.updateEnemies()

	// fade shots
	k := 0
	for _, s := range w.Shots {
		s.TTL -= DT
		if s.TTL > 0 {
			w.Shots[k] = s
			k++
		}
	}
	w.Shots = w.Shots[:k]

	// passive repair
	if w.Tick%10 == 0 {
		rate := float32(0.01 * 10 * DT) // 1% of max HP per second
		if w.Researched["repair"] {
			rate *= 5
		}
		for _, b := range w.Buildings {
			if b.Dmg > 0 && w.Tick-b.LastHit > regenDelay {
				b.Dmg -= w.MaxHP(b) * rate
				if b.Dmg < 0 {
					b.Dmg = 0
				}
			}
		}
	}
}

// pickTarget finds the nearest blocking building near an enemy, else the core.
func (w *World) pickTarget(e *Enemy) *Building {
	ex, ey := int(math.Floor(e.X)), int(math.Floor(e.Y))
	var best *Building
	bestD := math.MaxFloat64
	for dy := -aggroRadius; dy <= aggroRadius; dy++ {
		for dx := -aggroRadius; dx <= aggroRadius; dx++ {
			b := w.At(Pos{ex + dx, ey + dy})
			if b == nil || !b.Kind.Blocks() {
				continue
			}
			d, _, _ := distToBuilding(e.X, e.Y, b)
			if d < bestD && d <= aggroRadius {
				best, bestD = b, d
			}
		}
	}
	if best == nil {
		best = w.Core()
	}
	return best
}

// distToBuilding returns the distance from a point to a building's footprint
// and the nearest point on it.
func distToBuilding(x, y float64, b *Building) (float64, float64, float64) {
	s := float64(b.Size())
	nx := math.Max(float64(b.Pos.X), math.Min(x, float64(b.Pos.X)+s))
	ny := math.Max(float64(b.Pos.Y), math.Min(y, float64(b.Pos.Y)+s))
	return math.Hypot(x-nx, y-ny), nx, ny
}

func (w *World) updateEnemies() {
	for _, e := range w.Enemies {
		def := &EnemyDefs[e.Kind]
		e.Cool -= DT
		e.Retarget -= DT
		if e.Burn > 0 {
			e.Burn -= DT
		}
		tb := w.Buildings[e.Target]
		if tb == nil || e.Retarget <= 0 {
			tb = w.pickTarget(e)
			e.Retarget = 0.7
			if tb == nil {
				continue
			}
			e.Target = tb.ID
		}
		d, nx, ny := distToBuilding(e.X, e.Y, tb)
		if d <= def.Radius+0.08 {
			if e.Cool <= 0 {
				w.Damage(tb, def.Damage)
				e.Cool = def.Rate
			}
			continue
		}
		dx, dy := (nx-e.X)/d, (ny-e.Y)/d
		step := def.Speed * DT
		x, y := e.X+dx*step, e.Y+dy*step
		// collide with whatever stands in the way
		front := Pos{int(math.Floor(x + dx*def.Radius)), int(math.Floor(y + dy*def.Radius))}
		if ob := w.At(front); ob != nil && ob.Kind.Blocks() && ob != tb {
			e.Target = ob.ID
			e.Retarget = 2
			continue
		}
		e.X, e.Y = x, y
	}
	// gentle separation so groups don't collapse into one dot
	n := len(w.Enemies)
	if n > 1 && n < 400 {
		for i := 0; i < n; i++ {
			a := w.Enemies[i]
			for j := i + 1; j < n; j++ {
				b := w.Enemies[j]
				dx, dy := b.X-a.X, b.Y-a.Y
				min := EnemyDefs[a.Kind].Radius + EnemyDefs[b.Kind].Radius
				d2 := dx*dx + dy*dy
				if d2 < min*min && d2 > 1e-9 {
					d := math.Sqrt(d2)
					push := (min - d) * 0.25
					a.X -= dx / d * push
					a.Y -= dy / d * push
					b.X += dx / d * push
					b.Y += dy / d * push
				}
			}
		}
	}
	// remove the dead
	k := 0
	for _, e := range w.Enemies {
		if e.HP > 0 {
			w.Enemies[k] = e
			k++
		} else {
			w.Kills++
		}
	}
	for i := k; i < len(w.Enemies); i++ {
		w.Enemies[i] = nil
	}
	w.Enemies = w.Enemies[:k]
}

// Damage hurts a building and destroys it when its health runs out.
func (w *World) Damage(b *Building, amount float32) {
	b.Dmg += amount
	b.LastHit = w.Tick
	if b.Dmg < w.MaxHP(b) {
		return
	}
	if b.Kind == KCore {
		w.coreFall(b)
		return
	}
	w.Ruins = append(w.Ruins, SpecOf(b))
	w.unplace(b)
	w.notice("Зруйновано: %s", Builds[b.Kind].Name)
}

func (w *World) coreFall(core *Building) {
	for i := range w.Stock {
		w.Stock[i] -= w.Stock[i] / 4
	}
	w.Enemies = nil
	core.Dmg = 0
	w.notice("Ядро впало! Втрачено чверть запасів, але вороги відступили.")
}

func (w *World) nearestEnemy(x, y, r float64) *Enemy {
	var best *Enemy
	bd := r * r
	for _, e := range w.Enemies {
		dx, dy := e.X-x, e.Y-y
		if d := dx*dx + dy*dy; d <= bd && e.HP > 0 {
			best, bd = e, d
		}
	}
	return best
}

func (w *World) shoot(b *Building, e *Enemy, dmg float32) {
	x, y := b.center()
	if !w.Researched["dmg2"] {
		dmg -= EnemyDefs[e.Kind].Armor
	}
	if dmg < 1 {
		dmg = 1
	}
	e.HP -= dmg
	if len(w.Shots) < 300 {
		w.Shots = append(w.Shots, Shot{X0: float32(x), Y0: float32(y), X1: float32(e.X), Y1: float32(e.Y), TTL: 0.07})
	}
}

func (w *World) stepTurret(b *Building) {
	b.Cool -= DT
	x, y := b.center()
	rng, dmg := turretRange, float32(turretDamage)
	if b.Kind == KTurretBig {
		rng, dmg = turretRange+4, turretDamage*2.5
	}
	e := w.nearestEnemy(x, y, rng+w.RangeBonus())
	if e == nil {
		return
	}
	b.Aim = float32(math.Atan2(e.Y-y, e.X-x))
	if b.Cool > 0 || b.Ammo <= 0 {
		return
	}
	w.shoot(b, e, dmg*w.DamageMul())
	b.Ammo--
	b.Cool = turretCool
}

func (w *World) stepCoreGun(b *Building) {
	b.Cool -= DT
	if b.Cool > 0 || len(w.Enemies) == 0 {
		return
	}
	x, y := b.center()
	if e := w.nearestEnemy(x, y, coreRange); e != nil {
		b.Aim = float32(math.Atan2(e.Y-y, e.X-x))
		w.shoot(b, e, coreDamage)
		b.Cool = coreCool
	}
}

func (w *World) stepFlamer(b *Building) {
	x, y := b.center()
	e := w.nearestEnemy(x, y, flamerRange+w.RangeBonus()*0.5)
	if e == nil || b.Ammo <= 0 {
		return
	}
	b.Aim = float32(math.Atan2(e.Y-y, e.X-x))
	dmg := float32(flamerDPS*DT) * w.DamageMul()
	for _, o := range w.Enemies {
		if math.Hypot(o.X-e.X, o.Y-e.Y) <= flamerSplash {
			o.HP -= dmg
			o.Burn = 0.4
		}
	}
	if w.Tick%6 == 0 {
		b.Ammo--
	}
	if w.Tick%3 == 0 && len(w.Shots) < 300 {
		w.Shots = append(w.Shots, Shot{X0: float32(x), Y0: float32(y), X1: float32(e.X), Y1: float32(e.Y), TTL: 0.12, Flame: true})
	}
}

// RebuildRuins puts construction sites back where buildings were destroyed.
func (w *World) RebuildRuins() (built, left int) {
	var keep []Spec
	for _, sp := range w.Ruins {
		if w.At(sp.Pos) != nil || sp.Kind.Legacy() {
			continue // something else was built there; forget the ruin
		}
		if _, err := w.PlaceSpec(sp, 0); err == nil {
			built++
		} else if err == ErrLocked {
			keep = append(keep, sp)
		}
	}
	w.Ruins = keep
	return built, len(keep)
}

// ClearRuins forgets all ruin markers.
func (w *World) ClearRuins() { w.Ruins = nil }
