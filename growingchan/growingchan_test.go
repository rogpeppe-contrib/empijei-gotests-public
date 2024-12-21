package growingchan

import (
	"log"
	"math/rand"
	"sort"
	"sync"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/rogpeppe/generic/ring"
)

func TestBenchWhatGoesOn(t *testing.T) {
	var q *sliceQueue[int]
	runBjitter(3000, func() Queue[int] {
		q = &sliceQueue[int]{}
		return q
	})
	log.Printf("allocs: %d", q.allocs)
}

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
		r    func(n int, qf func() Queue[int])
	}{
		{"1_by_11", runB1b1},
		{"send_first", runBsendFirst},
		{"with_jitter", runBjitter},
		{"concurrent", runBconcurrent},
		//{"more send", runBmoreSend},
		//{"more recv", runBmoreRecv},
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

	for _, t := range tests {
		b.Run(t.name, func(b *testing.B) {
			for _, i := range impls {
				b.Run(i.name, func(b *testing.B) {
					b.ReportAllocs()
					t.r(b.N, i.ctor)
				})
			}
		})
	}
}

func runB1b1(n int, qf func() Queue[int]) {
	in := make(chan int)
	out := BufLongLivedCustomQueue(in, qf())
	for range n {
		in <- 1
		<-out
	}
	close(in)
	for range out {
	}
}

func runBsendFirst(n int, qf func() Queue[int]) {
	in := make(chan int)
	out := BufLongLivedCustomQueue(in, qf())
	for range n {
		for range 10 {
			in <- 1
		}
		for range 10 {
			<-out
		}
	}
	close(in)
	for range out {
	}
}

func runBconcurrent(n int, qf func() Queue[int]) {
	in := make(chan int)
	out := BufLongLivedCustomQueue(in, qf())

	go func() {
		defer close(in)
		for range n {
			in <- 1
			//if (i & 0xff) == 0 {
			//	if atomic.AddInt64(&size, 256) > 1000 {
			//		time.Sleep(0)
			//	}
			//}
		}
	}()

	for range n {
		<-out
		//if (i & 0xff) == 0 {
		//	atomic.AddInt64(&size, -256)
		//}
	}
}

func runBjitter(n int, qf func() Queue[int]) {
	const jitter = 10
	in := make(chan int)
	out := BufLongLivedCustomQueue(in, qf())
	size := 0
	for range n {
		if rand.Intn(2) == 0 {
			in <- 1
			size++
		} else if size > 0 {
			<-out
			size--
		}
	}
	close(in)
	for range out {
	}
}

func runBmoreSend(n int, qf func() Queue[int], size int) {
	const jitter = 10
	for range n {
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

func runBmoreRecv(n int, qf func() Queue[int], size int) {
	const jitter = 10
	for range n {
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
