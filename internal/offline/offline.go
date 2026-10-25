// Package offline is the one classifier for a fetch that never reached its site: no connection,
// no DNS, or a refused or timed-out connection. Every feed words such a failure the same way,
// "Can't reach <site>. Check your internet connection, then try again.", and the error chain
// behind it goes to the log, not the screen. Standard library only, so any layer may use it.
package offline

import (
	"errors"
	"log"
	"net"
	"net/url"
	"regexp"
	"strings"
)

// Advice ends every offline sentence; the frontend finds the sentence by it.
const Advice = " Check your internet connection, then try again."

// League is the sentence for MFL, the site most screens read.
const League = "Can't reach the league site." + Advice

// Sentence names a recognised site without exposing its URL.
func Sentence(host string) string {
	host = strings.ToLower(strings.TrimSuffix(host, "."))
	sites := []struct{ suffix, name string }{
		{"myfantasyleague.com", "the league site"}, {"boards.net", "the league message board"},
		{"espn.com", "ESPN"}, {"espncdn.com", "ESPN"}, {"yahoo.com", "Yahoo"},
		{"sleepercdn.com", "Sleeper"}, {"github.com", "the NFL data site"},
		{"githubusercontent.com", "the NFL data site"}, {"collegefootballdata.com", "the college stats site"},
		{"ea.com", "the EA ratings site"},
	}
	for _, site := range sites {
		if host == site.suffix || strings.HasSuffix(host, "."+site.suffix) {
			return "Can't reach " + site.name + "." + Advice
		}
	}
	return "Can't reach the internet." + Advice
}

// Error keeps the network cause for errors.Is and the log while reading as its sentence alone.
type Error struct {
	sentence string
	cause    error
}

func (e Error) Error() string { return e.sentence }
func (e Error) Unwrap() error { return e.cause }

// Plain returns a network failure as an Error naming the site it could not reach, logging the
// chain once; any other error, and nil, is returned unchanged.
func Plain(err error) error {
	var plain Error
	if err == nil {
		return nil
	}
	if errors.As(err, &plain) {
		return plain // already logged; the context wrapped around it is dropped from the screen
	}
	host, ok := unreachable(err)
	if !ok {
		return err
	}
	log.Printf("the war room: offline: %v", err)
	return Error{sentence: Sentence(host), cause: err}
}

// Text is Plain's words: the sentence alone for a network failure, otherwise err.Error().
func Text(err error) string {
	if err == nil {
		return ""
	}
	return Plain(err).Error()
}

// Is reports whether err is a fetch that never reached its site.
func Is(err error) bool {
	var plain Error
	if errors.As(err, &plain) {
		return true
	}
	_, ok := unreachable(err)
	return ok
}

// Find extracts a sentence already made safe for display.
func Find(text string) string {
	start := strings.Index(text, "Can't reach ")
	if start < 0 {
		return ""
	}
	end := strings.Index(text[start:], Advice)
	if end < 0 {
		return ""
	}
	return text[start : start+end+len(Advice)]
}

// Recorded handles network failures persisted without their error chain.
func Recorded(from, text string) string {
	recorded := []string{"dial tcp", "dial udp", "no such host", "connection refused", "i/o timeout",
		"network is unreachable", "no route to host", "connection reset by peer", "Client.Timeout exceeded",
		"server misbehaving", "TLS handshake timeout"}
	for _, words := range recorded {
		if !strings.Contains(text, words) {
			continue
		}
		if u, err := url.Parse(from); err == nil && u.Hostname() != "" {
			return Sentence(u.Hostname())
		}
		if m := regexp.MustCompile(`https?://([^/"\s:?]+)`).FindStringSubmatch(text); m != nil {
			return Sentence(m[1])
		}
		return Sentence("")
	}
	return text
}

// unreachable reports a failure to connect, and the host it was trying when the chain says.
func unreachable(err error) (string, bool) {
	var fetch *url.Error
	host := ""
	if errors.As(err, &fetch) {
		if u, perr := url.Parse(fetch.URL); perr == nil {
			host = u.Hostname()
		}
		if fetch.Timeout() {
			return host, true
		}
	}
	var dns *net.DNSError
	if errors.As(err, &dns) {
		if host == "" {
			host = dns.Name
		}
		return host, true
	}
	var op *net.OpError
	return host, errors.As(err, &op)
}
