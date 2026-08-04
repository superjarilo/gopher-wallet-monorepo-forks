// baseURL/part_2/2.1.7/3.go
package main

import "fmt"

type InputFunc func(int, int) int
type OutputFunc func(int) int

func myClosure(a, b int, foo InputFunc) OutputFunc {
	return func(value int) int {
		return value - foo(a, b)
	}
}
func main() {
	globalValue := 99
	mainFunc := func(a, b int) int {
		globalValue--
		fmt.Printf("globalValue: %d, ", globalValue)
		if a < b {
			return globalValue - a + b
		}
		return -globalValue + b*a
	}
	calculation := myClosure(3, 5, mainFunc)
	fmt.Println(calculation(3)) // globalValue: 98, -97
	fmt.Println(calculation(2)) // globalValue: 97, -97
	calculation = myClosure(6, -2, mainFunc)
	fmt.Println(calculation(3)) // globalValue: 96, 111
	fmt.Println(calculation(7)) // globalValue: 95, 114
}
