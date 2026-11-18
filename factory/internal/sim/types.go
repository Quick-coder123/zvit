// Package sim contains the factory simulation. It has no graphics
// dependencies so it can be tested and run headless (e.g. by a host).
package sim

// Dir is a cardinal direction in screen space (Y grows downwards).
type Dir uint8

const (
	East Dir = iota
	South
	West
	North
)

var dirDX = [4]int{1, 0, -1, 0}
var dirDY = [4]int{0, 1, 0, -1}

func (d Dir) DX() int     { return dirDX[d&3] }
func (d Dir) DY() int     { return dirDY[d&3] }
func (d Dir) Opp() Dir    { return (d + 2) & 3 }
func (d Dir) CW() Dir     { return (d + 1) & 3 }
func (d Dir) CCW() Dir    { return (d + 3) & 3 }
func (d Dir) Valid() bool { return d < 4 }

// DirBetween returns the direction from a to an orthogonally adjacent b.
func DirBetween(a, b Pos) (Dir, bool) {
	switch {
	case b.X == a.X+1 && b.Y == a.Y:
		return East, true
	case b.X == a.X-1 && b.Y == a.Y:
		return West, true
	case b.Y == a.Y+1 && b.X == a.X:
		return South, true
	case b.Y == a.Y-1 && b.X == a.X:
		return North, true
	}
	return East, false
}

// Pos is a tile coordinate.
type Pos struct{ X, Y int }

func (p Pos) Step(d Dir) Pos   { return Pos{p.X + d.DX(), p.Y + d.DY()} }
func (p Pos) Add(x, y int) Pos { return Pos{p.X + x, p.Y + y} }

func floorDiv(a, b int) int {
	q := a / b
	if (a%b != 0) && ((a < 0) != (b < 0)) {
		q--
	}
	return q
}

func floorMod(a, b int) int { return a - floorDiv(a, b)*b }
