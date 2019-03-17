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
		logError(err, "Could not read config file", "fatal")
	}
	log.Debug("Read config file")
	cfg, err := config.ParseYaml(string(file))
	if err != nil {
		logError(err, "Could not read config file", "fatal")
	}
	log.Debug("Parsed config file")

	// create account and record details
	account := getAccount(cfg)
	records := getRecords(cfg, account)

	checkAndUpdate(account, records)

	// loop for the configured interval
	// fetch WAN address on every loop
	// update any records as needed
	ticker := time.NewTicker(getInterval(cfg))
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			checkAndUpdate(account, records)
		}
	}
}

func checkAndUpdate(account *cfAccount, recordsArray []record) {
	ipv4 := lookupExternalIP("4")
	ipv6 := lookupExternalIP("6")

	for _, r := range recordsArray {
		r.GetRecordDetails(account)
		switch r.recordType {
		case "A":
			if ipv4 == "" && r.ipAddr != "" {
				r.deleteRecord(account)
				break
			} else if ipv4 == "" && r.ipAddr == "" {
				break
			}
			if ipv4 != "" && r.ipAddr == "" {
				r.addRecord(account, ipv4)
			} else if r.ipAddr != ipv4 {
				r.updateRecord(account, ipv4)
			} else {
				logRecord(r.name, r.recordType, r.ipAddr, "No IPv4 update needed")
			}
		case "AAAA":
			if ipv6 == "" && r.ipAddr != "" {
				r.deleteRecord(account)
				break
			} else if ipv6 == "" && r.ipAddr == "" {
				break
			}
			if ipv6 != "" && r.ipAddr == "" {
				r.addRecord(account, ipv6)
			} else if r.ipAddr != ipv6 {
				r.updateRecord(account, ipv6)
			} else {
				logRecord(r.name, r.recordType, r.ipAddr, "No IPv6 update needed")
			}
		}
	}
}

func getInterval(c *config.Config) time.Duration {
	i, err := c.String("interval")
	if err != nil {
		logError(err, "Unable to parse interval from config, using default of 1h", "warn")
		defaultInterval, _ := time.ParseDuration("1h")
		return defaultInterval
	}
	configInterval, err := time.ParseDuration(i)
	if err != nil {
		logError(err, "Couldn't understand interval specified, using default of 1h", "warn")
		defaultInterval, _ := time.ParseDuration("1h")
		return defaultInterval
	}
	log.WithFields(log.Fields{
		"interval": i,
	}).Debug("Parsed interval from config")
	return configInterval
}

func getAccount(c *config.Config) *cfAccount {
	acc, err := c.Get("account")
	if err != nil {
		logError(err, "Unable to read account details from config", "fatal")
	}
	email, err := acc.String("email")
	if err != nil {
		logError(err, "Unable to read email from config", "fatal")
	}
	apiKey, err := acc.String("apiKey")
	if err != nil {
		logError(err, "Unable to read apiKey from config", "fatal")
	}
	zone, err := acc.String("zone")
	if err != nil {
		logError(err, "Unable to read zone from config", "fatal")
	}
	account := &cfAccount{
		email:  email,
		apiKey: apiKey,
		zone:   zone,
	}
	account.GetZoneID()
	log.Debug("Parsed account from config")
	return account
}

func getRecords(c *config.Config, a *cfAccount) []record {
	var records []record
	recordsList, err := c.List("records")
	if err != nil {
		logError(err, "Unable to read records to update from config", "fatal")
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

func lookupExternalIP(version string) string {
	var resp *resty.Response
	var err error
	switch version {
	case "4":
		resp, err = resty.R().Get(ipv4Checker)
	case "6":
		resp, err = resty.R().Get(ipv6Checker)
	}
	logTimings(resp, "External IP Retrieval Timings")
	if err != nil || resp.StatusCode() != 200 {
		logError(err, "Unable to retrieve external IP address", "warn")
		return ""
	}
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

	logTimings(resp, "Zone ID Retrieval Timings")

	var zr ZoneResponse

	if err := json.Unmarshal(resp.Body(), &zr); err != nil {
		logError(err, "Unable to retrieve zone ID", "warn")
	}

	c.zoneID = zr.Result[0].ID
}

func (r *record) GetRecordDetails(c *cfAccount) {

	resp, _ := resty.R().
		SetPathParams(map[string]string{
			"zone":       c.zoneID,
			"record":     r.name,
			"recordType": r.recordType,
		}).
		SetHeader("X-Auth-Email", c.email).
		SetHeader("X-Auth-Key", c.apiKey).
		Get("https://api.cloudflare.com/client/v4/zones/{zone}/dns_records?name={record}&type={recordType}")

	logTimings(resp, "Record Retrieval Timings")

	var rr RecordResponse

	if err := json.Unmarshal(resp.Body(), &rr); err != nil {
		logError(err, "Unable to read record details", "warn")
	}

	if rr.ResultInfo.Count > 0 {
		r.ID = rr.Result[0].ID
		r.ipAddr = rr.Result[0].Content
	} else {
		r.ID = ""
		r.ipAddr = ""
	}
}

func (r *record) updateRecord(c *cfAccount, ipAddr string) {
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
	if err != nil || resp.StatusCode() != 200 {
		logError(err, "Unable to update record", "error")
	} else {
		log.WithFields(log.Fields{
			"name":        r.name,
			"type":        r.recordType,
			"curr_ipAddr": r.ipAddr,
			"new_ipAddr":  ipAddr,
		}).Info("Updated record")
		logTimings(resp, "Update Record Timings")
	}
}

func (r *record) addRecord(c *cfAccount, ipAddr string) {
	resp, err := resty.R().
		SetPathParams(map[string]string{
			"zone": c.zoneID,
		}).
		SetHeader("X-Auth-Email", c.email).
		SetHeader("X-Auth-Key", c.apiKey).
		SetBody(RecordRequest{
			Type:    r.recordType,
			Name:    r.name,
			Content: ipAddr,
		}).
		Post("https://api.cloudflare.com/client/v4/zones/{zone}/dns_records")
	if err != nil || resp.StatusCode() != 200 {
		logError(err, "Unable to add record", "error")
	} else {
		logRecord(r.name, r.recordType, r.ipAddr, "Added Record")
		logTimings(resp, "Add Record Timings")
	}
}

func (r *record) deleteRecord(c *cfAccount) {
	resp, err := resty.R().
		SetPathParams(map[string]string{
			"zone":   c.zoneID,
			"record": r.ID,
		}).
		SetHeader("X-Auth-Email", c.email).
		SetHeader("X-Auth-Key", c.apiKey).
		Delete("https://api.cloudflare.com/client/v4/zones/{zone}/dns_records/{record}")
	if err != nil || resp.StatusCode() != 200 {
		logError(err, "Unable to delete record", "error")
	} else {
		logRecord(r.name, r.recordType, r.ipAddr, "Deleted Record")
		logTimings(resp, "Delete Record Timings")
	}
}
