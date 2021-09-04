package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
)

type ipResponse struct {
	Query string `json:"query"`
}

type cloudFlarePutResponse struct {
	Success bool `json:"success"`
}

type cloudFlareResponse struct {
	Result []cloudFlareDomain `json:"result"`
}

type cloudFlareDomain struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Content string `json:"content"`
}

func sendPushover(ctx context.Context, appToken, userToken, title, message string) error {
	body, err := json.Marshal(map[string]string{
		"token":   appToken,
		"user":    userToken,
		"message": message,
		"title":   title,
	})
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.pushover.net:443/1/messages.json", bytes.NewReader(body))
	if err != nil {
		return err
	}

	req.Header = map[string][]string{
		"Content-type": {"application/json"},
	}

	resp, err := http.DefaultClient.Do(req)
	if err == nil {
		_ = resp.Body.Close()
	}
	return err
}

func readResponse(resp *http.Response, value interface{}) error {
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Error reading response body: %v\n", err)
		return err
	}

	err = json.Unmarshal(bodyBytes, value)
	if err != nil {
		fmt.Printf("Error unmarshaling response body: %v\n", err)
	}

	return nil
}

func makeRequest(ctx context.Context, method, url string, body io.Reader, headers map[string][]string, responseObject interface{}) error {
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		fmt.Printf("Error create %v request for URL %v: %v\n", method, url, err)
		return err
	}
	req.Header = headers

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Printf("Error making %v request for URL %v: %v\n", method, url, err)
		return err
	}

	return readResponse(resp, responseObject)
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	cloudFlareEmail := os.Getenv("CLOUDFLARE_EMAIL")
	cloudFlareAPIKey := os.Getenv("CLOUDFLARE_KEY")
	baseDomain := os.Getenv("DOMAIN_NAME")
	subDomain := os.Getenv("SUBDOMAIN")

	pushoverAppToken := os.Getenv("PUSHOVER_APP_TOKEN")
	pushoverUserToken := os.Getenv("PUSHOVER_USER_TOKEN")

	recordType := "A"
	zoneID := ""
	dnsID := ""

	// Get the zoneID from CloudFlare
	headers := map[string][]string{
		"X-Auth-Key":   {cloudFlareAPIKey},
		"X-Auth-Email": {cloudFlareEmail},
		"Content-Type": {"application/json"},
	}

	respBody := &cloudFlareResponse{}
	if makeRequest(ctx, http.MethodGet, "https://api.cloudflare.com/client/v4/zones/", nil, headers, respBody) != nil {
		return
	}

	for _, item := range respBody.Result {
		if item.Name == baseDomain {
			zoneID = item.ID
			break
		}
	}
	fmt.Printf("Received zone ID: %v\n", zoneID)

	// Find the IP address that CloudFlare has for your (sub) domain based on the recordName and recordType
	if makeRequest(ctx, http.MethodGet,
		fmt.Sprintf("https://api.cloudflare.com/client/v4/zones/%v/dns_records?type=%v&name=%v.%v",
			zoneID, recordType, subDomain, baseDomain,
		),
		nil,
		headers,
		respBody) != nil {
		return
	}

	if len(respBody.Result) == 0 {
		fmt.Printf("Response has no results\n")
		return
	}
	ipFromCF := respBody.Result[0].Content
	dnsID = respBody.Result[0].ID

	fmt.Printf("Received IP address from Cloud Flare: %v\n", ipFromCF)

	// Get your current device IP Address
	ipResponseBody := &ipResponse{}
	if makeRequest(ctx, http.MethodGet, "http://ip-api.com/json", nil, nil, ipResponseBody) != nil {
		return
	}

	actualIP := ipResponseBody.Query
	fmt.Printf("Got actual IP address: %v\n", actualIP)

	if ipFromCF != actualIP {
		// Update with Cloud Flare
		fmt.Println("Updating IP address with Cloud Flare")
		data, err := json.Marshal(map[string]string{
			"type": recordType, "name": subDomain + "." + baseDomain, "content": actualIP,
		})
		if err != nil {
			fmt.Printf("ERROR: %v\n", err)
			return
		}

		putResp := &cloudFlarePutResponse{}
		err = makeRequest(ctx, http.MethodPut,
			fmt.Sprintf("https://api.cloudflare.com/client/v4/zones/%v/dns_records/%v", zoneID, dnsID),
			bytes.NewReader(data),
			headers,
			putResp)

		if pushoverAppToken != "" && pushoverUserToken != "" {
			fmt.Println("Sending Pushover notification")
			if !putResp.Success || err != nil {
				err = sendPushover(ctx, pushoverAppToken, pushoverUserToken, "DNS Update FAILED!", "DNS could not be updated with Cloud Flare. Check it out.")
			} else {
				err = sendPushover(ctx, pushoverAppToken, pushoverUserToken, "DNS Updated", "DNS updated with Cloud Flare successfully!")
			}
			if err != nil {
				fmt.Printf("ERROR: %v\n", err)
			}
		}
	} else {
		fmt.Println("IP addresses are the same, no update needed")
	}
}
