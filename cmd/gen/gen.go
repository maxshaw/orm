package main

import (
	"log"

	"github.com/maxshaw/orm/gen"
)

func main() {
	if err := gen.Gen(); err != nil {
		log.Fatal(err)
	}
}
