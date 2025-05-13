# POC - Sumsub eKYC Integration

This Proof of Concept (POC) demonstrates a basic backend implementation for integrating with Sumsub's electronic Know Your Customer (eKYC) service. It allows users to upload identification documents, initiates the verification process with Sumsub, and handles webhook notifications about the verification status.

## eKYC Flow (Backend and Frontend)

For a detailed breakdown of the eKYC flow involving both the backend and frontend components, please refer to the following Google Docs link:

**[eKYC Flow](https://docs.google.com/document/d/1XN9Y1sMRfrq2Gr_lL1negTMQSpmJKGpcLb6Flm3aFRg/edit?hl=vi&tab=t.0)**

This document outlines the steps involved from the user initiating the verification process on the frontend to the backend communicating with Sumsub and handling the verification results.

## Features Explained

This POC showcases the following key features:

* **Document Upload and Submission:**
    * Provides an HTTP endpoint (`/process-ekyc`) to receive identification documents (e.g., passport, ID card) uploaded by users from a frontend application.
    * Extracts user-provided information like `externalUserId`, `idDocType`, and `country` along with the document file.
    * Uses the Sumsub API to securely upload the document for a specific applicant.

* **Applicant Creation:**
    * Upon receiving a document submission, the backend creates a new applicant in the Sumsub system using the provided `externalUserId`. This ensures that the uploaded document is associated with a unique user.

* **Webhook Handling for Verification Status:**
    * Implements an HTTP endpoint (`/verify`) to receive webhook notifications from Sumsub.
    * **Security:** Verifies the authenticity of the webhook requests using the `X-Payload-Digest` and `X-Payload-Digest-Alg` headers (HMAC-SHA256). It also includes fallback logic to check for `X-Webhook-Signature` if present.
    * Processes the JSON payload of the webhook to extract the `applicantId` and the `reviewResult`.
    * Logs the verification status (`reviewAnswer`) for each applicant.
    * Provides a basic structure for updating your internal system (e.g., a database) based on the verification outcome (e.g., "GREEN" for success, "RED" for failure).

* **Sumsub API Interaction:**
    * Utilizes the Sumsub API for core functionalities:
        * Creating applicants.
        * Uploading documents.
        * (Optional) Retrieving applicant information.
        * (Optional) Generating access tokens for frontend status checks.
    * Includes request signing for secure communication with the Sumsub API using your `AppToken` and `SecretKey`.

* **Configuration via Environment Variables:**
    * Loads sensitive information like Sumsub API credentials (`SUMSUB_APP_TOKEN`, `SUMSUB_SECRET_KEY`), KYC level (`SUMSUB_LEVEL_NAME`), webhook endpoint (`SUMSUB_WEBHOOK_ENDPOINT`), and webhook secret (`SUMSUB_WEBHOOK_SECRET`) from environment variables for security and easy configuration.

* **Basic Logging:**
    * Includes basic logging to the console to track the flow of the eKYC process, API interactions, and webhook events.

## Sumsub API Documentation

For detailed information about the Sumsub API, its endpoints, request/response formats, and the full range of features, please refer to the official Sumsub API documentation:

* **Sumsub API Documentation:** [https://docs.sumsub.com/](https://docs.sumsub.com/)
* **Understanding How ID Verification Works:** [https://docs.sumsub.com/docs/how-id-verification-works](https://docs.sumsub.com/docs/how-id-verification-works)
* **Sumsub API References:** [https://docs.sumsub.com/reference/about-sumsub-api](https://docs.sumsub.com/reference/about-sumsub-api)

Key sections within the documentation that are relevant to this POC include:

* **Getting Started:** Provides an overview of the Sumsub API and the basic integration workflow.
* **Applicants API:** Covers endpoints for creating, retrieving, and managing applicant data. This includes the `/resources/applicants` endpoint used for creating applicants in this POC.
* **Adding Documents:** Explains how to upload identification documents using the `/resources/applicants/{applicantId}/info/idDoc` endpoint, which is utilized in the `addDocumentFromReader` function.
* **Webhooks:** Details the structure and security mechanisms of Sumsub webhook notifications, including the `X-Webhook-Signature`, `X-Payload-Digest`, and `X-Payload-Digest-Alg` headers. This is crucial for understanding how to securely process the notifications received at the `/verify` endpoint.
* **Access Tokens:** Describes how to generate temporary access tokens using the `/resources/accessTokens/sdk` endpoint, which can be used by the frontend for direct status checks (optional in this POC).
* **Authentication and Authorization:** Explains how to authenticate your API requests using the `X-App-Token`, `X-App-Access-Sig`, and `X-App-Access-Ts` headers, as implemented in the `makeSumsubRequest` function.
* **Applicant Statuses:** Provides information on the different stages of the verification process and the meaning of various applicant statuses.

By consulting the official Sumsub API documentation, you can gain a deeper understanding of the underlying mechanisms and explore more advanced features that can be integrated into a production-ready application.

## Getting Started (Example - Assumes Go Environment)

1.  **Clone the Repository:**
    ```bash
    git clone <repository_url>
    cd <repository_directory>
    ```

2.  **Set up Environment Variables:**
    Create a `.env` file in the project root and populate it with your Sumsub API credentials and configuration:
    ```env
    SUMSUB_APP_TOKEN="YOUR_SUMSUB_APP_TOKEN"
    SUMSUB_SECRET_KEY="YOUR_SUMSUB_SECRET_KEY"
    SUMSUB_LEVEL_NAME="YOUR_SUMSUB_KYC_LEVEL" # e.g., basic-kyc-level
    SUMSUB_WEBHOOK_ENDPOINT="YOUR_SERVER_BASE_URL/verify" # Replace with your actual webhook endpoint URL
    SUMSUB_WEBHOOK_SECRET="YOUR_OPTIONAL_WEBHOOK_SECRET" # If you configured a webhook secret in Sumsub
    PORT=8080 # Optional: Change the server port
    ```
    **Important:** Ensure your `SUMSUB_WEBHOOK_ENDPOINT` is accessible by Sumsub's webhook servers.

3.  **Run the Backend Server:**
    ```bash
    go run main.go
    ```
    The server will start and listen for requests on the specified port (default: 8080).

4.  **Integrate with your Frontend:**
    Your frontend application needs to implement a form that allows users to upload their identification document and submit it to the `/process-ekyc` endpoint of your backend server via a POST request with `multipart/form-data`. The form data should include the `content` file, `externalUserId`, `idDocType`, and `country` fields.

5.  **Configure Sumsub Webhook:**
    In your Sumsub dashboard, configure the webhook URL to match the `SUMSUB_WEBHOOK_ENDPOINT` you set in your environment variables (e.g., `https://your-server.com/verify`). If you configured a `Webhook Secret` in Sumsub, ensure the `SUMSUB_WEBHOOK_SECRET` environment variable in your backend matches it.

## Notes

* This is a basic POC and might require further development for production use, including more robust error handling, data validation, and integration with your user management system.
* Consider implementing more detailed logging and monitoring for the eKYC process.
* The frontend integration for document capture and submission is not included in this backend POC.
* Refer to the Sumsub API documentation for the latest best practices and security recommendations.