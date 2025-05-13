package main

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/k0kubun/pp"
	"github.com/pkg/errors"
)

// Constants
const (
	SumsubBaseURL   = "https://api.sumsub.com"
	ContentTypeJSON = "application/json"
)

func main() {
	config, err := loadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	go func() {
		http.HandleFunc("/process-ekyc", handleProcessEKYC(config))
		http.HandleFunc("/verify", handleSumsubWebhook(config))
		port := os.Getenv("PORT")
		if port == "" {
			port = "8080"
		}
		log.Printf("Server started on port %s", port)
		log.Printf("  - Document processing endpoint: /process-ekyc")
		log.Printf("  - Sumsub webhook endpoint: /verify", config.WebhookEndpoint)
		if err := http.ListenAndServe(":"+port, nil); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	// Keep the main goroutine alive
	select {}
}

func generateAccessToken(config *SumsubConfig, userID, levelName string) (*AccessToken, error) {
	accessTokenRequest := AccessTokenRequest{
		UserID:    userID,
		LevelName: levelName,
	}
	body, err := json.Marshal(accessTokenRequest)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal access token request")
	}

	respBody, err := makeSumsubRequest(config, http.MethodPost, "/resources/accessTokens/sdk", ContentTypeJSON, body)
	if err != nil {
		return nil, errors.Wrap(err, "failed to make generate access token request")
	}
	pp.Println("Generate Access Token Response:", string(respBody))
	ioutil.WriteFile("generateAccessToken.json", respBody, 0644)

	var tokenResponse AccessTokenResponse
	err = json.Unmarshal(respBody, &tokenResponse)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal generate access token response")
	}

	return &tokenResponse.AccessToken, nil
}

func createApplicant(config *SumsubConfig, applicant Applicant) (Applicant, error) {
	body, err := json.Marshal(applicant)
	if err != nil {
		return Applicant{}, errors.Wrap(err, "failed to marshal create applicant request")
	}

	path := fmt.Sprintf("/resources/applicants?levelName=%s", config.LevelName)
	respBody, err := makeSumsubRequest(config, http.MethodPost, path, ContentTypeJSON, body)
	if err != nil {
		return Applicant{}, errors.Wrap(err, "failed to make create applicant request")
	}
	pp.Println("Create Applicant Response:", string(respBody))
	ioutil.WriteFile("createApplicant.json", respBody, 0644)

	var createdApplicant Applicant
	err = json.Unmarshal(respBody, &createdApplicant)
	if err != nil {
		return Applicant{}, errors.Wrap(err, "failed to unmarshal create applicant response")
	}

	return createdApplicant, nil
}

func getApplicantInfo(config *SumsubConfig, applicantID string) (Applicant, error) {
	path := fmt.Sprintf("/resources/applicants/%s/one", applicantID)
	respBody, err := makeSumsubRequest(config, http.MethodGet, path, ContentTypeJSON, nil)
	if err != nil {
		return Applicant{}, errors.Wrap(err, "failed to make get applicant info request")
	}
	ioutil.WriteFile("getApplicant.json", respBody, 0644)

	var applicantInfo Applicant
	err = json.Unmarshal(respBody, &applicantInfo)
	if err != nil {
		return Applicant{}, errors.Wrap(err, "failed to unmarshal get applicant info response")
	}
	pp.Println("Get Applicant Info:", applicantInfo)

	return applicantInfo, nil
}

func addDocumentFromReader(config *SumsubConfig, applicantID, idDocType, country string, file io.Reader) (*IdDoc, error) {
	metadata := IdDocMetadata{
		IdDocType: idDocType,
		Country:   country,
	}
	metaJSON, err := json.Marshal(metadata)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal document metadata")
	}

	var b bytes.Buffer
	w := multipart.NewWriter(&b)

	// Create file part
	fw, err := w.CreateFormFile("content", "document") // Use a generic filename
	if err != nil {
		return nil, errors.Wrap(err, "failed to create form file")
	}
	if _, err = io.Copy(fw, file); err != nil {
		return nil, errors.Wrap(err, "failed to copy file content")
	}

	// Create metadata part
	fw, err = w.CreateFormField("metadata")
	if err != nil {
		return nil, errors.Wrap(err, "failed to create metadata form field")
	}
	if _, err = io.Copy(fw, strings.NewReader(string(metaJSON))); err != nil {
		return nil, errors.Wrap(err, "failed to copy metadata")
	}
	w.Close()

	path := fmt.Sprintf("/resources/applicants/%s/info/idDoc", applicantID)
	respBody, err := makeSumsubRequest(config, http.MethodPost, path, w.FormDataContentType(), b.Bytes())
	if err != nil {
		return nil, errors.Wrap(err, "failed to make add document request")
	}

	var docResponse IdDoc
	err = json.Unmarshal(respBody, &docResponse)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal add document response")
	}

	return &docResponse, nil
}

func makeSumsubRequest(config *SumsubConfig, method, path, contentType string, body []byte) ([]byte, error) {
	url := SumsubBaseURL + path
	req, err := http.NewRequest(method, url, bytes.NewBuffer(body))
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create HTTP request for %s %s", method, url)
	}

	ts := fmt.Sprintf("%d", time.Now().Unix())
	signature := signRequest(ts, config.SecretKey, method, path, body)

	req.Header.Set("X-App-Token", config.AppToken)
	req.Header.Set("X-App-Access-Sig", signature)
	req.Header.Set("X-App-Access-Ts", ts)
	req.Header.Set("Accept", ContentTypeJSON)
	req.Header.Set("Content-Type", contentType)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, errors.Wrapf(err, "HTTP request failed for %s %s", method, url)
	}
	defer resp.Body.Close()

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read response body")
	}

	if resp.StatusCode >= http.StatusBadRequest {
		return nil, fmt.Errorf("Sumsub API error: Status %d, Body: %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

func signRequest(timestamp, secret, method, path string, body []byte) string {
	data := []byte(timestamp + method + path)
	if body != nil {
		data = append(data, body...)
	}
	hash := hmac.New(sha256.New, []byte(secret))
	hash.Write(data)
	return hex.EncodeToString(hash.Sum(nil))
}
