package probe

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptrace"
	"sync"
	"time"

	"github.com/karimbayli/sentinel-v2/internal/models"
	"go.uber.org/zap"
)

// AnchorURLs are baseline sanity-check targets.
var AnchorURLs = []string{"https://1.1.1.1", "https://google.com", "https://cloudflare.com"}

// Prober performs multi-layer probing (DNS → TCP → TLS → HTTP) against targets.
type Prober struct {
	nodeID     string
	region     string
	targets    []models.Target
	tcpTimeout time.Duration
	httpClient *http.Client
	logger     *zap.Logger
	mu         sync.RWMutex
}

// New creates a new Prober.
func New(nodeID, region string, targets []models.Target, tcpTimeout, httpTimeout time.Duration, logger *zap.Logger) *Prober {
	return &Prober{
		nodeID:     nodeID,
		region:     region,
		targets:    targets,
		tcpTimeout: tcpTimeout,
		httpClient: &http.Client{
			Timeout: httpTimeout,
			Transport: &http.Transport{
				TLSClientConfig:     &tls.Config{InsecureSkipVerify: false},
				MaxIdleConns:        50,
				MaxIdleConnsPerHost: 5,
				IdleConnTimeout:     30 * time.Second,
				DisableKeepAlives:   true,
			},
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= 3 {
					return fmt.Errorf("too many redirects")
				}
				return nil
			},
		},
		logger: logger,
	}
}

// UpdateTargets dynamically updates the target list.
func (p *Prober) UpdateTargets(targets []models.Target) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.targets = targets
}

// RunCycle probes all targets once. It checks anchors first to determine node reliability.
func (p *Prober) RunCycle(ctx context.Context) []models.ProbeResult {
	// Phase 1: Check baseline anchors
	anchorFailures := 0
	var anchorResults []models.ProbeResult
	for _, url := range AnchorURLs {
		// FIX R-7: Per-probe timestamp
		r := p.probeTarget(ctx, url, "ANCHOR")
		anchorResults = append(anchorResults, r)
		if !r.TCPSuccess {
			anchorFailures++
		}
	}

	nodeReliable := anchorFailures < 2
	if !nodeReliable {
		p.logger.Warn("node marked unreliable",
			zap.String("node_id", p.nodeID),
			zap.Int("anchor_failures", anchorFailures))
	}

	// Mark anchor results with reliability
	for i := range anchorResults {
		anchorResults[i].NodeReliable = nodeReliable
	}

	// Phase 2: Probe all configured targets
	p.mu.RLock()
	targets := make([]models.Target, len(p.targets))
	copy(targets, p.targets)
	p.mu.RUnlock()

	var wg sync.WaitGroup
	results := make([]models.ProbeResult, 0, len(targets)+len(anchorResults))
	results = append(results, anchorResults...)

	resultCh := make(chan models.ProbeResult, len(targets))
	sem := make(chan struct{}, 10) // concurrency limiter

	for _, t := range targets {
		if !t.Enabled {
			continue
		}
		if t.Category == "ANCHOR" {
			continue
		}
		wg.Add(1)
		go func(target models.Target) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			// FIX R-7: Per-probe timestamp
			r := p.probeTarget(ctx, target.URL, target.Category)
			r.NodeReliable = nodeReliable
			resultCh <- r
		}(t)
	}

	go func() {
		wg.Wait()
		close(resultCh)
	}()

	for r := range resultCh {
		results = append(results, r)
	}

	return results
}

// probeTarget performs DNS → TCP → TLS → HTTP probe against a single target.
// FIX R-5: DNS is now a separate measurement layer.
// FIX R-7: Each probe gets its own timestamp at start of measurement.
func (p *Prober) probeTarget(ctx context.Context, targetURL, category string) models.ProbeResult {
	// FIX R-7: Timestamp at probe start, not cycle start
	now := time.Now().UTC()

	result := models.ProbeResult{
		Time:           now,
		NodeID:         p.nodeID,
		TargetURL:      targetURL,
		TargetCategory: category,
		NodeReliable:   true,
	}

	// Parse host and port from URL
	host, port, err := parseHostPort(targetURL)
	if err != nil {
		result.ErrorType = "PARSE_ERROR"
		result.ErrorDetail = err.Error()
		return result
	}

	// Phase 0: DNS Resolution (FIX R-5)
	resolvedIP, err := p.measureDNS(ctx, host, &result)
	if err != nil {
		return result
	}

	// Phase 1: TCP Dial (to resolved IP)
	conn, err := p.measureTCP(resolvedIP, port, &result)
	if err != nil {
		return result
	}

	// Phase 2: TLS Handshake over the SAME connection
	err = p.measureTLS(ctx, host, port, conn, &result)
	if err != nil {
		return result
	}

	// Phase 3: HTTP GET (TTFB measured from HTTP request start)
	p.measureHTTP(ctx, targetURL, &result)

	return result
}

