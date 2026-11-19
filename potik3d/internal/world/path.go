package world

import (
	"container/heap"
	"math"
)

// WalkHeight is the level a settler stands on at a tile (dams raise it).
func (w *World) WalkHeight(i int) int {
	h := int(w.Height[i])
	if b := w.Owner(i); b != nil && b.Kind == KDam && !b.Site {
		h++
	}
	return h
}

// passable reports whether a settler may stand on tile i on the way to
// building goal (0 for none).
func (w *World) passable(i int, goal int32) bool {
	if w.Water[i] > 0.6 {
		if b := w.Owner(i); b == nil || b.Kind != KDam || b.Site {
			return false
		}
	}
	if b := w.Owner(i); b != nil && b.ID != goal && !b.Site {
		switch b.Kind {
		case KDam, KFarm:
		default:
			return false
		}
	}
	return true
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
			if closed[j] || !(w.passable(j, goal) || isGoal(j)) {
				continue
			}
			hj := w.WalkHeight(j)
			if hj-hc > 1 || hc-hj > 1 {
				continue
			}
			cost := 1.0
			if k >= 4 {
				// no corner cutting; diagonals only across level ground
				a, b := Idx(x+d[0], y), Idx(x, y+d[1])
				if !w.passable(a, goal) || !w.passable(b, goal) || w.WalkHeight(a) != hc || w.WalkHeight(b) != hc || hj != hc {
					continue
				}
				cost = math.Sqrt2
			}
			if hj != hc {
				cost += 0.6
			}
			if w.Water[j] > 0.2 {
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
