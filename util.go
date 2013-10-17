package main

func assert(that bool, msg string) {
	if !that {
		panic("Assert failed!")
	}
}
