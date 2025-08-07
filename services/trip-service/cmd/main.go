package main

import (
	"log"
	"net"
	"os"
	"os/signal"
	"ride-sharing/services/trip-service/internal/infrastructure/events"
	"ride-sharing/services/trip-service/internal/infrastructure/grpc"
	"ride-sharing/services/trip-service/internal/infrastructure/repository"
	"ride-sharing/services/trip-service/internal/service"
	"ride-sharing/shared/db"
	"ride-sharing/shared/env"
	"ride-sharing/shared/messaging"
	"ride-sharing/shared/tracing"
	"syscall"

	"golang.org/x/net/context"
	grpcserver "google.golang.org/grpc"
)

var (
	GrpcAddr = ":9093"
)

func main() {
	tracerCfg := tracing.Config{
		ServiceName:    "trip-service",
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

	// Initilize MongoDB
	mongoClient, err := db.NewMongoClient(ctx, db.NewMongoDefaultConfig())
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	defer mongoClient.Disconnect(ctx)

	mongoDB := db.GetDatabase(mongoClient, db.NewMongoDefaultConfig())

	log.Println(mongoDB.Name())

	rabbitmqURI := env.GetString("RABBITMQ_URI", "amqp://guest:guest@rabbitmq:5672/")
	mongoDBRepo := repository.NewMongoRepository(mongoDB)
	svc := service.NewService(mongoDBRepo)

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

	publisher := events.NewTripEventPublisher(rabbitmq)

	// start driver consumer
	driverConsumer := events.NewDriverConsumer(rabbitmq, svc)
	go func() {
		if err := driverConsumer.Listen(); err != nil {
			log.Fatalf("failed to listen to the message: %v", err)
		}
	}()

	paymentConsumer := events.NewPaymentConsumer(rabbitmq, svc)
	go func() {
		if err := paymentConsumer.Listen(); err != nil {
			log.Fatalf("failed to listen to the message: %v", err)
		}
	}()

	// starting the grpc server
	grpcServer := grpcserver.NewServer(tracing.WithTracingInterceptors()...)
	grpc.NewGRPCHandler(grpcServer, svc, publisher)

	log.Printf("Starting gRPC server Trip service on port %s", lis.Addr())

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
