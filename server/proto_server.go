package server

import (
	"fmt"
	"github.com/blk-io/chimera-api/chimera"
	"github.com/blk-io/crux/utils"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	log "github.com/sirupsen/logrus"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"net"
	"net/http"
)

func (tm *TransactionManager) startRpcServer(port int, grpcJsonPort int, ipcPath string, tls bool, certFile, keyFile string) error {
	lis, err := utils.CreateIpcSocket(ipcPath)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := Server{Enclave: tm.Enclave}
	grpcServer := grpc.NewServer()
	chimera.RegisterClientServer(grpcServer, &s)
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
		if grpcJsonPort != -1 {
			if tls {
				err = tm.startJsonServerTLS(port, grpcJsonPort, certFile, keyFile, certFile)
			} else {
				err = tm.startJsonServer(port, grpcJsonPort)
			}
		}
		if err != nil {
			log.Fatalf("failed to start gRPC REST server: %s", err)
		}
		return err
	}()

	return err
}

func (tm *TransactionManager) startJsonServer(port int, grpcJsonPort int) error {
	address := fmt.Sprintf("%s:%d", "localhost", grpcJsonPort)
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	mux := runtime.NewServeMux()
	opts := []grpc.DialOption{grpc.WithInsecure()}
	err := chimera.RegisterClientHandlerFromEndpoint(ctx, mux, fmt.Sprintf("%s:%d", "localhost", port), opts)
	if err != nil {
		return fmt.Errorf("could not register service: %s", err)
	}
	log.Printf("starting HTTP/1.1 REST server on %s", address)
	err = http.ListenAndServe(address, mux)
	if err != nil {
		return fmt.Errorf("could not listen on %s due to: %s", address, err)
	}
	return nil
}

func (tm *TransactionManager) startRestServer(port int) error {
	grpcAddress := fmt.Sprintf(":%d", port)
	lis, err := net.Listen("tcp", grpcAddress)
	if err != nil {
		panic(err)
	}
	s := Server{Enclave: tm.Enclave}
	grpcServer := grpc.NewServer()
	chimera.RegisterClientServer(grpcServer, &s)
	go func() {
		log.Fatal(grpcServer.Serve(lis))
	}()
	return nil
}

func (tm *TransactionManager) startJsonServerTLS(port int, grpcJsonPort int, certFile, keyFile, ca string) error {
	address := fmt.Sprintf("%s:%d", "localhost", grpcJsonPort)
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	mux := runtime.NewServeMux()
	creds, err := credentials.NewServerTLSFromFile(certFile, keyFile)
	err = chimera.RegisterClientHandlerFromEndpoint(ctx, mux, fmt.Sprintf("%s:%d", "localhost", port), []grpc.DialOption{grpc.WithTransportCredentials(creds)})
	if err != nil {
		log.Fatalf("could not register service Ping: %s", err)
		return err
	}
	http.ListenAndServe(address, mux)
	log.Printf("started HTTPS REST server on %s", address)
	return nil
}

func (tm *TransactionManager) startRestServerTLS(port int, certFile, keyFile, ca string) error {
	grpcAddress := fmt.Sprintf("%s:%d", "localhost", port)
	lis, err := net.Listen("tcp", grpcAddress)
	if err != nil {
		log.Fatalf("failed to start gRPC REST server: %s", err)
	}
	s := Server{Enclave: tm.Enclave}
	creds, err := credentials.NewServerTLSFromFile(certFile, keyFile)
	opts := []grpc.ServerOption{grpc.Creds(creds)}
	if err != nil {
		log.Fatalf("failed to load credentials : %v", err)
	}
	grpcServer := grpc.NewServer(opts...)
	chimera.RegisterClientServer(grpcServer, &s)
	go func() {
		log.Fatal(grpcServer.Serve(lis))
	}()
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
