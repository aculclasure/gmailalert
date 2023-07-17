package main

import (
	"log"
	"os"

	"github.com/aculclasure/gmailalert/internal/ui/cli"
)

func main() {
	// if err := gmailalert.CLI(os.Args[1:]); err != nil {
	// 	log.Fatal(err)
	// }
	err := cli.Run(os.Args[1:])
	if err != nil {
		log.Fatal(err)
	}
}
