package main

/*
 * Detect the client machine's external IPv4 address
 * -------------------------------------------------
 *
 * Uses http://ifconfig.co/json
 */

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
)

// A subset of the IP address information returned by ifconfig.co.
type ipInfo struct {
	IPAddress string `json:"ip"`
}

// Retrieve the client machine's public IPv4 address.
func getClientPublicIPv4Address() (string, error) {
	response, err := http.Get("http://ifconfig.co/json")
	if err != nil {
		return "", err
	}
	defer response.Body.Close()

	responseBody, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return "", err
	}

	info := &ipInfo{}
	err = json.Unmarshal(responseBody, info)
	if err != nil {
		return "", err
	}

	return info.IPAddress, nil
}
