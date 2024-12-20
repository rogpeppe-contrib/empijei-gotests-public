package growingchan

import (
	"math/rand"
	"sort"
	"strconv"
	"sync"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/rogpeppe/generic/ring"
)

func TestBufLongLived(t *testing.T) {
	tsize := 10000
	input := func() []int {
		var b []int
		for i := range tsize {
			b = append(b, i)
		}
		return b
	}()
	t.Run("buf first", func(t *testing.T) {
		in := make(chan int)
		ob := BufLongLived(in)
		for i := range input {
			in <- i
		}
		close(in)
		var got []int
		for i := range ob {
			got = append(got, i)
		}
		if diff := cmp.Diff(input, got); diff != "" {
			t.Errorf("got %v want %v diff:\n%s", got, input, diff)
		}
	})
	t.Run("parallel", func(t *testing.T) {
		in := make(chan int, tsize/2)
		ob := BufLongLived(in)
		go func() {
			defer close(in)
			for i := range input {
				in <- i
			}
		}()
		var got []int
		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range ob {
				got = append(got, i)
			}
		}()
		wg.Wait()
		if diff := cmp.Diff(input, got); diff != "" {
			t.Errorf("got %v want %v diff:\n%s", got, input, diff)
		}
	})
	t.Run("parallel many", func(t *testing.T) {
		in := make(chan int)
		ob := BufLongLived(in)

		go func() {
			defer close(in)
			var wg sync.WaitGroup
			for i := 0; i < tsize; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					in <- i
				}()
			}
			wg.Wait()
		}()
		var got []int
		{
			var wg sync.WaitGroup
			wg.Add(1)
			go func() {
				defer wg.Done()
				for i := range ob {
					got = append(got, i)
				}
			}()
			wg.Wait()
		}
		sort.IntSlice(got).Sort()
		if diff := cmp.Diff(input, got); diff != "" {
			t.Errorf("got %v want %v diff:\n%s", got, input, diff)
		}
	})
}

func TestBufLongLivedOrdered(t *testing.T) {
	tsize := 10000
	input := func() []int {
		var b []int
		for i := range tsize {
			b = append(b, i)
		}
		return b
	}()
	t.Run("buf first", func(t *testing.T) {
		in := make(chan int)
		ob := BufLongLivedTwoWorkers(in)
		for i := range input {
			in <- i
		}
		close(in)
		var got []int
		for i := range ob {
			got = append(got, i)
		}
		if diff := cmp.Diff(input, got); diff != "" {
			t.Errorf("got %v want %v diff:\n%s", got, input, diff)
		}
	})
	t.Run("parallel", func(t *testing.T) {
		in := make(chan int)
		ob := BufLongLivedTwoWorkers(in)
		go func() {
			defer close(in)
			for i := range input {
				in <- i
			}
		}()
		var got []int
		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range ob {
				got = append(got, i)
			}
		}()
		wg.Wait()
		if diff := cmp.Diff(input, got); diff != "" {
			t.Errorf("got %v want %v diff:\n%s", got, input, diff)
		}
	})
	t.Run("parallel many", func(t *testing.T) {
		in := make(chan int, tsize/2)
		ob := BufLongLived(in)

		go func() {
			defer close(in)
			var wg sync.WaitGroup
			for i := 0; i < tsize; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					in <- i
				}()
			}
			wg.Wait()
		}()
		var (
			mu  sync.Mutex
			got []int
		)
		{
			var wg sync.WaitGroup
			for i := 0; i < tsize; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					for {
						i, ok := <-ob
						if !ok {
							return
						}
						mu.Lock()
						got = append(got, i)
						mu.Unlock()
					}
				}()
			}
			wg.Wait()
		}
		sort.IntSlice(got).Sort()
		if diff := cmp.Diff(input, got); diff != "" {
			t.Errorf("got %v want %v diff:\n%s", got, input, diff)
		}
	})
}

