package server

import (
	"fmt"
	"google.golang.org/grpc"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	log "github.com/sirupsen/logrus"
	"golang.org/x/net/context"
	"net/http"
	"github.com/blk-io/crux/utils"
	"net"
	"google.golang.org/grpc/credentials"
)

func (tm *TransactionManager) startRpcServer(port int, ipcPath string, tls bool, certFile, keyFile string) error {
	lis, err := utils.CreateIpcSocket(ipcPath)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := Server{Enclave : tm.Enclave}
	grpcServer := grpc.NewServer()
	RegisterClientServer(grpcServer, &s)
	go func() {
		log.Fatal(grpcServer.Serve(lis))
	}()

	go func() error {
		var err error
		if tls {
			err = tm.startRestServerTLS(port, certFile, keyFile, certFile)
		} else {
			err = tm.startRestServer(port)
		}
		if err != nil {
			log.Fatalf("failed to start gRPC REST server: %s", err)
		}
		return err
	}()

	return err
}

func (tm *TransactionManager) startRestServer(port int) error {
	grpcAddress := fmt.Sprintf("%s:%d", "localhost", port-1)
	lis, err := net.Listen("tcp", grpcAddress)

	s := Server{Enclave : tm.Enclave}
	grpcServer := grpc.NewServer()
	RegisterClientServer(grpcServer, &s)
	go func() {
		log.Fatal(grpcServer.Serve(lis))
	}()

	address := fmt.Sprintf("%s:%d", "localhost", port)
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	mux := runtime.NewServeMux()
	opts := []grpc.DialOption{grpc.WithInsecure()}
	err = RegisterClientHandlerFromEndpoint(ctx, mux, grpcAddress, opts)
	if err != nil {
		return fmt.Errorf("could not register service Ping: %s", err)
	}
	log.Printf("starting HTTP/1.1 REST server on %s", address)
	http.ListenAndServe(address, mux)
	return nil
}

func (tm *TransactionManager) startRestServerTLS(port int, certFile, keyFile, ca string) error {
	freePort, err := GetFreePort()
	if err != nil {
		log.Fatalf("failed to find a free port to start gRPC REST server: %s", err)
	}
	grpcAddress := fmt.Sprintf("%s:%d", "localhost", freePort)
	lis, err := net.Listen("tcp", grpcAddress)
	if err != nil {
		log.Fatalf("failed to start gRPC REST server: %s", err)
	}
	s := Server{Enclave : tm.Enclave}
	creds, err := credentials.NewServerTLSFromFile(certFile, keyFile)
	opts := []grpc.ServerOption{grpc.Creds(creds)}
	if err != nil {
		log.Fatalf("failed to load credentials : %v", err)
	}
	grpcServer := grpc.NewServer(opts...)
	RegisterClientServer(grpcServer, &s)
	go func() {
		log.Fatal(grpcServer.Serve(lis))
	}()

	address := fmt.Sprintf("%s:%d", "localhost", port)
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	mux := runtime.NewServeMux()
	err = RegisterClientHandlerFromEndpoint(ctx, mux, grpcAddress, []grpc.DialOption{grpc.WithTransportCredentials(creds)})
	if err != nil {
		log.Fatalf("could not register service Ping: %s", err)
		return err
	}
	http.ListenAndServeTLS(address, certFile, keyFile, mux)
	log.Printf("started HTTPS REST server on %s", address)
	return nil
}

func GetFreePort() (int, error) {
	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		return 0, err
	}

	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}