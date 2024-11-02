package experience

import "math"

const (
	a = 0.15
	b = float64(2)
)

func XPToLevel(xp int) int {
	level := a * math.Pow(float64(xp), 1/b)
	return int(level)
}

func LevelToXP(level int) int {
	xp := math.Pow(float64(level)/a, b)
	return int(math.Ceil(xp))
}
