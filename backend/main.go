package main

import (
	"log"

	"couple-mini/backend/bootstrap"
)

func main() {
	if err := bootstrap.Run(); err != nil {
		log.Fatal(err)
	}
}
