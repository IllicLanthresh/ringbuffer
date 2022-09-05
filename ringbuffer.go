package ringbuffer

import (
	"fmt"
	"sync"
)

// RingBuffer is the main struct of the package.
// It's only exported so that it can be referenced in field and variable declarations.
// To create a new RingBuffer, use the New function.
type RingBuffer[T any] struct {
	dataMx      sync.Mutex
	pushQueue   chan *T
	capacity    uint
	head        uint
	tail        uint
	data        []*T
	closeBuffer chan struct{}
	closed      bool
	waiterMx    sync.Mutex
	waiters     []func()
}

// New creates a new ring buffer of type T with the specified capacity and push buffer capacity.
// The capacity is the maximum number of elements that can be stored in the buffer.
// The ring buffer performs push operations in a dedicated goroutine, sequentially.
// Those operations are buffered so that Push calls do not block execution,
// pushBufferSize is the maximum number of push operations that can be buffered.
func New[T any](capacity uint, pushBufferSize uint) *RingBuffer[T] {
	ring := &RingBuffer[T]{
		pushQueue:   make(chan *T, pushBufferSize),
		capacity:    capacity,
		data:        make([]*T, capacity),
		closeBuffer: make(chan struct{}),
	}

	go func() {
		for {
			select {
			case v := <-ring.pushQueue:
				ring.dataMx.Lock()
				ring.push(v)
				ring.dataMx.Unlock()
			default:
				select {
				case <-ring.closeBuffer:
					close(ring.pushQueue)
					close(ring.closeBuffer)
					ring.closed = true
					return
				default:
					ring.waiterMx.Lock()
					if len(ring.waiters) > 0 {
						for _, waiter := range ring.waiters {
							waiter()
						}
					}
					ring.waiters = nil
					ring.waiterMx.Unlock()
				}

			}
		}
	}()

	return ring
}

func (r *RingBuffer[T]) push(v *T) {
	r.data[r.head] = v
	r.head = (r.head + 1) % r.capacity
	if r.head == r.tail {
		r.tail = (r.tail + 1) % r.capacity
	}
}

func (r *RingBuffer[T]) Push(v ...*T) {
	if r.closed {
		panic("attempted to push to closed ring buffer")
	}
	for _, v := range v {
		r.pushQueue <- v
	}
}

func (r *RingBuffer[T]) pop() (*T, bool) {
	if r.head == r.tail {
		return nil, false
	}
	v := r.data[r.tail]
	r.tail = (r.tail + 1) % r.capacity
	return v, true
}

func (r *RingBuffer[T]) Pop() (*T, bool) {
	r.dataMx.Lock()
	defer r.dataMx.Unlock()
	return r.pop()
}

func (r *RingBuffer[T]) Len() uint {
	if r.head >= r.tail {
		return r.head - r.tail
	}
	return r.capacity - r.tail + r.head
}

func (r *RingBuffer[T]) Reset() {
	r.head = 0
	r.tail = 0
}

func (r *RingBuffer[T]) Flush() []*T {
	var result []*T
	for {
		v, ok := r.Pop()
		if !ok {
			break
		}
		result = append(result, v)
	}
	return result
}

// TODO: add resize

// Iterate over the ring buffer in order, from oldest to newest.
// Without popping any elements from the ring.
func (r *RingBuffer[T]) Iterate() <-chan *T {
	ch := make(chan *T)
	go func() {
		for i := r.tail; i != r.head; i = (i + 1) % r.capacity {
			ch <- r.data[i]
		}
		close(ch)
	}()
	return ch
}

// IterateReverse iterates over the ring buffer in reverse order, from newest to oldest.
// Without popping any elements from the ring.
func (r *RingBuffer[T]) IterateReverse() <-chan *T {
	ch := make(chan *T)
	go func() {
		for i := r.head; i != r.tail; i = (i + r.capacity - 1) % r.capacity {
			ch <- r.data[i]
		}
		close(ch)
	}()
	return ch
}

func (r *RingBuffer[T]) String() string {
	return fmt.Sprintf("ringBuffer{capacity: %d, head: %d, tail: %d, data: %v}", r.capacity, r.head, r.tail, r.data)
}

func (r *RingBuffer[T]) Close() {
	if r.closed {
		panic("attempted to close already closed ring buffer")
	}
	r.closeBuffer <- struct{}{}
}

func (r *RingBuffer[T]) FlushAndClose() []*T {
	r.Close()
	return r.Flush()
}

func (r *RingBuffer[T]) IsClosed() bool {
	return r.closed
}

// AddQueueIdleHook adds a function to be called when the ring buffer push queue is idle.
func (r *RingBuffer[T]) AddQueueIdleHook(f func()) {
	if r.closed {
		panic("attempted to execute on closed ring buffer")
	}
	r.waiterMx.Lock()
	r.waiters = append(r.waiters, f)
	r.waiterMx.Unlock()
}

// PushAndWait pushes the given value to the ring and blocks execution until the ring has acknowledged the push.
// Keep in mind that this will not only block until the values passed have been pushed,
// it will potentially block until other values coming from other goroutines have been pushed as well.
func (r *RingBuffer[T]) PushAndWait(v ...*T) {
	if r.closed {
		panic("attempted to push to closed ring buffer")
	}
	r.Push(v...)
	var wg sync.WaitGroup
	wg.Add(1)
	r.AddQueueIdleHook(func() {
		wg.Done()
	})
	wg.Wait()
}

func (r *RingBuffer[T]) Wait() {
	if r.closed {
		panic("attempted to wait on closed ring buffer")
	}
	var wg sync.WaitGroup
	wg.Add(1)
	r.AddQueueIdleHook(func() {
		wg.Done()
	})
	wg.Wait()
}
