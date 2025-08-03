package main

import (
	"fmt"
	"regexp"
)

func main() {
	test := "7656119800000000a"
	matched, _ := regexp.MatchString(`^\d+$`, test)
	fmt.Printf("Testing '%s' with '^\\d+$': %v\n", test, matched)
	
	test2 := "76561198000000000"
	matched2, _ := regexp.MatchString(`^\d+$`, test2)
	fmt.Printf("Testing '%s' with '^\\d+$': %v\n", test2, matched2)
}
