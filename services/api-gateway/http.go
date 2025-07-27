package main

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	grpcclient "ride-sharing/services/api-gateway/grpc_client"
	"ride-sharing/shared/contracts"
)

func handleTripPreview(w http.ResponseWriter, r *http.Request) {
	var reqBody previewTripRequest
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		http.Error(w, "failed to parse JSON data", http.StatusBadRequest)
		return
	}

	if reqBody.UserID == "" {
		http.Error(w, "user id is required", http.StatusBadRequest)
		return
	}

	jsonBody, _ := json.Marshal(reqBody)
	reader := bytes.NewReader(jsonBody)

	tripService, err := grpcclient.NewTripServiceClient()
	if err != nil {
		log.Fatal(err)
	}

	defer tripService.Close()

	// TODO: Call trip service
	res, err := http.Post("http://trip-service:8083/preview", "application/json", reader)
	if err != nil {
		log.Println(err)
		return
	}

	defer res.Body.Close()

	var resBody any
	if err := json.NewDecoder(res.Body).Decode(&resBody); err != nil {
		http.Error(w, "failed to parse JSON data from trip service", http.StatusBadRequest)
		return
	}

	apiRes := contracts.APIResponse{Data: resBody}

	writeJSON(w, http.StatusCreated, apiRes)
}