/*
The "<==" is added for the implementation that performed best
If two performed identically or within 2% the simplest is chosen.
If there was an approximation a "*" is added to the absolute best.

One by one:

	BenchmarkQueue/1_by_1/100/slice-14         	            28870 ns/op	     288 B/op	       5 allocs/op <==
	BenchmarkQueue/1_by_1/100/ring_buffer-14   	            29945 ns/op	     312 B/op	       5 allocs/op
	BenchmarkQueue/1_by_1/100/linked_list-14   	            28956 ns/op	     288 B/op	       5 allocs/op
	BenchmarkQueue/1_by_1/100/pooled_linked_list-14         29632 ns/op	     360 B/op	       7 allocs/op

	BenchmarkQueue/1_by_1/1000/slice-14                    279611 ns/op	     288 B/op	       5 allocs/op <==
	BenchmarkQueue/1_by_1/1000/ring_buffer-14              280677 ns/op	     312 B/op	       5 allocs/op
	BenchmarkQueue/1_by_1/1000/linked_list-14              279397 ns/op	     288 B/op	       5 allocs/op * B
	BenchmarkQueue/1_by_1/1000/pooled_linked_list-14       278834 ns/op	     360 B/op	       7 allocs/op * time

	BenchmarkQueue/1_by_1/10000/slice-14                  2691739 ns/op	     298 B/op	       5 allocs/op <==
	BenchmarkQueue/1_by_1/10000/ring_buffer-14            2716783 ns/op	     321 B/op	       5 allocs/op
	BenchmarkQueue/1_by_1/10000/linked_list-14            2712676 ns/op	     288 B/op	       5 allocs/op * B
	BenchmarkQueue/1_by_1/10000/pooled_linked_list-14     2717948 ns/op	     360 B/op	       7 allocs/op

Send first:

	BenchmarkQueue/send_first/100/slice-14                  31152 ns/op	    2328 B/op	      13 allocs/op <==
	BenchmarkQueue/send_first/100/ring_buffer-14            31331 ns/op	    2352 B/op	      13 allocs/op
	BenchmarkQueue/send_first/100/linked_list-14            31276 ns/op	    2328 B/op	      13 allocs/op
	BenchmarkQueue/send_first/100/pooled_linked_list-14     31253 ns/op	    2400 B/op	      15 allocs/op

	BenchmarkQueue/send_first/1000/slice-14                303290 ns/op	   25497 B/op	      17 allocs/op <==
	BenchmarkQueue/send_first/1000/ring_buffer-14          305274 ns/op	   25522 B/op	      17 allocs/op
	BenchmarkQueue/send_first/1000/linked_list-14          303891 ns/op	   25497 B/op	      17 allocs/op
	BenchmarkQueue/send_first/1000/pooled_linked_list-14   304922 ns/op	   25568 B/op	      19 allocs/op

	BenchmarkQueue/send_first/10000/slice-14              3022709 ns/op	  357913 B/op	      24 allocs/op <==
	BenchmarkQueue/send_first/10000/ring_buffer-14        3044514 ns/op	  357938 B/op	      24 allocs/op
	BenchmarkQueue/send_first/10000/linked_list-14        3015314 ns/op	  357914 B/op	      24 allocs/op
	BenchmarkQueue/send_first/10000/pooled_linked_list-14 3003803 ns/op	  357985 B/op	      26 allocs/op * time

Jitter:

	BenchmarkQueue/with_jitter/100/slice-14                 15437 ns/op	    1228 B/op	      12 allocs/op <==
	BenchmarkQueue/with_jitter/100/ring_buffer-14           15346 ns/op	    1249 B/op	      12 allocs/op * time
	BenchmarkQueue/with_jitter/100/linked_list-14           15346 ns/op	    1228 B/op	      12 allocs/op * time/B
	BenchmarkQueue/with_jitter/100/pooled_linked_list-14    15350 ns/op	    1299 B/op	      14 allocs/op

	BenchmarkQueue/with_jitter/1000/slice-14               153032 ns/op	   11202 B/op	      15 allocs/op <==
	BenchmarkQueue/with_jitter/1000/ring_buffer-14         153418 ns/op	   11285 B/op	      15 allocs/op
	BenchmarkQueue/with_jitter/1000/linked_list-14         153326 ns/op	   11259 B/op	      15 allocs/op
	BenchmarkQueue/with_jitter/1000/pooled_linked_list-14  152349 ns/op	   11304 B/op	      17 allocs/op

	BenchmarkQueue/with_jitter/10000/slice-14             1513335 ns/op	  151453 B/op	      21 allocs/op <==
	BenchmarkQueue/with_jitter/10000/ring_buffer-14       1523959 ns/op	  152704 B/op	      21 allocs/op
	BenchmarkQueue/with_jitter/10000/linked_list-14       1510417 ns/op	  152154 B/op	      21 allocs/op * time
	BenchmarkQueue/with_jitter/10000/pooled_linked_list-1 1533025 ns/op	  154405 B/op	      23 allocs/op

More send:

	BenchmarkQueue/more_send/100/slice-14                   30849 ns/op	    2293 B/op	      13 allocs/op <==
	BenchmarkQueue/more_send/100/ring_buffer-14             30786 ns/op	    2315 B/op	      13 allocs/op
	BenchmarkQueue/more_send/100/linked_list-14             30454 ns/op	    2286 B/op	      13 allocs/op * time/B
	BenchmarkQueue/more_send/100/pooled_linked_list-14      30658 ns/op	    2362 B/op	      15 allocs/op

	BenchmarkQueue/more_send/1000/slice-14                 302285 ns/op	   24196 B/op	      16 allocs/op * time
	BenchmarkQueue/more_send/1000/ring_buffer-14           306708 ns/op	   24064 B/op	      16 allocs/op <==!
	BenchmarkQueue/more_send/1000/linked_list-14           304913 ns/op	   24088 B/op	      16 allocs/op
	BenchmarkQueue/more_send/1000/pooled_linked_list-14    305184 ns/op	   24081 B/op	      18 allocs/op

	BenchmarkQueue/more_send/10000/slice-14               3020093 ns/op	  332577 B/op	      23 allocs/op
	BenchmarkQueue/more_send/10000/ring_buffer-14         3110136 ns/op	  338783 B/op	      23 allocs/op
	BenchmarkQueue/more_send/10000/linked_list-14         3010354 ns/op	  332460 B/op	      23 allocs/op
	BenchmarkQueue/more_send/10000/pooled_linked_list-14  3009928 ns/op	  331923 B/op	      25 allocs/op <==

More recv:

	BenchmarkQueue/more_recv/100/slice-14                   15504 ns/op	    1228 B/op	      12 allocs/op
	BenchmarkQueue/more_recv/100/ring_buffer-14             15503 ns/op	    1248 B/op	      12 allocs/op
	BenchmarkQueue/more_recv/100/linked_list-14             15529 ns/op	    1225 B/op	      12 allocs/op <==
	BenchmarkQueue/more_recv/100/pooled_linked_list-14      15465 ns/op	    1297 B/op	      14 allocs/op

	BenchmarkQueue/more_recv/1000/slice-14                 153664 ns/op	   11231 B/op	      15 allocs/op <==
	BenchmarkQueue/more_recv/1000/ring_buffer-14           154177 ns/op	   11239 B/op	      15 allocs/op
	BenchmarkQueue/more_recv/1000/linked_list-14           153735 ns/op	   11251 B/op	      15 allocs/op
	BenchmarkQueue/more_recv/1000/pooled_linked_list-14    153226 ns/op	   11315 B/op	      17 allocs/op

	BenchmarkQueue/more_recv/10000/slice-14               1532120 ns/op	  152869 B/op	      21 allocs/op
	BenchmarkQueue/more_recv/10000/ring_buffer-14         1528769 ns/op	  153357 B/op	      21 allocs/op * time
	BenchmarkQueue/more_recv/10000/linked_list-14         1533995 ns/op	  152676 B/op	      21 allocs/op <==
	BenchmarkQueue/more_recv/10000/pooled_linked_list-14  1549247 ns/op	  154387 B/op	      23 allocs/op
*/
func BenchmarkQueue(b *testing.B) {
	tests := []struct {
		name string
		r    func(b *testing.B, qf func() Queue[int], size int)
	}{
		{"1 by 1", runB1b1},
		{"send first", runBsendFirst},
		{"with jitter", runBjitter},
		{"more send", runBmoreSend},
		{"more recv", runBmoreRecv},
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
		{"ring buffer",
			func() Queue[int] {
				var rb ring.Buffer[int]
				return &rb
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

	sizes := []int{100, 1000, 10000}

	for _, t := range tests {
		b.Run(t.name, func(b *testing.B) {
			for _, s := range sizes {
				b.Run(strconv.Itoa(s), func(b *testing.B) {
					for _, i := range impls {
						b.Run(i.name, func(b *testing.B) {
							t.r(b, i.ctor, s)
						})
					}
				})
			}
		})
	}
}

func runB1b1(b *testing.B, qf func() Queue[int], size int) {
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		in := make(chan int)
		out := BufLongLivedCustomQueue(in, qf())
		for range size {
			in <- 1
			<-out
		}
		close(in)
		for range out {
		}
	}
}

func runBsendFirst(b *testing.B, qf func() Queue[int], size int) {
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		in := make(chan int)
		out := BufLongLivedCustomQueue(in, qf())
		for range size {
			in <- 1
		}
		close(in)
		for range out {
		}
	}
}

func runBjitter(b *testing.B, qf func() Queue[int], size int) {
	b.ReportAllocs()
	b.ResetTimer()
	const jitter = 10
	for range b.N {
		in := make(chan int)
		out := BufLongLivedCustomQueue(in, qf())
		for range jitter {
			for range rand.Intn(size / jitter) {
				in <- 1
			}
			for range rand.Intn(size / jitter) {
				select {
				case <-out:
				default:
				}
			}
		}
		close(in)
		for range out {
		}
	}
}

func runBmoreSend(b *testing.B, qf func() Queue[int], size int) {
	b.ReportAllocs()
	b.ResetTimer()
	const jitter = 10
	for range b.N {
		in := make(chan int)
		out := BufLongLivedCustomQueue(in, qf())
		for range jitter {
			for range rand.Intn(size / jitter * 2) {
				in <- 1
			}
			for range rand.Intn(size / jitter) {
				select {
				case <-out:
				default:
				}
			}
		}
		close(in)
		for range out {
		}
	}
}

func runBmoreRecv(b *testing.B, qf func() Queue[int], size int) {
	b.ReportAllocs()
	b.ResetTimer()
	const jitter = 10
	for range b.N {
		in := make(chan int)
		out := BufLongLivedCustomQueue(in, qf())
		for range jitter {
			for range rand.Intn(size / jitter) {
				in <- 1
			}
			for range rand.Intn(size / jitter * 2) {
				select {
				case <-out:
				default:
				}
			}
		}
		close(in)
		for range out {
		}
	}
}
