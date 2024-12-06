package feishu

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type AccessTokenRequest struct {
	AppID     string `json:"app_id"`
	AppSecret string `json:"app_secret"`
}

type AccessTokenResponse struct {
	Code              int    `json:"code"`
	Message           string `json:"msg"`
	TenantAccessToken string `json:"tenant_access_token"`
}

func GetTenantAccessToken() (string, error) {
	url := "https://open.feishu.cn/open-apis/auth/v3/tenant_access_token/internal"

	// Create the request body as JSON
	requestBody := AccessTokenRequest{
		AppID:     "xxxxx",
		AppSecret: "xxxxxx",
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		fmt.Println("JSON encoding error:", err)
		return "", err
	}

	// Create a new request with the JSON data
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Println("Creating request failed:", err)
		return "", err
	}

	req.Header.Set("Content-Type", "application/json; charset=utf-8")

	// Create a new HTTP client
	client := &http.Client{}

	// Send the request
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Sending request failed:", err)
		return "", err
	}
	defer resp.Body.Close()

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Reading response failed:", err)
		return "", err
	}

	// Parse the response JSON
	var accessTokenResponse AccessTokenResponse
	err = json.Unmarshal(body, &accessTokenResponse)
	if err != nil {
		fmt.Println("Decoding JSON failed:", err)
		return "", err
	}

	return accessTokenResponse.TenantAccessToken, nil
}
