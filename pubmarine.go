package main

import (
	"flag"
	"fmt"
	"log"
)

func main() {
	fmt.Println("pubmarine server starting")
	addr := flag.String("addr", "localhost:10209", "http service address")

	flag.Parse()
	log.SetFlags(0)

	s := makeServer()

	fmt.Println("endpoint:", "ws://"+*addr)

	s.init(*addr)

}
