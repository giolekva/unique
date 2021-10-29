package controller

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	nrpc "net/rpc"
	"sync"
	"time"

	"github.com/DataDog/hyperloglog"
	"lekva.me/unique/rpc"
)

const maxBatchSize = 100
const numAddressesToProcess = 100

func minInt(a, b int) int {
	if a < b {
		return a
	} else {
		return b
	}
}

type server struct {
	hll                   *hyperloglog.HyperLogLog
	lock                  sync.Locker
	numDocumentsProcessed int
	addresses             []string
	allAddresses          map[string]bool
	workers               map[string]string
}

func newServer(numBits uint) (*server, error) {
	hll, err := hyperloglog.New(numBits)
	if err != nil {
		return nil, err
	}
	return &server{
		hll:                   hll,
		lock:                  &sync.Mutex{},
		numDocumentsProcessed: 0,
		addresses:             make([]string, 0),
		allAddresses:          make(map[string]bool),
		workers:               make(map[string]string),
	}, nil
}

func (s *server) getNumDocumentsProcessed() int {
	s.lock.Lock()
	defer s.lock.Unlock()
	return s.numDocumentsProcessed
}

func (s *server) Register(req *rpc.RegisterRequest, resp *rpc.RegisterResponse) error {
	s.workers[req.Name] = req.Address
	return nil
}

func (s *server) GetDocumentsToProcess(req *rpc.GetDocumentsToProcessRequest, resp *rpc.GetDocumentsToProcessResponse) error {
	s.lock.Lock()
	defer s.lock.Unlock()
	cut := minInt(len(s.addresses), maxBatchSize)
	resp.Addresses = s.addresses[:cut]
	s.addresses = s.addresses[cut:]
	return nil
}

func (s *server) ReportProgress(req *rpc.ReportProgressRequest, resp *rpc.ReportProgressResponse) error {
	s.lock.Lock()
	defer s.lock.Unlock()
	if err := s.hll.Merge(&req.HLL); err != nil {
		return err
	}
	s.numDocumentsProcessed += req.Stats.Processed
	for _, addr := range req.Outbound {
		if _, ok := s.allAddresses[addr]; !ok {
			s.addresses = append(s.addresses, addr)
			s.allAddresses[addr] = true
		}
	}
	return nil
}

type Controller struct {
	s *server
}

func NewController(numBits uint) (*Controller, error) {
	server, err := newServer(numBits)
	if err != nil {
		return nil, err
	}
	return &Controller{
		s: server,
	}, nil
}

func (c *Controller) Serve(port int, startFrom string, numDocuments int) error {
	nrpc.RegisterName("Controller", c.s)
	nrpc.HandleHTTP()
	l, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", port))
	if err != nil {
		return err
	}
	fmt.Println(startFrom)
	c.s.addresses = append(c.s.addresses, startFrom)
	var srv http.Server
	go srv.Serve(l)
	for {
		time.Sleep(1 * time.Second)
		numProcessed := c.s.getNumDocumentsProcessed()
		log.Printf("Number of documents processed: %d\n", numProcessed)
		if numProcessed < numDocuments {
			continue
		}
		var done sync.WaitGroup
		for name, address := range c.s.workers {
			done.Add(1)
			go func(name, address string) {
				log.Printf("Shutting down worker: %s %s", name, address)
				worker, err := nrpc.DialHTTP("tcp", address)
				if err != nil {
					log.Fatal("dialing:", err)
				}
				worker.Call("Worker.Shutdown", &rpc.ShutdownRequest{}, &rpc.ShutdownResponse{})
				log.Printf("Done")
				done.Done()
			}(name, address)
		}
		done.Wait()
		srv.Shutdown(context.TODO())
		break
	}
	return nil
}
