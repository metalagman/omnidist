package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "--version" {
		fmt.Println("mytool v0.0.1")
		return
	}
	fmt.Println("mytool - a sample CLI")
}
