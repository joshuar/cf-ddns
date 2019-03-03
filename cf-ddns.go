package main

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"time"

	resty "gopkg.in/resty.v1"

	"github.com/akamensky/argparse"
	"github.com/olebedev/config"
	log "github.com/sirupsen/logrus"
)

const ipv4Checker = "http://ipv4.icanhazip.com"
const ipv6Checker = "http://ipv6.icanhazip.com"
const cfAPI = "https://api.cloudflare.com/client/v4/"

type cfAccount struct {
	email, apiKey, zone, zoneID string
}

type record struct {
	name, ID, recordType, ipAddr string
}

func main() {

	log.SetFormatter(&log.TextFormatter{})

	// parse command-line arguments
	parser := argparse.NewParser("cf-ddns", "Cloudflare Dynamic DNS Client")
	var configFile *string = parser.String("c", "config", &argparse.Options{Required: true, Help: "Path to config file"})
	var logLevel *string = parser.Selector("d", "log-level", []string{"INFO", "DEBUG", "WARN"}, &argparse.Options{Required: false, Help: "Log level"})
	err := parser.Parse(os.Args)
	if err != nil {
		log.Fatal(parser.Usage(err))
	}
	// setup logging level
	switch *logLevel {
	case "INFO":
		log.SetLevel(log.InfoLevel)
	case "DEBUG":
		log.SetLevel(log.DebugLevel)
	case "WARN":
		log.SetLevel(log.WarnLevel)
	default:
		log.SetLevel(log.InfoLevel)
	}
	log.Debug("Parsed command-line arguments")

	// read config file
	file, err := ioutil.ReadFile(*configFile)
	if err != nil {
		log.WithFields(log.Fields{
			"config": *configFile,
			"error":  err,
		}).Fatal("Could not read config file")
	}
	log.Debug("Read config file")
	cfg, err := config.ParseYaml(string(file))
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Fatal("Could not read config file")
	}
	log.Debug("Parsed config file")

	// create account and record details
	account := getAccount(cfg)
	records := getRecords(cfg, account)

	// loop for the configured interval
	// fetch WAN address on every loop
	// update any records as needed
	ticker := time.NewTicker(getInterval(cfg))
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			ipv4 := lookupIPv4()
			ipv6 := lookupIPv6()

			for _, r := range records {
				r.GetRecordDetails(account)
				switch r.recordType {
				case "A":
					if r.ipAddr != ipv4 {
						r.updateRecord(account, ipv4)
					} else {
						log.WithFields(log.Fields{
							"record": r.name,
							"ipAddr": r.ipAddr,
						}).Info("no ipv4 update needed")
					}
				case "AAAA":
					if r.ipAddr != ipv6 {
						r.updateRecord(account, ipv6)
					} else {
						log.WithFields(log.Fields{
							"record": r.name,
							"ipAddr": r.ipAddr,
						}).Info("no ipv6 update needed")
					}
				}

			}
		}
	}
}

func getInterval(c *config.Config) time.Duration {
	i, err := c.String("interval")
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Warn("Unable to parse interval from config, using default of 1h")
		defaultInterval, _ := time.ParseDuration("1h")
		return defaultInterval
	}
	configInterval, err := time.ParseDuration(i)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Warn("Unable to use interval from config, using default of 1h")
		defaultInterval, _ := time.ParseDuration("1h")
		return defaultInterval
	}
	log.WithFields(log.Fields{
		"interval": i,
	}).Debug("Parsed interval from config")
	return configInterval
}

func getAccount(c *config.Config) cfAccount {
	acc, err := c.Get("account")
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Fatal("Unable to read account details from config")
	}
	email, err := acc.String("email")
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Fatal("Unable to read email from config")
	}
	apiKey, err := acc.String("apiKey")
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Fatal("Unable to read apiKey from config")
	}
	zone, err := acc.String("zone")
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Fatal("Unable to read zone from config")
	}
	account := cfAccount{
		email:  email,
		apiKey: apiKey,
		zone:   zone,
	}
	account.GetZoneID()
	log.Debug("Parsed account from config")
	return account
}

