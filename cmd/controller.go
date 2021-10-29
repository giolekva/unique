package main

import (
	"flag"
	"log"

	"github.com/giolekva/unique/controller"
)

var port int
var numBits uint
var startFrom string
var numDocuments int

func init() {
	flag.IntVar(&port, "port", 4321, "Port to listen on")
	flag.UintVar(&numBits, "num-bits", 1024, "Number of hyperloglog bits to use")
	flag.StringVar(&startFrom, "start-from", "", "Web address to start crawling from")
	flag.IntVar(&numDocuments, "num-documents", 100, "Number of documents to process")
}

func main() {
	flag.Parse()
	c, err := controller.NewController(numBits)
	if err != nil {
		log.Fatal("Error creating controller: ", err)
	}
	if err := c.Serve(port, startFrom, numDocuments); err != nil {
		log.Fatal(err)
	}
	log.Printf("!!! Number of unique words: %d\n", c.NumUniques())
}
