package main

import (
	"log"
	"os"

	"github.com/aculclasure/gmailalert"
)

func main() {
	if err := gmailalert.CLI(os.Args[1:]); err != nil {
		log.Fatal(err)
	}
}
