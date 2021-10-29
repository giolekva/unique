package worker

import (
	"context"
	"errors"
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

type shutdownListener struct {
	cond *sync.Cond
}

func (l *shutdownListener) Shutdown(req *rpc.ShutdownRequest, resp *rpc.ShutdownResponse) error {
	l.cond.Broadcast()
	return nil
}

type Worker struct {
	c Controller
	p *Processor
}

func NewWorker(c Controller, p *Processor) *Worker {
	return &Worker{c, p}
}

func (w *Worker) Serve(port int) error {
	if err := w.c.Register(); err != nil {
		return err
	}
	go w.periodicallyGetDocumentsFromController()
	go w.p.Start(w.reportToController)
	var lock sync.Mutex
	lock.Lock()
	defer lock.Unlock()
	cond := sync.NewCond(&lock)
	sl := &shutdownListener{cond}
	nrpc.RegisterName("Worker", sl)
	nrpc.HandleHTTP()
	l, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", port))
	if err != nil {
		return err
	}
	var srv http.Server
	go func() {
		cond.Wait()
		srv.Shutdown(context.TODO())

	}()
	if err := srv.Serve(l); err != nil {
		if !errors.Is(err, http.ErrServerClosed) {
			return err
		}
	}
	return nil
}

func (w *Worker) reportToController(stats rpc.WorkerStatistics, hll *hyperloglog.HyperLogLog, outbound []string) {
	for {
		if err := w.c.ReportProgress(stats, hll, outbound); err != nil {
			fmt.Println(err.Error())
			time.Sleep(5 * time.Second)
		} else {
			break
		}
	}
}

func (w *Worker) periodicallyGetDocumentsFromController() {
	for {
		if addresses, err := w.c.GetDocumentsToProcess(); err != nil {
			log.Printf("Failed to retrieve document addresses from controller: %s\n", err.Error())
		} else {
			for _, addr := range addresses {
				w.p.Add(addr)
			}
		}
		time.Sleep(2 * time.Second)
	}
}
