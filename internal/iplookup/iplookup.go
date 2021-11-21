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
package iplookup

import (
	"net"
	"sync"

	"github.com/go-resty/resty/v2"
	log "github.com/sirupsen/logrus"
)

type address struct {
	Ipv4, Ipv6 string
}

var ipLookupHosts = map[string]address{
	"icanhazip": {Ipv4: "https://4.icanhazip.com", Ipv6: "https://6.icanhazip.com"},
	"ipify":     {Ipv4: "https://api.ipify.org", Ipv6: "https://api6.ipify.org"},
}

func LookupExternalIP() *address {
	var wg sync.WaitGroup

	var externalAddr address

	for host, addr := range ipLookupHosts {
		log.Debugf("Trying to find external IP addresses with %s", host)
		wg.Add(2)
		go func() {
			defer wg.Done()
			client := resty.New().SetLogger(&log.Logger{})
			client.SetRetryCount(3)
			resp, err := client.R().Get(addr.Ipv4)
			if err != nil {
				log.Warnf("Unable to retrieve external IPv4 address: %v", err)
			}
			if resp.StatusCode() == 200 && resp.Body() != nil {
				externalAddr.Ipv4 = resp.String()
				log.Debugf("Found external IPv4 address %s using %s", externalAddr.Ipv4, host)
			}
		}()
		go func() {
			defer wg.Done()
			client := resty.New().SetLogger(&log.Logger{})
			client.SetRetryCount(3)
			resp, err := client.R().Get(addr.Ipv6)
			if err != nil {
				log.Warnf("Unable to retrieve external IPv6 address: %v", err)
			}
			if resp.StatusCode() == 200 && resp.Body() != nil {
				log.Debugf("Found external IPv6 address %s using %s", externalAddr.Ipv6, host)
				externalAddr.Ipv6 = resp.String()
			}
		}()
		wg.Wait()
		// If we have at least, either a valid v4 or v6 external address,
		// return those
		if net.ParseIP(externalAddr.Ipv4) != nil || net.ParseIP(externalAddr.Ipv6) != nil {
			return &externalAddr
		}
	}
	// At this point, we've gone through all IP checkers and not found an
	// external address
	log.Warn("Couldn't retrieve any external IP address.")
	return &address{}
}
