package server

import (
	"net"
	"fmt"
	"google.golang.org/grpc"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	log "github.com/sirupsen/logrus"
	"golang.org/x/net/context"
	"net/http"
)

func startRPCServer(port int) error {
	lis, err := net.Listen("tcp", fmt.Sprintf("%s:%d", "localhost", port - 1))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := Server{}
	grpcServer := grpc.NewServer()
	RegisterClientServer(grpcServer, &s)
	go func() {
		log.Fatal(grpcServer.Serve(lis))
	}()


	go func() error {
		err := startRESTServer(port)
		if err != nil {
			log.Fatalf("failed to start gRPC REST server: %s", err)
		}
		return err
	}()

	return err
}

func startRESTServer(port int) error {
	grpcAddress := fmt.Sprintf("%s:%d", "localhost", port - 1)
	address := fmt.Sprintf("%s:%d", "localhost", port)
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	mux := runtime.NewServeMux()
	opts := []grpc.DialOption{grpc.WithInsecure()}
	err := RegisterClientHandlerFromEndpoint(ctx, mux, grpcAddress, opts)
	if err != nil {
		return fmt.Errorf("could not register service Ping: %s", err)
	}
	log.Printf("starting HTTP/1.1 REST server on %s", address)
	http.ListenAndServe(address, mux)
	return nil
}