func (p *Prober) measureDNS(ctx context.Context, host string, result *models.ProbeResult) (string, error) {
	// Skip for raw IP targets (e.g., https://1.1.1.1)
	resolvedIP := host
	if net.ParseIP(host) == nil {
		dnsStart := time.Now()
		ips, err := net.DefaultResolver.LookupHost(ctx, host)
		dnsDuration := time.Since(dnsStart)
		result.DNSResolveMs = int(dnsDuration.Milliseconds())

		if err != nil {
			result.DNSError = err.Error()
			// FIX A-10: Classify DNS error into specific sub-types
			result.ErrorType = classifyDNSError(err)
			result.ErrorDetail = err.Error()
			return "", err
		}

		if len(ips) > 0 {
			resolvedIP = ips[0]
			result.DNSResolvedIP = resolvedIP
		}
	}
	return resolvedIP, nil
}

func (p *Prober) measureTCP(resolvedIP, port string, result *models.ProbeResult) (net.Conn, error) {
	tcpStart := time.Now()
	conn, err := net.DialTimeout("tcp", net.JoinHostPort(resolvedIP, port), p.tcpTimeout)
	tcpDuration := time.Since(tcpStart)
	result.TCPDialMs = int(tcpDuration.Milliseconds())

	if err != nil {
		result.TCPSuccess = false
		result.ErrorType = "TCP_DIAL"
		result.ErrorDetail = err.Error()
		return nil, err
	}
	result.TCPSuccess = true
	return conn, nil
}

func (p *Prober) measureTLS(ctx context.Context, host, port string, conn net.Conn, result *models.ProbeResult) error {
	if port == "443" || port == "https" {
		tlsStart := time.Now()
		// ServerName must be the hostname, not the IP
		tlsConn := tls.Client(conn, &tls.Config{ServerName: host, InsecureSkipVerify: false})
		err := tlsConn.HandshakeContext(ctx)
		tlsDuration := time.Since(tlsStart)
		result.TLSHandshakeMs = int(tlsDuration.Milliseconds())

		if err != nil {
			conn.Close()
			valid := false
			result.TLSValid = &valid
			result.ErrorType = "TLS_HANDSHAKE"
			result.ErrorDetail = err.Error()
			return err
		}

		state := tlsConn.ConnectionState()
		if len(state.PeerCertificates) > 0 {
			cert := state.PeerCertificates[0]
			valid := time.Now().Before(cert.NotAfter) && time.Now().After(cert.NotBefore)
			result.TLSValid = &valid
			expiry := cert.NotAfter
			result.TLSExpiry = &expiry
			result.CertIssuer = cert.Issuer.CommonName
		}

		tlsConn.Close()
	} else {
		conn.Close()
	}
	return nil
}

func (p *Prober) measureHTTP(ctx context.Context, targetURL string, result *models.ProbeResult) {
	httpStart := time.Now()
	var ttfb time.Duration
	trace := &httptrace.ClientTrace{
		GotFirstResponseByte: func() {
			ttfb = time.Since(httpStart)
		},
	}

	req, err := http.NewRequestWithContext(httptrace.WithClientTrace(ctx, trace), "GET", targetURL, nil)
	if err != nil {
		result.ErrorType = "HTTP_REQUEST"
		result.ErrorDetail = err.Error()
		return
	}
	req.Header.Set("User-Agent", "SentinelV2-Probe/1.0")

	resp, err := p.httpClient.Do(req)
	httpDuration := time.Since(httpStart)

	if err != nil {
		result.ErrorType = "HTTP_GET"
		result.ErrorDetail = err.Error()
		result.TotalMs = int(httpDuration.Milliseconds())
		return
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, io.LimitReader(resp.Body, 1024*1024))

	result.HTTPStatus = resp.StatusCode
	result.TTFBMs = int(ttfb.Milliseconds())
	result.TotalMs = int(httpDuration.Milliseconds())
}

// classifyDNSError inspects a DNS error and returns a specific error type.
// FIX A-10: Enables distinguishing censorship (NXDOMAIN) from infrastructure failure.
func classifyDNSError(err error) string {
	msg := err.Error()
	switch {
	case contains(msg, "no such host"):
		return "DNS_NXDOMAIN" // Domain doesn't exist — possible censorship/blocking
	case contains(msg, "i/o timeout"), contains(msg, "deadline exceeded"):
		return "DNS_TIMEOUT" // DNS server unreachable or too slow
	case contains(msg, "server misbehaving"), contains(msg, "SERVFAIL"):
		return "DNS_SERVFAIL" // DNS server error
	case contains(msg, "connection refused"):
		return "DNS_REFUSED" // DNS server explicitly refused
	default:
		return "DNS_RESOLVE" // Generic DNS failure
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsImpl(s, substr))
}

func containsImpl(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// parseHostPort extracts host and port from a URL string.
func parseHostPort(rawURL string) (host, port string, err error) {
	u := rawURL
	scheme := "https"
	if len(u) > 8 && u[:8] == "https://" {
		u = u[8:]
	} else if len(u) > 7 && u[:7] == "http://" {
		scheme = "http"
		u = u[7:]
	}

	for i, c := range u {
		if c == '/' {
			u = u[:i]
			break
		}
	}

	if h, p, err := net.SplitHostPort(u); err == nil {
		return h, p, nil
	}

	if scheme == "https" {
		return u, "443", nil
	}
	return u, "80", nil
}
