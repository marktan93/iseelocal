package ports

import "testing"

func TestAllocatorReturnsFirstAvailablePort(t *testing.T) {
	allocator := NewAllocator(18080, 18082)

	port, err := allocator.Next(map[int]bool{18080: true})
	if err != nil {
		t.Fatalf("Next returned error: %v", err)
	}

	if port != 18081 {
		t.Fatalf("expected 18081, got %d", port)
	}
}

func TestAllocatorErrorsWhenRangeIsFull(t *testing.T) {
	allocator := NewAllocator(18080, 18081)

	_, err := allocator.Next(map[int]bool{18080: true, 18081: true})
	if err == nil {
		t.Fatal("expected full allocator to return error")
	}
}
