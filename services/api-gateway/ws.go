package main

import (
	"log"
	"net/http"
	grpcclient "ride-sharing/services/api-gateway/grpc_client"
	"ride-sharing/shared/contracts"
	"ride-sharing/shared/messaging"
	"ride-sharing/shared/proto/driver"
)

var (
	connManager = messaging.NewConnectionManager()
)

func handleRidersWebSocket(w http.ResponseWriter, r *http.Request, rabbitmq *messaging.Rabbitmq) {
	conn, err := connManager.Upgrade(w, r)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}

	defer conn.Close()

	userID := r.URL.Query().Get("userID")
	if userID == "" {
		log.Println("No user id provided")
		return
	}

	connManager.Add(userID, conn)
	defer connManager.Remove(userID)

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			log.Printf("Error reading message: %v", err)
			break
		}

		log.Printf("Received message: %s", message)
	}
}

func handleDriversWebSocket(w http.ResponseWriter, r *http.Request, rabbitmq *messaging.Rabbitmq) {
	conn, err := connManager.Upgrade(w, r)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}

	defer conn.Close()

	userID := r.URL.Query().Get("userID")
	if userID == "" {
		log.Println("No user id provided")
		return
	}

	packageSlug := r.URL.Query().Get("packageSlug")
	if packageSlug == "" {
		log.Println("No user id provided")
		return
	}

	connManager.Add(userID, conn)
	defer connManager.Remove(userID)

	ctx := r.Context()

	driverService, err := grpcclient.NewDriverServiceClient()
	if err != nil {
		log.Fatal(err)
	}

	// Closing connections
	defer func() {
		connManager.Remove(userID)

		driverService.Client.UnRegisterDriver(ctx, &driver.RegisterDriverRequest{
			DriverID:    userID,
			PackageSlug: packageSlug,
		})

		driverService.Close()

		log.Println("Driver unregistered: ", userID)
	}()

	driverData, err := driverService.Client.RegisterDriver(ctx, &driver.RegisterDriverRequest{
		DriverID:    userID,
		PackageSlug: packageSlug,
	})
	if err != nil {
		log.Printf("Failed to register driver: %v", err)
		return
	}

	if err := connManager.SendMessage(userID, contracts.WSMessage{
		Type: contracts.DriverCmdRegister,
		Data: driverData.Driver,
	}); err != nil {
		log.Printf("Error sending message: %v", err)
		return
	}

	// Initialize queue consumers
	queues := []string{
		messaging.DriverCmdTripRequestQueue,
	}

	for _, q := range queues {
		consumer := messaging.NewQueueConsumer(rabbitmq, connManager, q)

		if err := consumer.Start(); err != nil {
			log.Printf("Failed to start consumer for queue: %s: %v", q, err)
			return
		}
	}

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			log.Printf("Error reading message: %v", err)
			break
		}

		log.Printf("Received message: %s", message)
	}
}
