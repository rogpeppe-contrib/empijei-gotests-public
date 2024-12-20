package growingchan

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestQueues(t *testing.T) {
	insert := func(q Queue[int], qt int) {
		for i := range qt {
			q.PushStart(i)
		}
	}
	pop := func(q Queue[int], qt int) {
		for range qt {
			q.PopEnd()
		}
	}
	tests := []struct {
		name string
		ops  func(Queue[int])
		want []int
	}{
		{
			name: "insert 5",
			ops: func(q Queue[int]) {
				for i := range 5 {
					q.PushStart(i)
				}
			},
			want: []int{0, 1, 2, 3, 4},
		},
		{
			name: "insert 5, pop 3, insert 3",
			ops: func(q Queue[int]) {
				insert(q, 5)
				pop(q, 3)
				insert(q, 3)
			},
			want: []int{3, 4, 0, 1, 2},
		},
		{
			name: "insert 5, pop 5, insert 3",
			ops: func(q Queue[int]) {
				insert(q, 5)
				pop(q, 5)
				insert(q, 3)
			},
			want: []int{0, 1, 2},
		},
		{
			name: "insert 5, pop 3, insert 5, pop 2",
			ops: func(q Queue[int]) {
				insert(q, 5)
				pop(q, 3)
				insert(q, 3)
				pop(q, 2)
			},
			want: []int{0, 1, 2},
		},
	}
	impls := []struct {
		name string
		ctor func() Queue[int]
	}{
		{"slice",
			func() Queue[int] {
				return &sliceQueue[int]{}
			},
		},
		{"linked list",
			func() Queue[int] {
				var ll linkedListQueue[int]
				return &ll
			},
		},
		{"pooled linked list",
			func() Queue[int] {
				return newPooled[int]()
			},
		},
	}

	for _, i := range impls {
		for _, tt := range tests {
			t.Run(i.name+"/"+tt.name, func(t *testing.T) {
				q := i.ctor()
				tt.ops(q)
				var got []int
				for q.Len() > 0 {
					got = append(got, q.PopEnd())
				}
				if diff := cmp.Diff(tt.want, got); diff != "" {
					t.Errorf("got %v want %v diff:\n%s", got, tt.want, diff)
				}
			})
		}
	}
}
