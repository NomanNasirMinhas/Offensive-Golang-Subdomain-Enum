package main

import "github.com/miekg/dns"

type empty struct{}

func lookup(fqdn, serverAddr string) []result {
	var results []result
	var cfqdn = fqdn // Don't modify the original.
	for {
		cnames, err := getDNSResult(cfqdn, serverAddr, dns.TypeCNAME)
		if err == nil && len(cnames) > 0 {
			cfqdn = cnames[0]
			continue // We have to process the next CNAME.
		}
		ips, err := getDNSResult(cfqdn, serverAddr, dns.TypeA)
		if err != nil {
			break // There are no A records for this hostname.
		}
		for _, ip := range ips {
			results = append(results, result{IP: ip, Subdomain: fqdn})
		}
		break // We have processed all the results.
	}
	return results
}

func worker(tracker chan empty, fqdns chan string, gather chan []result, serverAddr string) {
	for fqdn := range fqdns {
		results := lookup(fqdn, serverAddr)
		if len(results) > 0 {
			gather <- results
		}
	}
	var e empty
	tracker <- e
}
