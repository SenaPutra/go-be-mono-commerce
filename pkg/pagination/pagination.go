package pagination

import "strconv"

func Parse(pageS, sizeS string) (int, int) {
	p, _ := strconv.Atoi(pageS)
	s, _ := strconv.Atoi(sizeS)
	if p < 1 {
		p = 1
	}
	if s < 1 || s > 100 {
		s = 10
	}
	return p, s
}
