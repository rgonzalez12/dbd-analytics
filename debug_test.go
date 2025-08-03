package main

import (
	"fmt"
	"strconv"
)

func main() {
	test := "7656119800000000a"
	_, err := strconv.ParseUint(test, 10, 64)
	fmt.Printf("Testing '%s': Error = %v\n", test, err)
	
	test2 := "7656119800000000"
	_, err2 := strconv.ParseUint(test2, 10, 64)
	fmt.Printf("Testing '%s': Error = %v\n", test2, err2)
}
