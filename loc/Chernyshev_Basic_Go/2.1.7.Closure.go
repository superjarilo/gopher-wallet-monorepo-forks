package main

import "fmt"

type microwaveCreator func(dish string, mode int) string

var microwaveIndex int = 0

func factory(microwaveName string, power int) microwaveCreator {
	microwaveIndex++
	model := fmt.Sprintf("%s–RU–001%d", microwaveName, microwaveIndex)
	return func(dish string, mode int) (info string) {
		info = fmt.Sprintf("Микроволновка %s мощностью %d w Вт,",
			model, power)
		info += fmt.Sprintf("греет блюдо %s в режиме %d", dish, mode)
		return
	}
}
func main() {
	microwave := factory("Scarlet", 800)
	fmt.Println(microwave("Суп", 3))
	fmt.Println(microwave("Пюре", 2))
	newMicrowave := factory("LG", 1200)
	fmt.Println(newMicrowave("Плов", 4))
}