func getRecords(c *config.Config, a cfAccount) []record {
	var records []record
	recordsList, err := c.List("records")
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Fatal("Unable to read records to update from config")
	}
	for _, name := range recordsList {
		ARecord := record{
			name:       name.(string),
			recordType: "A",
		}
		ARecord.GetRecordDetails(a)
		records = append(records, ARecord)
		AAAARecord := record{
			name:       name.(string),
			recordType: "AAAA",
		}
		AAAARecord.GetRecordDetails(a)
		records = append(records, AAAARecord)
	}
	log.Debug("Parsed all records from config")
	return records
}

func lookupIPv4() string {
	resp, _ := resty.R().Get(ipv4Checker)
	log.WithFields(log.Fields{
		"ipAddr":       resp.String(),
		"statusCode":   resp.StatusCode(),
		"status":       resp.Status(),
		"responseTime": resp.Time(),
		"receivedAt":   resp.ReceivedAt(),
	}).Debug("Retrieved WAN IPv4 Address")
	return resp.String()
}

func lookupIPv6() string {
	resp, _ := resty.R().Get(ipv6Checker)
	log.WithFields(log.Fields{
		"ipAddr":       resp.String(),
		"statusCode":   resp.StatusCode(),
		"status":       resp.Status(),
		"responseTime": resp.Time(),
		"receivedAt":   resp.ReceivedAt(),
	}).Debug("Retrieved WAN IPv6 Address")
	return resp.String()
}

func (c *cfAccount) GetZoneID() {
	resp, _ := resty.R().
		SetPathParams(map[string]string{
			"zone": c.zone,
		}).
		SetHeader("X-Auth-Email", c.email).
		SetHeader("X-Auth-Key", c.apiKey).
		Get("https://api.cloudflare.com/client/v4/zones?name={zone}")
	log.WithFields(log.Fields{
		"statusCode":   resp.StatusCode(),
		"status":       resp.Status(),
		"responseTime": resp.Time(),
		"receivedAt":   resp.ReceivedAt(),
	}).Debug("Retrieved Zone ID")

	var zr ZoneResponse

	if err := json.Unmarshal(resp.Body(), &zr); err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Fatal("Unable to retrieve zone ID")
	}

	c.zoneID = zr.Result[0].ID
}

func (r *record) GetRecordDetails(c cfAccount) {

	resp, _ := resty.R().
		SetPathParams(map[string]string{
			"zone":       c.zoneID,
			"record":     r.name,
			"recordType": r.recordType,
		}).
		SetHeader("X-Auth-Email", c.email).
		SetHeader("X-Auth-Key", c.apiKey).
		Get("https://api.cloudflare.com/client/v4/zones/{zone}/dns_records?name={record}&type={recordType}")
	log.WithFields(log.Fields{
		"statusCode":   resp.StatusCode(),
		"status":       resp.Status(),
		"responseTime": resp.Time(),
		"receivedAt":   resp.ReceivedAt(),
	}).Debug("Retrieved Record Details")

	var rr RecordResponse

	if err := json.Unmarshal(resp.Body(), &rr); err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Fatal("Unable to read record details")
	}

	r.ID = rr.Result[0].ID
	r.ipAddr = rr.Result[0].Content
}

func (r *record) updateRecord(c cfAccount, ipAddr string) {
	resp, err := resty.R().
		SetPathParams(map[string]string{
			"zone":   c.zoneID,
			"record": r.ID,
		}).
		SetHeader("X-Auth-Email", c.email).
		SetHeader("X-Auth-Key", c.apiKey).
		SetBody(RecordRequest{
			Type:    r.recordType,
			Name:    r.name,
			Content: ipAddr,
		}).
		Put("https://api.cloudflare.com/client/v4/zones/{zone}/dns_records/{record}")
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Warn("Unable to update record")
	} else {
		log.WithFields(log.Fields{
			"name":        r.name,
			"curr_ipAddr": r.ipAddr,
			"new_ipAddr":  ipAddr,
		}).Info("Updated record")
	}
	log.WithFields(log.Fields{
		"statusCode":   resp.StatusCode(),
		"status":       resp.Status(),
		"responseTime": resp.Time(),
		"receivedAt":   resp.ReceivedAt(),
		"recordType":   r.recordType,
		"ID":           r.ID,
		"name":         r.name,
		"ipAddr":       ipAddr,
	}).Debug("Update details")
}
