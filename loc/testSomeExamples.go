package main

import "fmt"

func main() {
	var i interface{} = "Hello World"

	// вариант без паники
	//v, ok := i.(int)
	//if ok {
	//	fmt.Println(v)
	//} else {
	//	fmt.Println("var i is not int")
	//}

	// вариант с паникой
	v := i.(int)
	fmt.Println(v)
}
