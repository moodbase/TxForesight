package txfpool

import (
	"testing"
)

func TestPageInfo(t *testing.T) {
	a := []int{1, 2, 3, 4, 5}
	total := len(a)

	tests := []struct {
		page     int
		pageSize int
		start    int
		end      int
	}{
		{-1, 2, 0, 2},
		{1, 10, 0, 5},
		{1, 2, 0, 2},
		{2, 2, 2, 4},
		{3, 2, 4, 5},
		{4, 2, 5, 5},
	}
	for i, test := range tests {
		start, end := pageInfo(test.page, test.pageSize, total)
		if start != test.start || end != test.end {
			t.Errorf("test %d: pageInfo(%d, %d, %d) = %d, %d, want %d, %d", i, test.page, test.pageSize, total, start, end, test.start, test.end)
		}
	}
}
