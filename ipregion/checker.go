package ipregion

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"

	"golang.org/x/sync/errgroup"
)

// Checker runs IP region probes against external services.
type Checker struct {
	client       *httpClient
	keys         ServiceKeys
	opts         Options
	externalIPv4 string
	externalIPv6 string
	peerResults  map[string]string
	peerMu       sync.RWMutex
}

// NewChecker creates a Checker with default options applied at Run time.
func NewChecker() *Checker {
	return &Checker{}
}

func normalizeOptions(opts Options) Options {
	if len(opts.Groups) == 0 {
		opts.Groups = []Group{GroupPrimary, GroupCustom}
	}
	if opts.Timeout <= 0 {
		opts.Timeout = defaultTimeout
	}
	if opts.UserAgent == "" {
		opts.UserAgent = defaultUserAgent
	}
	if opts.MaxConcurrency <= 0 {
		opts.MaxConcurrency = defaultMaxConcurrency
	}
	return opts
}

type probeTask struct {
	group   Group
	service string
	run     func(ctx context.Context) (ipv4, ipv6 string)
}

func (c *Checker) setPeerResult(service string, ipVersion int, value string) {
	key := peerKey(service, ipVersion)
	c.peerMu.Lock()
	defer c.peerMu.Unlock()
	if c.peerResults == nil {
		c.peerResults = make(map[string]string)
	}
	c.peerResults[key] = value
}

func (c *Checker) peerResult(service string, ipVersion int) string {
	c.peerMu.RLock()
	defer c.peerMu.RUnlock()
	if c.peerResults == nil {
		return ""
	}
	return c.peerResults[peerKey(service, ipVersion)]
}

func peerKey(service string, ipVersion int) string {
	return fmt.Sprintf("%s:%d", service, ipVersion)
}

// Run executes the configured scan and returns a report.
func (c *Checker) Run(ctx context.Context, opts Options) (*Report, error) {
	opts = normalizeOptions(opts)
	c.opts = opts
	c.keys = opts.ServiceKeys
	c.client = newHTTPClient(opts.Timeout, opts.UserAgent)
	c.peerResults = make(map[string]string)

	report := &Report{}

	tasks := c.buildTasks(opts)
	taskCount := len(tasks)
	if taskCount == 0 {
		return report, nil
	}

	prepSteps := 0
	if !opts.IPv6Only {
		prepSteps++
	}
	if !opts.IPv4Only {
		prepSteps++
	}
	prepSteps++ // ASN lookup

	totalWork := prepSteps + taskCount

	emitProgress := func(service string, completed int) {
		if opts.OnProgress != nil {
			opts.OnProgress(Progress{
				Service:   service,
				Completed: completed,
				Total:     totalWork,
			})
		}
	}

	done := 0
	if !opts.IPv6Only {
		emitProgress("Detecting IPv4", done)
		report.ExternalIPv4 = detectExternalIP(ctx, c.client, 4)
		c.externalIPv4 = report.ExternalIPv4
		done++
	}
	if !opts.IPv4Only {
		emitProgress("Detecting IPv6", done)
		report.ExternalIPv6 = detectExternalIP(ctx, c.client, 6)
		c.externalIPv6 = report.ExternalIPv6
		done++
	}

	ip := report.ExternalIPv4
	ipVersion := 4
	if ip == "" {
		ip = report.ExternalIPv6
		ipVersion = 6
	}
	emitProgress("Detecting ASN", done)
	if ip != "" {
		report.ASN, report.ASNOrg = fetchASN(ctx, c.client, ip, ipVersion)
	}
	done++

	prepDone := done

	results := make([]ServiceResult, taskCount)
	var completed atomic.Int32

	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(opts.MaxConcurrency)

	for i, task := range tasks {
		g.Go(func() error {
			if gctx.Err() != nil {
				return nil
			}
			if opts.OnProgress != nil {
				emitProgress(task.service, prepDone+int(completed.Load()))
			}
			ipv4, ipv6 := task.run(gctx)
			results[i] = ServiceResult{
				Group:   task.group,
				Service: task.service,
				IPv4:    ipv4,
				IPv6:    ipv6,
			}
			completed.Add(1)
			if opts.OnProgress != nil {
				emitProgress(task.service, prepDone+int(completed.Load()))
			}
			return nil
		})
	}

	_ = g.Wait()
	linkYouTubeFromGoogle(report)
	report.Results = results
	return report, ctx.Err()
}

