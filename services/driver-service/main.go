package main

import (
	"log"
	"net"
	"os"
	"os/signal"
	"ride-sharing/shared/env"
	"ride-sharing/shared/messaging"
	"ride-sharing/shared/tracing"
	"syscall"

	"golang.org/x/net/context"
	grpcserver "google.golang.org/grpc"
)

var (
	GrpcAddr = ":9092"
)

func main() {
	tracerCfg := tracing.Config{
		ServiceName:    "driver-service",
		Environment:    env.GetString("ENVIRONMENT", "development"),
		JaegerEndpoint: env.GetString("JAEGER_ENDPOINT", "http://jaeger:14268/api/traces"),
	}

	sh, err := tracing.InitTracer(tracerCfg)
	if err != nil {
		log.Fatalf("Failed to initialize tracer: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer sh(ctx)
	defer cancel()

	rabbitmqURI := env.GetString("RABBITMQ_URI", "amqp://guest:guest@rabbitmq:5672/")

	go func() {
		signalChan := make(chan os.Signal, 1)
		signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)
		<-signalChan
		cancel()
	}()

	lis, err := net.Listen("tcp", GrpcAddr)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	rabbitmq, err := messaging.NewRebbitmq(rabbitmqURI)
	if err != nil {
		log.Fatal(err)
	}
	defer rabbitmq.Close()

	service := NewService()

	// starting the grpc server
	grpcServer := grpcserver.NewServer()
	NewGrpcHandler(grpcServer, service)

	consumer := NewTripConsumer(rabbitmq, service)
	go func() {
		if err := consumer.Listen(); err != nil {
			log.Fatalf("failed to listen to the message: %v", err)
		}
	}()

	log.Printf("Starting gRPC server Driver service on port %s", lis.Addr())

	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			log.Printf("failed to serve: %v", err)
			cancel()
		}
	}()

	// wait for the shutdown signal
	<-ctx.Done()
	log.Printf("Shutting down the gRPC server")
	grpcServer.GracefulStop()
}
