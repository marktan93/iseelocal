package ports

import "fmt"

type Allocator struct {
	start int
	end   int
}

func NewAllocator(start int, end int) Allocator {
	return Allocator{start: start, end: end}
}

func (a Allocator) Next(used map[int]bool) (int, error) {
	if a.start < 1 || a.end > 65535 || a.start > a.end {
		return 0, fmt.Errorf("invalid port range %d-%d", a.start, a.end)
	}
	for port := a.start; port <= a.end; port++ {
		if !used[port] {
			return port, nil
		}
	}
	return 0, fmt.Errorf("no remote ports available in range %d-%d", a.start, a.end)
}
