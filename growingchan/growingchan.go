package growingchan

import "sync"

func BufShortLived[T any](in <-chan T) <-chan T {
	var buf []T
	out := make(chan T)
	go func() {
		defer close(out)
		for v := range in {
			select {
			case out <- v:
			default:
				buf = append(buf, v)
			}
		}
		for _, v := range buf {
			out <- v
		}
	}()
	return out
}

func BufLongLived[T any](in <-chan T) <-chan T {
	out := make(chan T)
	go func() {
		defer close(out)
		var (
			queue []T
			next  T
			nout  chan T
		)
		for in != nil || nout != nil {
			select {
			case nout <- next:
				if len(queue) == 0 {
					nout = nil
					continue
				}
				next = queue[0]
				queue = queue[1:]
				continue
			default:
			}
			select {
			case v, ok := <-in:
				if !ok {
					in = nil
					continue
				}
				if nout == nil {
					nout = out
					next = v
				} else {
					queue = append(queue, v)
				}
			case nout <- next:
				if len(queue) == 0 {
					nout = nil
					continue
				}
				next = queue[0]
				queue = queue[1:]
			}
		}
	}()
	return out
}

func BufLongLivedTwoWorkers[T any](in <-chan T) <-chan T {
	out := make(chan T)

	done := make(chan struct{})
	var (
		// Mutex hat for the following fields
		mu      sync.Mutex
		barrier chan struct{}
		buf     []T
	)

	consumer := func() { // Consumer of in
		defer close(done) // Signal producer that we're done
		for v := range in {
			mu.Lock()
			buf = append(buf, v)
			// If consumer is waiting on us
			if barrier != nil {
				close(barrier)
				barrier = nil
			}
			mu.Unlock()
		}
	}

	producer := func() { // Producer of out
		defer close(out) // Signal our caller that we're done
		for {
			v, wait := func() (v T, wait chan struct{}) {
				mu.Lock()
				defer mu.Unlock()
				if len(buf) == 0 {
					if barrier == nil {
						barrier = make(chan struct{})
					}
					// We have no values to send, let's wait on the barrier
					return v, barrier
				}
				v = buf[0]
				buf = buf[1:]
				return v, nil
			}()
			if wait != nil {
				// It's important that this wait happens while we are not holding
				// the mutex's lock, or we'll deadlock.
				select {
				case <-wait:
					continue
				case <-done:
					return
				}
			}
			out <- v
		}
	}

	go consumer()
	go producer()
	return out
}

type Queue[T any] interface {
	Len() int
	Cap() int
	SetCap(v int)

	PopEnd() (t T)
	PushStart(t T)
}

func BufLongLivedCustomQueue[T any](in <-chan T, q Queue[T]) <-chan T {
	out := make(chan T)
	go func() {
		defer close(out)
		var (
			queue []T
			next  T
			nout  chan T
		)
		for in != nil || nout != nil {
			select {
			case nout <- next:
				if len(queue) == 0 {
					nout = nil
					continue
				}
				next = queue[0]
				queue = queue[1:]
				continue
			default:
			}
			select {
			case v, ok := <-in:
				if !ok {
					in = nil
					continue
				}
				if nout == nil {
					nout = out
					next = v
				} else {
					queue = append(queue, v)
				}
			case nout <- next:
				if len(queue) == 0 {
					nout = nil
					continue
				}
				next = queue[0]
				queue = queue[1:]
			}
		}
	}()
	return out
}
