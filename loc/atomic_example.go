// You can edit this code!
// Click here and start typing.
package main

import (
	"fmt"
	"sync"
)

type A struct {
	i int
}

func (a *A) Inc(delta int) {
	a.i += delta
}

func (a *A) Get() int {
	return a.i
}

type B struct {
}

func (b *B) II(c int) {
	for i := 0; i < c; i++ {
		a.Inc(1)
	}
	wg.Done()
}

type C struct {
}

func (c *C) DD(count int) {
	for i := 0; i < count; i++ {
		a.Inc(-1)
	}
	wg.Done()
}

var a A
var b B
var c C

var wg sync.WaitGroup

func main() {
	var nCount = 1000000
	a.i = 0

	wg.Add(1)
	go b.II(nCount)

	wg.Add(1)
	go c.DD(nCount)

	wg.Wait()
	fmt.Println("Sum", a.Get())
}
