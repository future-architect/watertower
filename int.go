package watertower

import (
	"sort"
)

func intersection(sorted ...[]uint32) []uint32 {
	sort.Slice(sorted, func(i, j int) bool {
		return len(sorted[i]) < len(sorted[j])
	})
	var result []uint32
	if len(sorted) == 0 || len(sorted[0]) == 0 {
		return result
	}
	cursors := make([]int, len(sorted))
	terminate := false
	for _, value := range sorted[0] {
		needIncrement := false
		for i := 1; i < len(sorted); i++ {
			found := false
			for j := cursors[i]; j < len(sorted[i]); j++ {
				valueOfOtherSlice := sorted[i][cursors[i]]
				if valueOfOtherSlice < value {
					cursors[i] = j + 1
				} else if value < valueOfOtherSlice {
					needIncrement = true
					break
				} else {
					found = true
					break
				}
			}
			if needIncrement {
				break
			}
			if !found {
				terminate = true
				break
			}
		}
		if terminate {
			break
		}
		if !needIncrement {
			result = append(result, value)
		}
	}
	return result
}
