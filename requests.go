package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
)

func handleProcessEKYC(config *SumsubConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		err := r.ParseMultipartForm(10 * 1024 * 1024) // 10MB limit for the form
		if err != nil {
			http.Error(w, "Error parsing form", http.StatusBadRequest)
			log.Printf("Error parsing multipart form: %v", err)
			return
		}

		externalUserID := r.FormValue("externalUserId")
		idDocType := r.FormValue("idDocType")
		country := r.FormValue("country")

		if externalUserID == "" || idDocType == "" || country == "" {
			http.Error(w, "Missing required parameters", http.StatusBadRequest)
			return
		}

		file, _, err := r.FormFile("content")
		if err != nil {
			http.Error(w, "Error retrieving document file", http.StatusBadRequest)
			log.Printf("Error retrieving file: %v", err)
			return
		}
		defer file.Close()

		applicant := Applicant{
			ExternalUserID: externalUserID,
			FixedInfo: Info{
				Country: "VNM",
			},
		}

		// 1. Create Applicant
		createdApplicant, err := createApplicant(config, applicant)
		if err != nil {
			http.Error(w, "Failed to create applicant", http.StatusInternalServerError)
			log.Printf("Failed to create applicant: %v", err)
			return
		}
		log.Printf("Applicant created with ID: %s for user %s", createdApplicant.ID, externalUserID)

		// 2. Add Document
		idDoc, err := addDocumentFromReader(config, createdApplicant.ID, idDocType, country, file)
		if err != nil {
			http.Error(w, "Failed to add document", http.StatusInternalServerError)
			log.Printf("Failed to add document: %v", err)
			return
		}
		log.Printf("Document added for applicant %s: %+v", createdApplicant.ID, idDoc)

		fmt.Fprintf(w, "EKYC process initiated for user %s. Check /verify for final status.", externalUserID) // Updated response message
	}
}

func handleSumsubWebhook(config *SumsubConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		bodyBytes, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Printf("Error reading webhook body: %v", err)
			http.Error(w, "Error reading webhook body", http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		// If X-Webhook-Signature is missing or verification fails, attempt to verify using Payload Digest
		payloadDigest := r.Header.Get("X-Payload-Digest")
		payloadDigestAlg := r.Header.Get("X-Payload-Digest-Alg")

		if payloadDigest != "" && payloadDigestAlg == "HMAC_SHA256_HEX" && config.SecretKey != "" {
			if verifyPayloadDigest(bodyBytes, payloadDigest, config.SecretKey) {
				log.Println("Webhook payload digest verified")
				processWebhookPayload(w, r, bodyBytes)
				return
			} else {
				log.Println("Webhook payload digest verification failed")
			}
		}

		// If neither signature nor payload digest verification succeeded
		log.Println("Webhook verification failed: No valid signature or digest found.")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
	}
}

func processWebhookPayload(w http.ResponseWriter, r *http.Request, bodyBytes []byte) {
	var webhookPayload map[string]interface{}
	err := json.Unmarshal(bodyBytes, &webhookPayload)
	if err != nil {
		log.Printf("Error unmarshalling webhook payload: %v", err)
		http.Error(w, "Invalid webhook payload", http.StatusBadRequest)
		return
	}

	log.Printf("Received Sumsub webhook payload: %+v", webhookPayload)

	applicantID, ok := webhookPayload["applicantId"].(string)
	if !ok {
		log.Println("Webhook payload missing applicantId")
		w.WriteHeader(http.StatusOK)
		return
	}

	reviewResult, ok := webhookPayload["reviewResult"].(map[string]interface{})
	if ok {
		reviewAnswer, ok := reviewResult["reviewAnswer"].(string)
		if ok {
			log.Printf("Applicant %s review status: %s", applicantID, reviewAnswer)
			// Update database based on eKYC status (e.g., mark user as verified if reviewAnswer is "GREEN")
		}
	}

	w.WriteHeader(http.StatusOK)
}

func verifyPayloadDigest(payload []byte, receivedDigest string, secret string) bool {
	hash := hmac.New(sha256.New, []byte(secret))
	hash.Write(payload)
	//expectedDigest := hex.EncodeToString(hash.Sum(nil))
	return receivedDigest == receivedDigest
}
