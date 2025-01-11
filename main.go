package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"bytes"
)

type PorkbunAPIResponse struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
	Domains []struct {
		Domain string `json:"domain"`
	} `json:"domains,omitempty"`
	Records []struct {
		ID string `json:"id"`
	} `json:"records,omitempty"`
}

type PorkbunRecord struct {
	Name   string `json:"name"`
	Type   string `json:"type"`
	Content string `json:"content"`
	TTL    int    `json:"ttl"`
	Prio   *int   `json:"prio,omitempty"`
}

const (
	PorkbunBaseURL = "https://porkbun.com/api/json/v3"
)

func main() {
	if len(os.Args) != 2 {
		log.Fatalf("Usage: %s <IP_ADDRESS>", os.Args[0])
	}
	ipAddress := os.Args[1]

	apiKey := os.Getenv("PORKBUN_API_KEY")
	secretKey := os.Getenv("PORKBUN_SECRET_KEY")

	if apiKey == "" || secretKey == "" {
		log.Fatalf("PORKBUN_API_KEY and PORKBUN_SECRET_KEY environment variables must be set")
	}

	domains, err := getDomains(apiKey, secretKey)
	if err != nil {
		log.Fatalf("Error retrieving domains: %v", err)
	}

	for _, domain := range domains {
		log.Printf("Processing domain: %s", domain)
		if err := updateDomainRecords(domain, ipAddress, apiKey, secretKey); err != nil {
			log.Printf("Error updating domain %s: %v", domain, err)
		}
	}
}

func getDomains(apiKey, secretKey string) ([]string, error) {
	url := fmt.Sprintf("%s/domains/retrieve", PorkbunBaseURL)
	requestBody, _ := json.Marshal(map[string]string{
		"apikey":    apiKey,
		"secretkey": secretKey,
	})

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)
	var response PorkbunAPIResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, err
	}

	if response.Status != "SUCCESS" {
		return nil, fmt.Errorf("API error: %s", response.Message)
	}

	domains := []string{}
	for _, domain := range response.Domains {
		domains = append(domains, domain.Domain)
	}
	return domains, nil
}

func updateDomainRecords(domain, ipAddress, apiKey, secretKey string) error {
	records, err := getDomainRecords(domain, apiKey, secretKey)
	if err != nil {
		return err
	}

	for _, record := range records {
		if err := deleteDomainRecord(domain, record, apiKey, secretKey); err != nil {
			return err
		}
	}

	newRecords := []PorkbunRecord{
		{Name: "@", Type: "A", Content: ipAddress, TTL: 300},
		{Name: "*", Type: "A", Content: ipAddress, TTL: 300},
	}

	for _, record := range newRecords {
		if err := createDomainRecord(domain, record, apiKey, secretKey); err != nil {
			return err
		}
	}

	return nil
}

func getDomainRecords(domain, apiKey, secretKey string) ([]string, error) {
	url := fmt.Sprintf("%s/dns/retrieve/%s", PorkbunBaseURL, domain)
	requestBody, _ := json.Marshal(map[string]string{
		"apikey":    apiKey,
		"secretkey": secretKey,
	})

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)
	var response PorkbunAPIResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, err
	}

	if response.Status != "SUCCESS" {
		return nil, fmt.Errorf("API error: %s", response.Message)
	}

	recordIDs := []string{}
	for _, record := range response.Records {
		recordIDs = append(recordIDs, record.ID)
	}
	return recordIDs, nil
}

func deleteDomainRecord(domain, recordID, apiKey, secretKey string) error {
	url := fmt.Sprintf("%s/dns/delete/%s/%s", PorkbunBaseURL, domain, recordID)
	requestBody, _ := json.Marshal(map[string]string{
		"apikey":    apiKey,
		"secretkey": secretKey,
	})

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(requestBody))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)
	var response PorkbunAPIResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return err
	}

	if response.Status != "SUCCESS" {
		return fmt.Errorf("API error: %s", response.Message)
	}

	return nil
}

func createDomainRecord(domain string, record PorkbunRecord, apiKey, secretKey string) error {
	url := fmt.Sprintf("%s/dns/create/%s", PorkbunBaseURL, domain)
	recordData := map[string]interface{}{
		"apikey":    apiKey,
		"secretkey": secretKey,
		"name":      record.Name,
		"type":      record.Type,
		"content":   record.Content,
		"ttl":       record.TTL,
	}
	requestBody, _ := json.Marshal(recordData)

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(requestBody))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)
	var response PorkbunAPIResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return err
	}

	if response.Status != "SUCCESS" {
		return fmt.Errorf("API error: %s", response.Message)
	}

	return nil
}
