package evictingqueue

import "testing"

func TestSimpleAdd(t *testing.T) {
  queue := New(3)

  if size := queue.Len(); size != 0 {
    t.Errorf("The queue should have a length of 0, but instead had a length of %d,", size)
  }

  queue.Add("One")
  queue.Add("Two")
  queue.Add("Three")

  if size := queue.Len(); size != 3 {
    t.Errorf("The queue should have a length of 3, but instead had a length of %d,", size)
  }

  if val, _ := queue.Get(0); val != "One" {
    t.Errorf("The first expected element was not in the queue at the expected position.")
  }

  if val, _ := queue.Get(1); val != "Two" {
    t.Errorf("The second expected element was not in the queue at the expected position.")
  }

  if val, _ := queue.Get(2); val != "Three" {
    t.Errorf("The third expected element was not in the queue at the expected position.")
  }
}

func TestEvictingAdd(t *testing.T) {
  queue := New(3)

  if size := queue.Len(); size != 0 {
    t.Errorf("The queue should have a length of 0, but instead had a length of %d,", size)
  }

  queue.Add("One")
  queue.Add("Two")
  queue.Add("Three")
  queue.Add("Four")

  if size := queue.Len(); size != 3 {
    t.Errorf("The queue should have a length of 3, but instead had a length of %d,", size)
  }

  if val, _ := queue.Get(0); val != "Two" {
    t.Errorf("The first expected element was not in the queue at the expected position.")
  }

  if val, _ := queue.Get(1); val != "Three" {
    t.Errorf("The second expected element was not in the queue at the expected position.")
  }

  if val, _ := queue.Get(2); val != "Four" {
    t.Errorf("The third expected element was not in the queue at the expected position.")
  }
}