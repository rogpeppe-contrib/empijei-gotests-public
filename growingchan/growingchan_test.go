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

func TestRedditSuggestion(t *testing.T) {
	t.Skip("This is an actual problem with all channels operators")
	src := make(chan int)
	ch := BufLongLived(src)

	mu := sync.Mutex{}
	mu.Lock()

	go func() {
		src <- 1
		mu.Unlock()
	}()

	mu.Lock()
	mu.Unlock()

	select {
	case <-ch:
	default:
		panic("unreachable")
	}
}

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

BenchmarkQueue/1_by_1/1000/slice-14         	            3852	    294186 ns/op	     307 B/op	       5 allocs/op <==
BenchmarkQueue/1_by_1/1000/ring_buffer-14   	            4143	    294708 ns/op	     331 B/op	       5 allocs/op

BenchmarkQueue/1_by_1/10000/slice-14        	             422	   2842268 ns/op	     322 B/op	       5 allocs/op <==
BenchmarkQueue/1_by_1/10000/ring_buffer-14  	             420	   2819256 ns/op	     336 B/op	       5 allocs/op

BenchmarkQueue/send_first/1000/slice-14     	            3784	    310462 ns/op	   27674 B/op	      21 allocs/op *time
BenchmarkQueue/send_first/1000/ring_buffer-14         	    3852	    317170 ns/op	   24849 B/op	      24 allocs/op *mem

BenchmarkQueue/send_first/10000/slice-14              	     387	   3069608 ns/op	  376268 B/op	      30 allocs/op <==
BenchmarkQueue/send_first/10000/ring_buffer-14        	     387	   3120257 ns/op	  393494 B/op	      32 allocs/op

BenchmarkQueue/with_jitter/1000/slice-14              	    7381	    157133 ns/op	   12413 B/op	      18 allocs/op <==
BenchmarkQueue/with_jitter/1000/ring_buffer-14        	    7185	    160532 ns/op	   17471 B/op	      22 allocs/op

BenchmarkQueue/with_jitter/10000/slice-14             	     769	   1564188 ns/op	  162834 B/op	      25 allocs/op <==
BenchmarkQueue/with_jitter/10000/ring_buffer-14       	     753	   1562542 ns/op	  181868 B/op	      29 allocs/op

BenchmarkQueue/more_send/1000/slice-14                	    3838	    310126 ns/op	   25690 B/op	      20 allocs/op <==
BenchmarkQueue/more_send/1000/ring_buffer-14          	    3717	    317278 ns/op	   35321 B/op	      24 allocs/op

BenchmarkQueue/more_send/10000/slice-14               	     384	   3230385 ns/op	  345779 B/op	      28 allocs/op <==
BenchmarkQueue/more_send/10000/ring_buffer-14         	     388	   3131739 ns/op	  359550 B/op	      31 allocs/op *time

BenchmarkQueue/more_recv/1000/slice-14                	    7532	    158864 ns/op	   12465 B/op	      18 allocs/op <==
BenchmarkQueue/more_recv/1000/ring_buffer-14          	    7489	    160908 ns/op	   17499 B/op	      22 allocs/op

BenchmarkQueue/more_recv/10000/slice-14               	     752	   1566762 ns/op	  159937 B/op	      25 allocs/op <==
BenchmarkQueue/more_recv/10000/ring_buffer-14         	     771	   1598661 ns/op	  181581 B/op	      29 allocs/op
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
		/*
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
		*/
	}

	sizes := []int{1000, 10000}

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
