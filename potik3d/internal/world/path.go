package world

import (
	"container/heap"
	"math"
)

// WalkHeight is the level a settler stands on at a tile: the surface, plus
// one on a finished dam.
func (w *World) WalkHeight(i int) int {
	h := w.Surface(i)
	if b := w.Owner(i); b != nil && b.Kind == KDam && !b.Site {
		h++
	}
	return h
}

// passable reports whether a settler may stand on tile i on the way to
// building goal (0 for none).
func (w *World) passable(i int, goal int32) bool {
	if float64(w.Height[i])+float64(w.Water[i]) > float64(w.WalkHeight(i))+0.6 {
		// deep water over the walking surface (a dam's top is dry)
		if b := w.Owner(i); b == nil || b.Kind != KDam || b.Site {
			return false
		}
	}
	if b := w.Owner(i); b != nil && b.ID != goal && !b.Site {
		switch b.Kind {
		case KDam, KFarm, KStairs:
		default:
			return false
		}
	}
	return true
}

var dirs4 = [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}}

// stairFacing reports whether tile s holds a finished stair facing tile t.
func (w *World) stairFacing(s, t int) bool {
	b := w.Owner(s)
	if b == nil || b.Kind != KStairs || b.Site || b.Dir < 0 || b.Dir > 3 {
		return false
	}
	x, y := XY(s)
	d := dirs4[b.Dir]
	return In(x+d[0], y+d[1]) && Idx(x+d[0], y+d[1]) == t
}

// CanStep is the single movement rule: may a settler walking towards
// building goal move from tile from to the neighbouring tile to?
// Orthogonal steps climb at most one level, or two between a finished stair
// and the tile it faces; diagonal steps need level ground on both sides.
func (w *World) CanStep(from, to int, goal int32) bool {
	return w.step(from, to, goal, w.passable(to, goal))
}

// step is CanStep with the destination check supplied by the caller (A*
// also accepts goal tiles that are not otherwise passable).
func (w *World) step(from, to int, goal int32, destOK bool) bool {
	if !destOK {
		return false
	}
	fx, fy := XY(from)
	tx, ty := XY(to)
	dx, dy := tx-fx, ty-fy
	if dx < -1 || dx > 1 || dy < -1 || dy > 1 || (dx == 0 && dy == 0) {
		return false
	}
	hc, hj := w.WalkHeight(from), w.WalkHeight(to)
	if dx != 0 && dy != 0 {
		// no corner cutting; diagonals only across level ground
		a, b := Idx(tx, fy), Idx(fx, ty)
		return hj == hc && w.passable(a, goal) && w.passable(b, goal) &&
			w.WalkHeight(a) == hc && w.WalkHeight(b) == hc
	}
	diff := absi(hj - hc)
	if diff <= 1 {
		return true
	}
	return diff <= 2 && (w.stairFacing(from, to) || w.stairFacing(to, from))
}

type pnode struct {
	i    int
	f    float64
	heap int
}

type pqueue []*pnode

func (q pqueue) Len() int           { return len(q) }
func (q pqueue) Less(a, b int) bool { return q[a].f < q[b].f }
func (q pqueue) Swap(a, b int)      { q[a], q[b] = q[b], q[a]; q[a].heap = a; q[b].heap = b }
func (q *pqueue) Push(x any)        { n := x.(*pnode); n.heap = len(*q); *q = append(*q, n) }
func (q *pqueue) Pop() any          { old := *q; n := old[len(old)-1]; *q = old[:len(old)-1]; return n }

var dirs8 = [8][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}, {1, 1}, {1, -1}, {-1, 1}, {-1, -1}}

// FindPath returns tiles from start (exclusive) to a tile where isGoal is
// true, walking through tiles passable for building goal. tx, ty guide the
// search. It returns nil when there is no way.
func (w *World) FindPath(start int, isGoal func(int) bool, tx, ty float64, goal int32) []int {
	if isGoal(start) {
		return []int{}
	}
	n := Size * Size
	if len(w.pathG) != n {
		w.pathG = make([]float32, n)
		w.pathFrom = make([]int32, n)
		w.pathSeen = make([]uint32, n)
	}
	w.pathGen++
	gen := w.pathGen
	hfun := func(i int) float64 {
		x, y := XY(i)
		dx, dy := math.Abs(float64(x)+0.5-tx), math.Abs(float64(y)+0.5-ty)
		return math.Max(dx, dy) + 0.414*math.Min(dx, dy)
	}
	q := &pqueue{}
	w.pathG[start], w.pathFrom[start], w.pathSeen[start] = 0, -1, gen
	heap.Push(q, &pnode{i: start, f: hfun(start)})
	closed := map[int]bool{}
	expanded := 0
	for q.Len() > 0 && expanded < 8000 {
		cur := heap.Pop(q).(*pnode)
		if closed[cur.i] {
			continue
		}
		closed[cur.i] = true
		expanded++
		if isGoal(cur.i) {
			var path []int
			for i := cur.i; i != start; i = int(w.pathFrom[i]) {
				path = append(path, i)
			}
			for a, b := 0, len(path)-1; a < b; a, b = a+1, b-1 {
				path[a], path[b] = path[b], path[a]
			}
			return path
		}
		x, y := XY(cur.i)
		hc := w.WalkHeight(cur.i)
		for k, d := range dirs8 {
			nx, ny := x+d[0], y+d[1]
			if !In(nx, ny) {
				continue
			}
			j := Idx(nx, ny)
			if closed[j] || !w.step(cur.i, j, goal, w.passable(j, goal) || isGoal(j)) {
				continue
			}
			hj := w.WalkHeight(j)
			cost := 1.0
			if k >= 4 {
				cost = math.Sqrt2
			}
			if hj != hc {
				cost += 0.6 * float64(absi(hj-hc))
			}
			if float64(w.Height[j])+float64(w.Water[j]) > float64(hj)+0.2 {
				cost += 1.5
			}
			g := float64(w.pathG[cur.i]) + cost
			if w.pathSeen[j] == gen && float64(w.pathG[j]) <= g {
				continue
			}
			w.pathSeen[j], w.pathG[j], w.pathFrom[j] = gen, float32(g), int32(cur.i)
			heap.Push(q, &pnode{i: j, f: g + hfun(j)})
		}
	}
	return nil
}
