package evictingqueue

import "sync"

//
// EvictingQueue is a thread-safe queue structure that automatically maintains the desired maximum
// size by evicting its oldest element if a new element is being added when at capacity. It is
// modeled after the EvictingQueue class from the Google Guava library for Java.
//
type EvictingQueue struct {
  mu *sync.Mutex
  size int
  queue []interface{}
}

//
// New instantiates a new evicting queue with the specified maximum size.
//
func New(maxSize int) *EvictingQueue {
  return &EvictingQueue{
    mu: &sync.Mutex{},
    size: maxSize,
    queue: make([]interface{}, 0),
  }
}

//
// Add appends the provided element to the evicting queue and evicts the oldest element if necessary
// to maintain its maximum size.
//
func (o *EvictingQueue) Add(e interface{}) {
  o.mu.Lock()
  defer o.mu.Unlock()

  //
  // Remove the oldest element from the tail of the queue if we are currently at capacity.
  //
  if len(o.queue) == o.size {
    o.queue = o.queue[1:]
  }

  //
  // Append the new element to head of the queue.
  //
  o.queue = append(o.queue, e)
}

//
// Get returns the element that exists at the specified index of the queue and a true sentinel, or
// nil and a false sentinel if the index is out-of-range.
//
func (o *EvictingQueue) Get(index int) (interface{}, bool) {
  o.mu.Lock()
  defer o.mu.Unlock()

  if index > len(o.queue) {
    return nil, false
  }

  return o.queue[index], true
}

//
// Len returns the current length of the queue.
//
func (o *EvictingQueue) Len() int {
  return len(o.queue)
}