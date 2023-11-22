package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

func basicApiRequest(path string, periodFrom, periodTo time.Time) (*http.Request, error) {
	reqUrl, err := url.Parse(path)
	if err != nil {
		return nil, err
	}

	query := url.Values{}
	query.Add("period_from", periodFrom.Format(time.RFC3339))
	query.Add("period_to", periodTo.Format(time.RFC3339))
	query.Add("page_size", fmt.Sprintf("%d", pageSize))
	reqUrl.RawQuery = query.Encode()

	headers := http.Header{}
	headers.Add("Authorization", apiAuth)
	headers.Add("User-Agent", userAgent)

	return &http.Request{
		Method: "GET",
		URL:    reqUrl,
		Header: headers,
	}, nil
}

func gasApiRequest(periodFrom, periodTo time.Time) (*http.Request, error) {
	path := fmt.Sprintf("%s/v1/gas-meter-points/%s/meters/%s/consumption/", apiHostname, gasMprn, gasMeterSerial)
	return basicApiRequest(path, periodFrom, periodTo)
}

func electricityApiRequest(periodFrom, periodTo time.Time) (*http.Request, error) {
	path := fmt.Sprintf("%s/v1/electricity-meter-points/%s/meters/%s/consumption/", apiHostname, electricityMpan, electricityMeterSerial)
	return basicApiRequest(path, periodFrom, periodTo)
}

type ApiConsumption struct {
	IntervalStart string  `json:"interval_start"`
	Consumption   float64 `json:"consumption"`
}

type ApiBody struct {
	Count uint32 `json:"count"`
	// FIXME: Support pagination?
	// Or error if the time period exceeds the time period that pageSize * 30m covers
	Next     *string          `json:"next"`
	Previous *string          `json:"previous"`
	Results  []ApiConsumption `json:"results"`
}

func fetchApiResponse(req *http.Request) ([]ApiConsumption, error) {
	client := http.Client{Timeout: httpTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status code %d from octopus api: %q", resp.StatusCode, string(body))
	}

	var body ApiBody
	d := json.NewDecoder(resp.Body)
	if err = d.Decode(&body); err != nil {
		return nil, err
	}
	if d.More() {
		return nil, fmt.Errorf("extraneous data after JSON response")
	}

	return body.Results, nil
}
