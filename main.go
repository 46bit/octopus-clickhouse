package main

import (
	"encoding/base64"
	"encoding/json"
	"log"
	"os"
	"time"
)

const (
	pageSize    = 25000
	httpTimeout = 30 * time.Second
	userAgent   = "github.com/46bit/octopus-clickhouse"

	defaultApiHostname = "https://api.octopus.energy"
	defaultChDatabase  = "default"
	defaultChTable     = "octopus_30m_data"
)

var (
	apiHostname string
	apiUser     string
	apiAuth     string

	chUrl      string
	chUser     string
	chPassword string
	chHeaders  map[string]string
	chAuth     string
	chDatabase string
	chTable    string

	backfillDuration       time.Duration
	gasMprn                string
	gasMeterSerial         string
	electricityMpan        string
	electricityMeterSerial string
)

func init() {
	var err error

	apiHostname = os.Getenv("API_HOSTNAME")
	if apiHostname == "" {
		apiHostname = defaultApiHostname
	}
	apiUser = requireStringEnv("API_USER")
	apiAuth = "Basic " + base64.StdEncoding.EncodeToString([]byte(apiUser+":"))

	chUrl = requireStringEnv("CH_URL")
	chUser = requireStringEnv("CH_USER")
	chPassword = os.Getenv("CH_PASSWORD")
	chHeadersJson := os.Getenv("CH_HEADERS")
	if chHeadersJson != "" {
		if err := json.Unmarshal([]byte(chHeadersJson), &chHeaders); err != nil {
			log.Fatalf("Unable to deserialise CH_HEADERS: %v", err)
		}
	}
	chAuth = "Basic " + base64.StdEncoding.EncodeToString([]byte(chUser+":"+chPassword))
	chDatabase = os.Getenv("CH_DATABASE")
	if chDatabase == "" {
		chDatabase = defaultChDatabase
	}
	chTable = os.Getenv("CH_TABLE")
	if chTable == "" {
		chTable = defaultChTable
	}

	backfillDurationStr := requireStringEnv("BACKFILL_DURATION")
	backfillDuration, err = time.ParseDuration(backfillDurationStr)
	if err != nil {
		log.Fatalf("unable to parse backfill duration %q: %w", backfillDurationStr, err)
	}

	gasMprn = requireStringEnv("GAS_MPRN")
	gasMeterSerial = requireStringEnv("GAS_METER_SERIAL")
	electricityMpan = requireStringEnv("ELECTRICITY_MPAN")
	electricityMeterSerial = requireStringEnv("ELECTRICITY_METER_SERIAL")
}

func main() {
	log.Printf("Attempting to backfill %v of data from %q into %q\n", backfillDuration, apiHostname, chUrl)

	if err := chTableExists(); err != nil {
		log.Fatalf("You may need to apply the schema: %v", err)
	}

	end := time.Now()
	start := end.Add(-backfillDuration)
	log.Printf("Calculated timerange is from %s to %s\n", start.Format(time.RFC3339), end.Format(time.RFC3339))
	log.Println()

	gas(start, end)
	log.Println()
	electricity(start, end)
}

func gas(start, end time.Time) {
	req, err := gasApiRequest(start, end)
	if err != nil {
		log.Fatalf("Unable to build request for gas data: %v", err)
	}
	gasResults, err := fetchApiResponse(req)
	if err != nil {
		log.Fatalf("Unable to fetch gas data: %v", err)
	}
	log.Printf("Fetched %d gas consumption rows\n", len(gasResults))
	if err = chInsert("gas", gasResults); err != nil {
		log.Fatalf("Unable to insert gas consumption into ClickHouse: %v", err)
	}
	log.Printf("Inserted gas consumption into ClickHouse\n")
}

func electricity(start, end time.Time) {
	req, err := electricityApiRequest(start, end)
	if err != nil {
		log.Fatalf("Unable to build request for electricity data: %v", err)
	}
	electricityResults, err := fetchApiResponse(req)
	if err != nil {
		log.Fatalf("Unable to fetch electricity data: %v", err)
	}
	log.Printf("Fetched %d electricity consumption rows\n", len(electricityResults))
	if err = chInsert("electricity", electricityResults); err != nil {
		log.Fatalf("Unable to insert electricity consumption into ClickHouse: %v", err)
	}
	log.Printf("Inserted electricity consumption into ClickHouse\n")
}

func requireStringEnv(name string) string {
	value := os.Getenv(name)
	if value == "" {
		log.Fatalf("%q environment variable must be provided\n", name)
	}
	return value
}
