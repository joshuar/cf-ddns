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
	api     *cf.API
	zoneID  string
	records map[string]*dnsRecord
}

type dnsRecord struct {
	name, iptype, ipaddr string
}

func NewCloudflare() *cloudflare {
	log.Debug("Establishing API connection")
	api, err := cf.New(
		viper.GetString("account.apiKey"),
		viper.GetString("account.email"))
	if err != nil {
		log.Fatalf("Unable to establish Cloudflare API connection: %v", err)
	}

	log.Debug("Retrieving Zone ID")
	id, err := api.ZoneIDByName(viper.GetString("account.zone"))
	if err != nil {
		log.Fatalf("Unable to retrieve zone ID from Cloudflare API: %v", err)
	}

	cfDetails := &cloudflare{
		api:     api,
		zoneID:  id,
		records: make(map[string]*dnsRecord),
	}

	if viper.GetStringSlice("records") == nil {
		log.Fatal("No records specified in config, nothing to do")
	}

	for _, r := range viper.GetStringSlice("records") {
		log.Debugf("Getting DNS record(s) for %s", r)
		recs, _, err := api.ListDNSRecords(
			context.Background(),
			cf.ZoneIdentifier(cfDetails.zoneID),
			cf.ListDNSRecordsParams{Name: r})
		if err != nil {
			log.Fatalf("Unable to retrieve record details for %s from Cloudflare API: %v", r, err)
		}
		if recs != nil {
			for _, r := range recs {
				cfDetails.records[r.ID] = &dnsRecord{
					name:   r.Name,
					ipaddr: r.Content,
					iptype: r.Type,
				}
			}
		} else {
			log.Warnf("%s has no matching DNS record(s) in Cloudflare zone, creating them", r)
			addr := iplookup.LookupExternalIP()
			if addr.IPv4 != nil {
				record := &dnsRecord{
					name:   r,
					iptype: "A",
					ipaddr: addr.IPv4.String(),
				}
				cfDetails.setDNSRecord(record)
			}
			if addr.IPv6 != nil {
				record := &dnsRecord{
					name:   r,
					iptype: "AAAA",
					ipaddr: addr.IPv6.String(),
				}
				cfDetails.setDNSRecord(record)
			}
		}
	}

	return cfDetails
}

func (c *cloudflare) CheckAndUpdate() {
	addr := iplookup.LookupExternalIP()
	if addr != nil {
		for recordID, details := range c.records {
			switch details.iptype {
			case "A":
				if details.ipaddr != addr.IPv4.String() {
					log.Debugf("Record %s (type %s) needs updating.  Previously %s, now %s", details.name, details.iptype, details.ipaddr, addr.IPv4.String())
					c.setDNSRecord(c.records[recordID])
				}
			case "AAAA":
				if details.ipaddr != addr.IPv6.String() {
					log.Debugf("Record %s (type %s) needs updating.  Previously %s, now %s", details.name, details.iptype, details.ipaddr, addr.IPv6.String())
					c.setDNSRecord(c.records[recordID])
				}
			}
		}
	}
}

func (c *cloudflare) setDNSRecord(record *dnsRecord) {
	_, err := c.api.CreateDNSRecord(
		context.Background(),
		cf.ResourceIdentifier(c.zoneID),
		cf.CreateDNSRecordParams{
			Name:    record.name,
			Content: record.ipaddr,
			Type:    record.iptype,
			TTL:     1,
		})
	if err != nil {
		log.Errorf("Unable to update record %s (type %s, content %s): %v", record.name, record.iptype, record.ipaddr, err)
	}
}
