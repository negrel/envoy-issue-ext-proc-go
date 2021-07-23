package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"regexp"
	"sync"
	"syscall"

	envoy_config_core_v3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	extproc "github.com/envoyproxy/go-control-plane/envoy/service/ext_proc/v3alpha"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	listenFlag  string
	srcPathFlag string
	dstPathFlag string
)

var pathRegex *regexp.Regexp

func init() {
	flag.StringVar(&listenFlag, "listen", ":18080", "listen to <address>:<port>")
	flag.StringVar(&srcPathFlag, "src-path", "", "a regex/string that match request path to change")
	flag.StringVar(&dstPathFlag, "dst-path", "/", "the new path that replace matched source paths")
	flag.Parse()

	if srcPathFlag == "" {
		fmt.Fprintln(os.Stderr, "src-path flag must have a non empty value")
		flag.PrintDefaults()
		os.Exit(1)
	}

	pathRegex = regexp.MustCompile(srcPathFlag)

	log.SetOutput(os.Stderr)
	log.SetPrefix("[ENVOY_EXT_PROC_GO] ")
	log.SetFlags(log.LstdFlags | log.Lmicroseconds | log.Lmsgprefix)
}

// ExternalProcessorServerFunc is a function type that implements the extproc.ExternalProcessorServer interface.
type ExternalProcessorServerFunc func(extproc.ExternalProcessor_ProcessServer) error

// Process implements the extproc.ExternalProcessorServer interface.
func (epsf ExternalProcessorServerFunc) Process(arg extproc.ExternalProcessor_ProcessServer) error {
	return epsf(arg)
}

func main() {
	listener, err := net.Listen("tcp", listenFlag)
	if err != nil {
		log.Fatal(err)
	}

	grpcSrv := grpc.NewServer()
	extproc.RegisterExternalProcessorServer(grpcSrv, ExternalProcessorServerFunc(process))

	var wg sync.WaitGroup
	wg.Add(1)

	go onSignal(func(signal os.Signal) {
		log.Printf("WARNING: %v signal caught, stopping the server...", signal)
		grpcSrv.GracefulStop()
		log.Println("Server stopped.")
		wg.Done()
	}, syscall.SIGINT, syscall.SIGSTOP)

	log.Println("Starting gRPC server on", listenFlag)
	err = grpcSrv.Serve(listener)
	if err != nil {
		log.Fatal(err)
	}

	wg.Wait()
}

func onSignal(callback func(os.Signal), signals ...os.Signal) {
	stopSig := make(chan os.Signal)
	signal.Notify(stopSig, signals...)

	callback(<-stopSig)
}

func process(procSrv extproc.ExternalProcessor_ProcessServer) error {
	log.Println("processing request...")

	ctx := procSrv.Context()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			req, err := procSrv.Recv()
			if err == io.EOF {
				return nil
			} else if err != nil {
				log.Println("an error occured while processing the requets:", err)
				return status.Errorf(codes.Unknown, "cannot receive stream request: %v", err)
			}

			h, ok := req.Request.(*extproc.ProcessingRequest_RequestHeaders)
			if !ok {
				log.Println("WARNING: Unknown request type", req)
				return status.Errorf(codes.Unknown, "unknown request type")
			}

			processRequestHeaders(h, procSrv)
			log.Println("request processed.")
		}
	}
}

func processRequestHeaders(h *extproc.ProcessingRequest_RequestHeaders, procSrv extproc.ExternalProcessor_ProcessServer) {
	headers := h.RequestHeaders.Headers.Headers

	for _, header := range headers {
		if header.Key == ":path" {
			processPath(header, procSrv)
			return
		}
	}
}

func processPath(pathHeader *envoy_config_core_v3.HeaderValue, procSrv extproc.ExternalProcessor_ProcessServer) {
	if pathRegex.Match([]byte(pathHeader.Value)) {
		procSrv.Send(
			&extproc.ProcessingResponse{
				Response: &extproc.ProcessingResponse_RequestHeaders{
					RequestHeaders: &extproc.HeadersResponse{
						Response: &extproc.CommonResponse{
							HeaderMutation: &extproc.HeaderMutation{
								SetHeaders: []*envoy_config_core_v3.HeaderValueOption{
									{
										Header: &envoy_config_core_v3.HeaderValue{
											Key:   ":path",
											Value: dstPathFlag,
										},
									},
								},
							},
						},
					},
				},
			},
		)
		return
	}

	procSrv.Send(&extproc.ProcessingResponse{})
}
