package guardian

import (
	"testing"
)

func TestMergeUnique(t *testing.T) {
	cases := []struct {
		existing []string
		newItems []string
		wantLen  int
	}{
		{[]string{"a", "b"}, []string{"c"}, 3},
		{[]string{"a", "b"}, []string{"b", "c"}, 3},      // b 重复，不增加
		{[]string{"a"}, []string{"a", "a", "a"}, 1},       // 全部重复
		{nil, []string{"x", "y"}, 2},                       // existing 为 nil
		{[]string{"a"}, nil, 1},                            // newItems 为 nil
	}
	for _, c := range cases {
		got := mergeUnique(c.existing, c.newItems)
		if len(got) != c.wantLen {
			t.Errorf("mergeUnique(%v, %v) len=%d, want %d; result=%v",
				c.existing, c.newItems, len(got), c.wantLen, got)
		}
		// 验证无重复
		seen := make(map[string]int)
		for _, s := range got {
			seen[s]++
		}
		for k, v := range seen {
			if v > 1 {
				t.Errorf("duplicate item %q in result %v", k, got)
			}
		}
	}
}
