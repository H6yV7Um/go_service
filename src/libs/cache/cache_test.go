package cache

import "testing"

func TestCache_HGet(t *testing.T) {
	c := New("test", 0, 0, 0)
	c.HSet("test", "a", 1, 0)
	n, _ := c.HGet("test", "a")
	if n.(int) != 1 {
		t.Fail()
	}
}
