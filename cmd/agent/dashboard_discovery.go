package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	pb "github.com/shenxianhq/agent/proto"
	"google.golang.org/grpc"
)

const (
	dashboardDiscoveryInterval = time.Minute
	dashboardDiscoveryMaxBody  = 4096
)

var dashboardHostnameLabel = regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?$`)

type dashboardDiscovery struct {
	sources     []string
	http        *http.Client
	nextRefresh time.Time
	preferred   string
	revision    uint64
	lastGood    string
}

func newDashboardDiscovery(sources ...string) *dashboardDiscovery {
	unique := make([]string, 0, len(sources))
	for _, source := range sources {
		if source != "" && !slices.Contains(unique, source) {
			unique = append(unique, source)
		}
	}
	return &dashboardDiscovery{
		sources: unique,
		http: &http.Client{
			Timeout: 5 * time.Second,
			CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}
}

func (d *dashboardDiscovery) refresh(now time.Time) {
	if len(d.sources) == 0 || now.Before(d.nextRefresh) {
		return
	}
	d.nextRefresh = now.Add(dashboardDiscoveryInterval)
	type result struct {
		manifest dashboardManifest
		index    int
	}
	replies := make(chan result, len(d.sources))
	for index, source := range d.sources {
		go func() {
			replies <- result{manifest: d.fetch(source), index: index}
		}()
	}
	best := result{index: len(d.sources)}
	for range d.sources {
		reply := <-replies
		if reply.manifest.Server != "" && (best.manifest.Server == "" ||
			reply.manifest.Revision > best.manifest.Revision ||
			(reply.manifest.Revision == best.manifest.Revision && reply.index < best.index)) {
			best = reply
		}
	}
	if best.manifest.Server == "" {
		println("面板地址清单读取失败，将继续使用已知地址")
		return
	}
	if best.manifest.Revision < d.revision {
		return
	}
	if d.revision > 0 && best.manifest.Revision == d.revision && best.manifest.Server != d.preferred {
		return
	}
	d.preferred, d.revision = best.manifest.Server, best.manifest.Revision
}

type dashboardManifest struct {
	Server   string `json:"server"`
	Revision uint64 `json:"revision"`
}

func (d *dashboardDiscovery) fetch(source string) dashboardManifest {
	req, err := http.NewRequest(http.MethodGet, source, nil)
	if err != nil {
		return dashboardManifest{}
	}
	req.Header.Set("Cache-Control", "no-cache")
	resp, err := d.http.Do(req)
	if err != nil {
		return dashboardManifest{}
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return dashboardManifest{}
	}
	payload, err := io.ReadAll(io.LimitReader(resp.Body, dashboardDiscoveryMaxBody+1))
	if err != nil {
		return dashboardManifest{}
	}
	var manifest dashboardManifest
	if len(payload) > dashboardDiscoveryMaxBody || json.Unmarshal(payload, &manifest) != nil || !validDashboardTarget(manifest.Server) {
		return dashboardManifest{}
	}
	return manifest
}

func validDashboardTarget(target string) bool {
	host, port, err := net.SplitHostPort(target)
	if err != nil || host == "" {
		return false
	}
	number, err := strconv.ParseUint(port, 10, 16)
	if err != nil || number == 0 {
		return false
	}
	if net.ParseIP(host) != nil {
		return true
	}
	hostname := strings.TrimSuffix(host, ".")
	if len(hostname) > 253 || strings.HasPrefix(target, "[") {
		return false
	}
	for _, label := range strings.Split(hostname, ".") {
		if !dashboardHostnameLabel.MatchString(label) {
			return false
		}
	}
	return true
}

func (d *dashboardDiscovery) targets(configured string) []string {
	if len(d.sources) == 0 {
		return []string{configured}
	}
	targets := make([]string, 0, 3)
	for _, target := range []string{d.preferred, d.lastGood, configured} {
		if target != "" && !slices.Contains(targets, target) {
			targets = append(targets, target)
		}
	}
	return targets
}

type dashboardConnection struct {
	conn    *grpc.ClientConn
	client  pb.ShenxianServiceClient
	receipt *pb.Uint64Receipt
	target  string
}

func (d *dashboardDiscovery) connect(configured string, options []grpc.DialOption, host *pb.Host) (dashboardConnection, error) {
	lastErr := errors.New("no dashboard address configured")
	for _, target := range d.targets(configured) {
		connection, err := connectDashboard(target, options, host)
		if err == nil {
			return connection, nil
		}
		lastErr = err
	}
	return dashboardConnection{}, lastErr
}

func (d *dashboardDiscovery) replacement(active string, options []grpc.DialOption, host *pb.Host) (dashboardConnection, error) {
	d.refresh(time.Now())
	if d.preferred == "" || d.preferred == active {
		return dashboardConnection{}, nil
	}
	return connectDashboard(d.preferred, options, host)
}

func connectDashboard(target string, options []grpc.DialOption, host *pb.Host) (dashboardConnection, error) {
	conn, err := grpc.NewClient(target, options...)
	if err != nil {
		return dashboardConnection{}, err
	}
	client := pb.NewShenxianServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	receipt, err := client.ReportSystemInfo2(ctx, host)
	if err != nil {
		conn.Close()
		return dashboardConnection{}, err
	}
	return dashboardConnection{conn: conn, client: client, receipt: receipt, target: target}, nil
}
