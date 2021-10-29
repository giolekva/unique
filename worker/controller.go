package worker

import (
	nrpc "net/rpc"

	"github.com/DataDog/hyperloglog"
	"lekva.me/unique/rpc"
)

type Controller interface {
	Register() error
	ReportProgress(stats rpc.WorkerStatistics, hll *hyperloglog.HyperLogLog, outbound []string) error
	GetDocumentsToProcess() ([]string, error)
}

type direct struct {
	selfName    string
	selfAddress string
	ctrlClient  *nrpc.Client
}

func NewDirectControllerClient(selfName, selfAddress string, ctrlClient *nrpc.Client) Controller {
	return &direct{
		selfName,
		selfAddress,
		ctrlClient,
	}
}

func (c *direct) Register() error {
	req := &rpc.RegisterRequest{
		c.selfName,
		c.selfAddress,
	}
	resp := &rpc.RegisterResponse{}
	return c.ctrlClient.Call("Controller.Register", req, resp)
}

func (c *direct) ReportProgress(stats rpc.WorkerStatistics, hll *hyperloglog.HyperLogLog, outbound []string) error {
	req := &rpc.ReportProgressRequest{
		stats,
		*hll,
		outbound,
	}
	resp := &rpc.ReportProgressResponse{}
	return c.ctrlClient.Call("Controller.ReportProgress", req, resp)
}

func (c *direct) GetDocumentsToProcess() ([]string, error) {
	req := &rpc.GetDocumentsToProcessRequest{}
	resp := &rpc.GetDocumentsToProcessResponse{}
	if err := c.ctrlClient.Call("Controller.GetDocumentsToProcess", req, resp); err != nil {
		return nil, err
	}
	return resp.Addresses, nil
}
