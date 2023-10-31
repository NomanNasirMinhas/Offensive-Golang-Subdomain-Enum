package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/miekg/dns"
)

type result struct {
	Subdomain string
	IP        string
}

type Message struct {
	MsgHdr   dns.MsgHdr
	Question []dns.Question
	Answer   []dns.RR
	Ns       []dns.RR // Name server records
	Extra    []dns.RR // Additional records
	Compress bool     // If true, the message was compressed
}

func getDNSResult(fqdn, dnsServer string, DNStype uint16) ([]string, error) {
	var msg dns.Msg
	var ips []string
	msg.SetQuestion(dns.Fqdn(fqdn), DNStype)
	in, err := dns.Exchange(&msg, dnsServer)
	if err != nil {
		panic(err)
	}
	if len(in.Answer) < 1 {
		return ips, err
	}
	for _, answer := range in.Answer {
		if DNStype == dns.TypeA {
			if a, ok := answer.(*dns.A); ok {
				ips = append(ips, a.A.String())
			}
		} else if DNStype == dns.TypeCNAME {
			if cname, ok := answer.(*dns.CNAME); ok {
				ips = append(ips, cname.Target)
			}
		} else {
			// fmt.Println("Invalid DNS type")
		}
	}
	return ips, nil
}

func main() {
	var (
		flDomain    = flag.String("domain", "", "domain to enumemrate")
		flWordlist  = flag.String("wordlist", "", "wordlist to use")
		flThreads   = flag.Int("threads", 100, "number of threads to use")
		flDNSServer = flag.String("dns", "8.8.8.8:53", "dns server to use")
	)
	flag.Parse()
	if *flDomain == "" || *flWordlist == "" {
		fmt.Println("Usage: ./dnsclient -domain example.com -wordlist wordlist.txt")
		flag.PrintDefaults()
		return
	}
	fh, err := os.Open(*flWordlist)
	if err != nil {
		panic(err)
	}
	defer fh.Close()
	scanner := bufio.NewScanner(fh)
	var results []result

	fqdns := make(chan string, *flThreads)
	gather := make(chan []result)
	tracker := make(chan empty)

	// Start the workers
	for i := 0; i < *flThreads; i++ {
		go worker(tracker, fqdns, gather, *flDNSServer)
	}

	for scanner.Scan() {
		fqdns <- fmt.Sprintf("%s.%s", scanner.Text(), *flDomain)
	}

	go func() {
		for r := range gather {
			results = append(results, r...)
		}
		var e empty
		tracker <- e
	}()

	close(fqdns)
	for i := 0; i < *flThreads; i++ {
		<-tracker
	}
	close(gather)
	<-tracker

	res := tabwriter.NewWriter(os.Stdout, 0, 8, 4, ' ', 0)
	for _, r := range results {
		fmt.Fprintf(res, "%s\t%s\n", r.Subdomain, r.IP)
	}
	res.Flush()
}