func (c *Checker) buildTasks(opts Options) []probeTask {
	var tasks []probeTask
	groupSet := make(map[Group]bool)
	for _, g := range opts.Groups {
		groupSet[g] = true
	}

	if groupSet[GroupPrimary] {
		for _, svc := range defaultPrimaryServices(opts.ServiceKeys) {
			svc := svc
			tasks = append(tasks, probeTask{
				group:   GroupPrimary,
				service: svc.displayName,
				run: func(ctx context.Context) (string, string) {
					return c.probeServicePair(ctx, opts, func(ctx context.Context, ver int, ip string) string {
						return c.probePrimary(ctx, svc, ver, ip)
					})
				},
			})
		}
	}

	if groupSet[GroupCustom] {
		for _, svc := range defaultCustomServices() {
			svc := svc
			tasks = append(tasks, probeTask{
				group:   GroupCustom,
				service: svc.displayName,
				run: func(ctx context.Context) (string, string) {
					var ipv4, ipv6 string
					if !opts.IPv6Only {
						ipv4 = cleanResult(svc.probe(c, ctx, 4))
						if svc.displayName == "Google" && ipv4 != NotAvailable {
							c.setPeerResult("Google", 4, ipv4)
						}
					}
					if !opts.IPv4Only && c.externalIPv6 != "" {
						ipv6 = cleanResult(svc.probe(c, ctx, 6))
						if svc.displayName == "Google" && ipv6 != NotAvailable {
							c.setPeerResult("Google", 6, ipv6)
						}
					}
					return ipv4, ipv6
				},
			})
		}
	}

	return tasks
}

func (c *Checker) probeServicePair(ctx context.Context, opts Options, probe func(ctx context.Context, ver int, ip string) string) (string, string) {
	var ipv4, ipv6 string
	if !opts.IPv6Only && c.externalIPv4 != "" {
		ipv4 = probe(ctx, 4, c.externalIPv4)
	}
	if !opts.IPv4Only && c.externalIPv6 != "" {
		ipv6 = probe(ctx, 6, c.externalIPv6)
	}
	if ipv4 == "" {
		ipv4 = NotAvailable
	}
	if ipv6 == "" {
		ipv6 = NotAvailable
	}
	return ipv4, ipv6
}

// ResultByGroup returns results filtered by group in stable order.
func ResultByGroup(report *Report, group Group) []ServiceResult {
	if report == nil {
		return nil
	}
	var out []ServiceResult
	for _, r := range report.Results {
		if r.Group == group {
			out = append(out, r)
		}
	}
	return out
}

// FormatSummaryLine returns a short human-readable consensus line.
func FormatSummaryLine(report *Report) string {
	s := BuildSummary(report)
	if len(s.Countries) == 0 {
		return ""
	}
	top := s.Countries[0]
	pct := top.IPv4Pct
	if pct == 0 {
		pct = top.IPv6Pct
	}
	return fmt.Sprintf("%s (%s) %d%%", top.Code, top.Name, pct)
}

func linkYouTubeFromGoogle(report *Report) {
	if report == nil {
		return
	}
	var googleIPv4, googleIPv6 string
	for _, r := range report.Results {
		if r.Service == "Google" {
			googleIPv4, googleIPv6 = r.IPv4, r.IPv6
			break
		}
	}
	for i := range report.Results {
		if report.Results[i].Service != "YouTube" {
			continue
		}
		if report.Results[i].IPv4 == NotAvailable && googleIPv4 != NotAvailable && googleIPv4 != "" {
			report.Results[i].IPv4 = googleIPv4
		}
		if report.Results[i].IPv6 == NotAvailable && googleIPv6 != NotAvailable && googleIPv6 != "" {
			report.Results[i].IPv6 = googleIPv6
		}
	}
}
