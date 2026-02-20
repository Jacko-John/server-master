package utils

import (
	"math/rand"
	"sync"
)

type Queue[T comparable] struct {
	items           []T
	zero            T
	l, r, size, cnt int
	rw              sync.RWMutex
}

func NewQueue[T comparable](size int) *Queue[T] {
	return &Queue[T]{items: make([]T, size), size: size, cnt: 0, l: 0, r: -1}
}

func (q *Queue[T]) Enqueue(item T) {
	q.rw.Lock()
	defer q.rw.Unlock()
	if q.size == 0 || q.cnt == q.size {
		return
	}
	q.r = (q.r + 1) % q.size
	q.items[q.r] = item
	q.cnt++
}

func (q *Queue[T]) Dequeue() T {
	q.rw.Lock()
	defer q.rw.Unlock()
	if q.size == 0 || q.cnt == 0 {
		return q.zero
	}
	item := q.items[q.l]
	q.l = (q.l + 1) % q.size
	q.cnt--
	return item
}

func (q *Queue[T]) IsEmpty() bool {
	q.rw.RLock()
	defer q.rw.RUnlock()
	return q.cnt == 0
}

func (q *Queue[T]) IsFull() bool {
	q.rw.RLock()
	defer q.rw.RUnlock()
	return q.cnt == q.size
}

func (q *Queue[T]) Size() int {
	q.rw.RLock()
	defer q.rw.RUnlock()
	return q.cnt
}

func (q *Queue[T]) Clear() {
	q.rw.Lock()
	defer q.rw.Unlock()
	q.l = 0
	q.r = -1
	q.cnt = 0
}

func (q *Queue[T]) Has(item T) bool {
	q.rw.RLock()
	defer q.rw.RUnlock()
	for i := 0; i < q.cnt; i++ {
		if q.items[(q.l+i)%q.size] == item {
			return true
		}
	}
	return false
}

func (q *Queue[T]) Rand() T {
	q.rw.RLock()
	defer q.rw.RUnlock()
	switch q.cnt {
	case 0:
		return q.zero
	case 1:
		return q.items[q.l]
	case 2:
		return q.items[(q.l+1)%q.size]
	}
	l := q.l + 1
	rc := q.cnt - 1
	t := rc * (rc + 1) / 2
	rd := rand.Intn(t)
	for i := range rc {
		if rd < i+1 {
			return q.items[(l+i)%q.size]
		}
		rd -= i + 1
	}
	return q.items[q.r]
}
