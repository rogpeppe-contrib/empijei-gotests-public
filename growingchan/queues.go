package growingchan

import (
	"log"
	"sync"
)

// Slice

var _ Queue[int] = &sliceQueue[int]{}

type sliceQueue[T any] struct {
	q      []T
	allocs int
}

func (sq *sliceQueue[T]) Cap() int {
	return cap(sq.q)
}

func (sq *sliceQueue[T]) Len() int {
	return len(sq.q)
}

func (sq *sliceQueue[T]) SetCap(n int) {}

func (sq *sliceQueue[T]) PopEnd() T {
	v := sq.q[0]
	sq.q = sq.q[1:]
	log.Printf("PopEnd -> %v:%v", len(sq.q), cap(sq.q))
	return v
}

func (sq *sliceQueue[T]) PushStart(v T) {
	var old *T
	if len(sq.q) > 0 {
		old = &(sq.q)[0]
	}
	sq.q = append(sq.q, v)
	if &sq.q[0] != old {
		sq.allocs++
	}
	log.Printf("PushStart -> %v:%v (new %v)", len(sq.q), cap(sq.q), &sq.q[0] != old)
}

// LinkedList

var _ Queue[int] = &linkedListQueue[int]{}

type elem[T any] struct {
	v    T
	next *elem[T]
}

type linkedListQueue[T any] struct {
	len  int
	head *elem[T]
	tail *elem[T]
}

func (sq *linkedListQueue[T]) Cap() int {
	// A list is as long as it is capacious
	return sq.len
}

func (sq *linkedListQueue[T]) Len() int {
	return sq.len
}

func (sq *linkedListQueue[T]) SetCap(newCap int) {
	// This doesn't apply to a List
}

func (sq *linkedListQueue[T]) PopEnd() T {
	if sq.head == nil {
		panic("pop from empty list")
	}
	sq.len--
	v := sq.head.v
	sq.head = sq.head.next
	if sq.head == nil {
		sq.tail = nil
	}
	return v
}

func (sq *linkedListQueue[T]) PushStart(v T) {
	sq.len++
	var e elem[T] = elem[T]{v: v}
	if sq.tail == nil {
		sq.head = &e
		sq.tail = &e
		return
	}
	sq.tail.next = &e
	sq.tail = &e
}

// LinkedList with mempool

var _ Queue[int] = &linkedListPooledQueue[int]{}

type linkedListPooledQueue[T any] struct {
	len  int
	p    *sync.Pool
	head *elem[T]
	tail *elem[T]
}

func newPooled[T any]() *linkedListPooledQueue[T] {
	return &linkedListPooledQueue[T]{
		p: &sync.Pool{
			New: func() any {
				return &elem[T]{}
			},
		},
	}
}

func (sq *linkedListPooledQueue[T]) Cap() int {
	// A list is as long as it is capacious
	return sq.len
}

func (sq *linkedListPooledQueue[T]) Len() int {
	return sq.len
}

func (sq *linkedListPooledQueue[T]) SetCap(newCap int) {
	// This doesn't apply to a List
}

func (sq *linkedListPooledQueue[T]) PopEnd() T {
	if sq.head == nil {
		panic("pop from empty list")
	}
	sq.len--
	oldHead := sq.head
	v := oldHead.v
	sq.head = oldHead.next
	sq.p.Put(oldHead)
	if sq.head == nil {
		sq.tail = nil
	}
	return v
}

func (sq *linkedListPooledQueue[T]) PushStart(v T) {
	sq.len++
	e := sq.p.Get().(*elem[T])
	e.v = v
	if sq.tail == nil {
		sq.head = e
		sq.tail = e
		return
	}
	sq.tail.next = e
	sq.tail = e
}
