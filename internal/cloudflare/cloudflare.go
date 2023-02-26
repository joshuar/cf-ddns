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
	"context"

	"github.com/joshuar/cf-ddns/internal/iplookup"
	log "github.com/sirupsen/logrus"

	cf "github.com/cloudflare/cloudflare-go"
	"github.com/spf13/viper"
)

type cloudflare struct {
	api    *cf.API
	zoneID string
	record map[string]*dnsRecord
}

type dnsRecord struct {
	name, id, recordType, IpAddr string
}

func NewCloudflare() *cloudflare {
	api, err := cf.New(
		viper.GetString("account.apiKey"),
		viper.GetString("account.email"))
	if err != nil {
		log.Fatalf("Unable to establish Cloudflare API connection: %v", err)
	}

	id, err := api.ZoneIDByName(viper.GetString("account.zone"))
	if err != nil {
		log.Fatalf("Unable to retrieve zone details from Cloudflare API: %v", err)
	}

	ctx := context.Background()
	record := make(map[string]*dnsRecord)

	for _, r := range getRecordsFromConfig() {
		log.Debugf("Getting DNS record(s) for %s", r)
		recs, _, err := api.ListDNSRecords(
			ctx,
			cf.ZoneIdentifier(id),
			cf.ListDNSRecordsParams{Name: r})
		if err != nil {
			log.Fatalf("Unable to retrieve record details for %s from Cloudflare API: %v", r, err)
		}
		if recs != nil {
			for _, r := range recs {
				record[r.ID] = &dnsRecord{
					name:       r.Name,
					IpAddr:     r.Content,
					recordType: r.Type,
				}
			}
		} else {
			log.Warnf("%s has no matching DNS record(s) in Cloudflare zone, creating a new one", r)
			addr := iplookup.LookupExternalIP()
			if addr.Ipv4 != "" {
				res, err := api.CreateDNSRecord(
					ctx,
					cf.ZoneIdentifier(id),
					cf.CreateDNSRecordParams{
						Name:    r,
						Content: addr.Ipv4,
						Type:    "A",
					})
				if err != nil {
					log.Warnf("Unable to create new IPv4 record for %s: %v", r, err)
				} else {
					log.Infof("Created new IPv4 record for %s", r)
				}
				record[res.Result.ID] = &dnsRecord{
					name:       res.Result.Name,
					IpAddr:     res.Result.Content,
					recordType: res.Result.Type,
				}
			}
			if addr.Ipv6 != "" {
				res, err := api.CreateDNSRecord(
					ctx,
					cf.ZoneIdentifier(id),
					cf.CreateDNSRecordParams{
						Name:    r,
						Content: addr.Ipv6,
						Type:    "AAAA",
					})
				if err != nil {
					log.Warnf("Unable to create new IPv4 record for %s: %v", r, err)
				} else {
					log.Infof("Created new IPv4 record for %s", r)
				}
				record[res.Result.ID] = &dnsRecord{
					name:       res.Result.Name,
					IpAddr:     res.Result.Content,
					recordType: res.Result.Type,
				}
			}
		}
	}

	return &cloudflare{
		api:    api,
		zoneID: id,
		record: record,
	}
}

func (c *cloudflare) CheckAndUpdate() {
	addr := iplookup.LookupExternalIP()
	for id, details := range c.record {
		switch details.recordType {
		case "A":
			if details.IpAddr != addr.Ipv4 {
				log.Debugf("Record %s (type %s) needs updating.  Previously %s, now %s", details.name, details.recordType, details.IpAddr, addr.Ipv4)
				c.setDNSRecord(id, details.recordType, addr.Ipv4)
			}
		case "AAAA":
			if details.IpAddr != addr.Ipv6 {
				log.Debugf("Record %s (type %s) needs updating.  Previously %s, now %s", details.name, details.recordType, details.IpAddr, addr.Ipv6)
				c.setDNSRecord(id, details.recordType, addr.Ipv6)
			}
		}
	}
}

func (c *cloudflare) setDNSRecord(id string, recordType string, addr string) {
	_, err := c.api.CreateDNSRecord(
		context.Background(),
		cf.ResourceIdentifier(id),
		cf.CreateDNSRecordParams{
			Content: addr,
			Type:    recordType,
		})
	if err != nil {
		log.Errorf("Unable to update record %s: %v", id, err)
	}
}

func getRecordsFromConfig() []string {
	records := viper.GetStringSlice("records")
	if records == nil {
		log.Fatal("No records to check found in config? Nothing to do.")
	}
	return records
}
