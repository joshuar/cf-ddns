/*
Copyright © 2021 Joshua Rich <joshua.rich@gmail.com>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package cloudflare

import (
	"encoding/json"
	"sync"

	"github.com/joshuar/cf-ddns/internal/iplookup"
	log "github.com/sirupsen/logrus"

	"github.com/go-resty/resty/v2"
	"github.com/spf13/viper"
)

const cloudflareAPI = "https://api.cloudflare.com/client/v4/"

type cloudflareAccount struct {
	email, apiKey, zone, zoneID string
	records                     map[string]*dnsRecord
}

type dnsRecord struct {
	name, id, recordType, IpAddr string
}

// GetAccountDetails reads the Cloudflare account details from the config file
// and populates a struct for reference
func GetAccountDetails() *cloudflareAccount {
	account := &cloudflareAccount{
		email:   viper.GetString("account.email"),
		apiKey:  viper.GetString("account.apiKey"),
		zone:    viper.GetString("account.zone"),
		records: make(map[string]*dnsRecord),
	}
	account.getZoneID()
	account.getCurrentRecords()
	return account
}

func (c *cloudflareAccount) getZoneID() {
	client := resty.New().SetLogger(&log.Logger{})
	resp, err := client.R().
		SetPathParams(map[string]string{
			"zone": c.zone,
		}).
		SetHeader("X-Auth-Email", c.email).
		SetHeader("X-Auth-Key", c.apiKey).
		Get(cloudflareAPI + "zones?name={zone}")
	if err != nil {
		log.Fatalf("Unable to send request %s for ID for zone: %v", resp.Request.URL, err)
	}
	var zr ZoneResponse

	if err := json.Unmarshal(resp.Body(), &zr); err != nil {
		log.Fatalf("Unable to parse zone ID response: %v", err)
	} else {
		if !zr.Success {
			log.Fatalf("Failed to get zone ID: request returned error code %d with message '%s'", zr.Errors[0].Code, zr.Errors[0].Message)
		}
		if zr.ResultInfo.Count == 0 {
			log.Fatalf("No zone ID returned for zone: %s.  Check config and try again.", c.zone)
		}
		c.zoneID = zr.Result[0].ID
	}
}

func (c *cloudflareAccount) getCurrentRecords() {
	var wg sync.WaitGroup
	log.Debug("Fetching current records from Cloudflare...")
	for _, r := range getRecordsFromConfig() {
		for _, rt := range []string{"A", "AAAA"} {
			wg.Add(1)
			r := &dnsRecord{
				name:       r,
				recordType: rt,
			}
			go func(rt string) {
				defer wg.Done()
				c.records[rt] = c.getDNSRecord(r)
			}(rt)
		}
		wg.Wait()
	}
}

func (c *cloudflareAccount) getDNSRecord(r *dnsRecord) *dnsRecord {
	client := resty.New().SetLogger(&log.Logger{})
	client.SetRetryCount(3)

	resp, err := client.R().
		SetPathParams(map[string]string{
			"zone":       c.zoneID,
			"record":     r.name,
			"recordType": r.recordType,
		}).
		SetHeader("X-Auth-Email", c.email).
		SetHeader("X-Auth-Key", c.apiKey).
		Get(cloudflareAPI + "zones/{zone}/dns_records?name={record}&type={recordType}")
	if err != nil {
		log.Warnf("Unable to send request %s for dns record type: %v", resp.Request.URL, err)
		return nil
	}

	var rr RecordResponse
	if err := json.Unmarshal(resp.Body(), &rr); err != nil {
		log.Warnf("Unable to parse dns record response for %s (type %s): %v", r.name, r.recordType, err)
		return nil
	}
	if !rr.Success {
		log.Warnf("Failed to get dns record: request returned error code %d with message '%s'", rr.Errors[0].Code, rr.Errors[0].Message)
		return nil
	} else {
		if rr.ResultInfo.Count > 0 {
			r.id = rr.Result[0].ID
			r.IpAddr = rr.Result[0].Content
			log.Debugf("Found %s record for %s with address %s", r.recordType, r.name, r.IpAddr)
			return r
		} else {
			return nil
		}
	}
}

func (c *cloudflareAccount) CheckForUpdates() {
	addr := iplookup.LookupExternalIP()
	for t, r := range c.records {
		if r == nil {
			break
		}
		switch t {
		case "A":
			if r.IpAddr != addr.Ipv4 {
				log.Debugf("Record %s (type %s) needs updating.  Previously %s, now %s", r.name, t, r.IpAddr, addr.Ipv4)
				r.IpAddr = addr.Ipv4
				c.setDNSRecord(r)
			}
		case "AAAA":
			if r.IpAddr != addr.Ipv6 {
				log.Debugf("Record %s (type %s) needs updating.  Previously %s, now %s", r.name, t, r.IpAddr, addr.Ipv6)
				r.IpAddr = addr.Ipv6
				c.setDNSRecord(r)
			}
		}
	}
}

func (c *cloudflareAccount) setDNSRecord(record *dnsRecord) {
	client := resty.New().SetLogger(&log.Logger{})
	client.SetRetryCount(3)

	resp, err := client.R().
		SetPathParams(map[string]string{
			"zone":   c.zoneID,
			"record": record.id,
		}).
		SetHeader("X-Auth-Email", c.email).
		SetHeader("X-Auth-Key", c.apiKey).
		SetBody(map[string]interface{}{
			"type":    record.recordType,
			"name":    record.name,
			"content": record.IpAddr,
			"ttl":     1,
		}).
		Put(cloudflareAPI + "zones/{zone}/dns_records/{record}")
	if err != nil {
		log.Warnf("Unable to send request %s for dns record type: %v", resp.Request.URL, err)
	}
	if resp.StatusCode() == 200 {
		log.Infof("Updated %s record for %s to %s", record.recordType, record.name, record.IpAddr)
	} else {
		log.Warnf("Unable to update %s record for %s: %v", record.recordType, record.name, resp.Status())
	}
}

func getRecordsFromConfig() []string {
	records := viper.GetStringSlice("records")
	if records == nil {
		log.Fatal("No records to check found in config? Nothing to do.")
	}
	return records
}
