package main

import (
	"fmt"
	"os"

	"jaytaylor.com/archive.is"
)

func main() {
	s, err := archiveis.Capture(os.Args[1])
	if err != nil {
		panic(err)
	}
	fmt.Println(s)
}
