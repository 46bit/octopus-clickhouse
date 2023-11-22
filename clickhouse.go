package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

type ChConsumption struct {
	Fuel        string  `json:"fuel"`
	Start       int64   `json:"start"`
	Consumption float64 `json:"consumption"`
}

func apiTimestampToChUnixTimestamp(t string) (int64, error) {
	start, err := time.Parse(time.RFC3339, t)
	if err != nil {
		return 0, fmt.Errorf("unparseable timestamp %q: %w", t, err)
	}
	return start.Unix(), nil
}

func chTableExists() error {
	sql := "SELECT count() FROM {database:Identifier}.{table:Identifier}"
	req, err := basicChRequest(sql, nil)
	if err != nil {
		return fmt.Errorf("unable to build request to check table %q exists: %v", chTable, err)
	}
	if err = fetchChResponse(req); err != nil {
		return fmt.Errorf("unable to query table %q: %v", chTable, err)
	}
	return nil
}

func chInsert(fuel string, apiResults []ApiConsumption) error {
	rows := make([]ChConsumption, len(apiResults))
	for i, apiResult := range apiResults {
		start, err := apiTimestampToChUnixTimestamp(apiResult.IntervalStart)
		if err != nil {
			return err
		}
		rows[i] = ChConsumption{
			Fuel:        fuel,
			Start:       start,
			Consumption: apiResult.Consumption,
		}
	}

	req, err := chInsertRequest(rows)
	if err != nil {
		return err
	}
	return fetchChResponse(req)
}

func chInsertRequest(rows []ChConsumption) (*http.Request, error) {
	sql := "INSERT INTO {database:Identifier}.{table:Identifier} FORMAT JSONEachRow"
	body := bytes.Buffer{}
	e := json.NewEncoder(&body)
	for _, row := range rows {
		if err := e.Encode(row); err != nil {
			return nil, err
		}
	}

	closer := io.NopCloser(&body)
	return basicChRequest(sql, &closer)
}

func basicChRequest(sql string, body *io.ReadCloser) (*http.Request, error) {
	reqUrl, err := url.Parse(chUrl)
	if err != nil {
		return nil, err
	}

	query := url.Values{}
	query.Add("query", sql)
	query.Add("param_database", chDatabase)
	query.Add("param_table", chTable)
	reqUrl.RawQuery = query.Encode()

	headers := http.Header{}
	headers.Add("Authorization", chAuth)
	headers.Add("User-Agent", userAgent)
	for key, value := range chHeaders {
		headers.Add(key, value)
	}

	req := &http.Request{
		Method: "POST",
		URL:    reqUrl,
		Header: headers,
	}
	if body != nil {
		req.Body = *body
	}
	return req, nil
}

func fetchChResponse(req *http.Request) error {
	client := http.Client{
		Timeout: httpTimeout,
		// Don't follow redirects. Avoids being confused by my Cloudflare Access setup.
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	if req.Body != nil {
		defer req.Body.Close()
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status code %d from clickhouse: %q", resp.StatusCode, string(body))
	}
	return nil
}
