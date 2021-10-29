package rpc

import "github.com/DataDog/hyperloglog"

type RegisterRequest struct {
	Name    string
	Address string
}

type RegisterResponse struct {
}

type WorkerStatistics struct {
	Processed int
	InQueue   int
}

type GetDocumentsToProcessRequest struct {
}

type GetDocumentsToProcessResponse struct {
	Addresses []string
	Shutdown  bool
}

type ReportProgressRequest struct {
	Stats    WorkerStatistics
	HLL      hyperloglog.HyperLogLog
	Outbound []string
}

type ReportProgressResponse struct {
}

type ShutdownRequest struct {
}

type ShutdownResponse struct {
}
