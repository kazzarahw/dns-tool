//usr/bin/env go run "$0" "$@"; exit
// DNS Tool
// by: Kazzarah

// Package
package main

// Imports
import (
	"flag"
	"fmt"
	"net"
	"os"
	"slices"
	"sync"
	"text/tabwriter"
	"time"

	"github.com/miekg/dns"
)

// Handler
type Handler struct {
	Domain     string
	Nameserver string
	Client     *dns.Client
	Transfer   *dns.Transfer
}

// New Handler
func NewHandler(domain, nameserver string) *Handler {
	// Client
	client := &dns.Client{}
	client.Timeout = 5 * time.Second
	// Transfer
	transfer := &dns.Transfer{}
	transfer.DialTimeout = 5 * time.Second
	transfer.ReadTimeout = 5 * time.Second
	transfer.WriteTimeout = 5 * time.Second
	// Return
	return &Handler{
		Domain:     domain,
		Nameserver: nameserver,
		Client:     client,
		Transfer:   transfer,
	}
}

// Query DNS
func (h *Handler) Query(dnsType uint16) []dns.RR {
	// Create message
	msg := dns.Msg{}
	msg.SetQuestion(h.Domain, dnsType)
	// Query
	resp, _, err := h.Client.Exchange(&msg, h.Nameserver)
	if err != nil {
		return h.Query(dnsType)
	}
	// Return
	return resp.Answer
}

// Query All
func (h *Handler) QueryAll() []dns.RR {
	// DNS Type Blacklist
	blacklist := []uint16{
		dns.TypeANY,
		dns.TypeAXFR,
	}
	resp := []dns.RR{}
	// Query all
	wg := new(sync.WaitGroup)
	for dnsType := range dns.TypeToString {
		if !slices.Contains(blacklist, dnsType) {
			wg.Add(1)
			go func(dnsType uint16) {
				defer wg.Done()
				resp = append(resp, h.Query(dnsType)...)
			}(dnsType)
		}
	}
	// Wait
	wg.Wait()
	return resp
}

// AXFR
func (h *Handler) ZoneTransfer() ([]dns.RR, error) {
	// Create message
	msg := dns.Msg{}
	msg.SetAxfr(h.Domain)
	// Query
	env, err := h.Transfer.In(&msg, h.Nameserver)
	if err != nil {
		return nil, err
	}
	resp := []dns.RR{}
	// Print results
	for answer := range env {
		if answer.Error != nil {
			return nil, answer.Error
		}
		resp = append(resp, answer.RR...)
	}
	return resp, nil
}

// Help
func help() {
	fmt.Println("DNS-Tool")
	fmt.Println("by: Kazzarah")
	fmt.Println("")
	fmt.Println("Usage: ./dns-tool -d <domain> -ns <nameserver>")
	fmt.Println("")
	fmt.Println("Example: ./dns-tool -d zonetransfer.me -ns nsztm2.digi.ninja:53")
	fmt.Println("")
}

// Pretty Print
func prettyPrint(rr []dns.RR) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', 0)
	for _, r := range rr {
		fmt.Fprintf(w, "%s\n", r)
	}
	w.Flush()
}

// Main
func main() {
	// Parse variables from CLI flags
	domainPtr := flag.String("d", "", "Domain to query")
	nameserverPtr := flag.String("ns", "8.8.8.8:53", "Nameserver to use")
	helpPtr := flag.Bool("h", false, "Help")
	flag.Parse()

	// Unpack variables
	domain := *domainPtr
	nameserver := *nameserverPtr

	// Help
	if *helpPtr || domain == "" {
		help()
		return
	}

	// Assume nameserver port if not specified
	_, _, err := net.SplitHostPort(nameserver)
	if err != nil {
		nameserver = nameserver + ":53"
	}

	// Assert FQDN
	if !dns.IsFqdn(domain) {
		domain = dns.Fqdn(domain)
	}

	// // Print variables
	// fmt.Printf("Domain: %s\n", domain)
	// fmt.Printf("Nameserver: %s\n", nameserver)
	// fmt.Println("")

	// New Handler
	handler := NewHandler(domain, nameserver)

	// Run
	resp, err := handler.ZoneTransfer()
	if err != nil {
		resp = handler.QueryAll()
	}

	// Print
	prettyPrint(resp)
}
