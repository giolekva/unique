package main

import (
	"flag"
	"fmt"
	"log"
	"net/rpc"
	"time"

	"lekva.me/unique/worker"
)

var name string
var address string
var port int
var numWorkers int
var ctrlAddress string
var numBits uint

func init() {
	flag.StringVar(&name, "name", "worker", "Name of the current worker")
	flag.StringVar(&address, "address", "0.0.0.0", "IP address of the current worker")
	flag.IntVar(&port, "port", 1234, "Port to listen on")
	flag.IntVar(&numWorkers, "num-workers", 1, "Number of concurrent workers to launch")
	flag.StringVar(&ctrlAddress, "controller-address", "", "Controller RPC server address")
	flag.UintVar(&numBits, "num-bits", 1024, "Number of hyperloglog bits to use")
}

func main() {
	flag.Parse()
	p, err := worker.NewProcessor(numWorkers, numBits)
	if err != nil {
		log.Fatal("Error creating processor", err)
	}
	for {
		ctrlClient, err := rpc.DialHTTP("tcp", ctrlAddress)
		if err != nil {
			time.Sleep(3 * time.Second)
			continue
		}
		w := worker.NewWorker(
			worker.NewDirectControllerClient(name, fmt.Sprintf("%s:%d", address, port), ctrlClient),
			p)
		if err := w.Serve(port); err != nil {
			log.Fatal(err)
		}
		break
	}
}
