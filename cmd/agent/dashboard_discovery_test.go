package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/shenxianhq/agent/model"
	pb "github.com/shenxianhq/agent/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func TestDashboardDiscoveryRotation(t *testing.T) {
	const secret = "fixture-secret"
	const uuid = "00000000-0000-0000-0000-000000000123"
	old := startDiscoveryTestDashboard(t, 1, secret, uuid)
	next := startDiscoveryTestDashboard(t, 2, secret, uuid)
	denied := startDiscoveryTestDashboard(t, 3, "different-secret", uuid)
	address := func(s *httptest.Server) string { return strings.TrimPrefix(s.URL, "https://") }
	oldAddress, newAddress, deniedAddress := address(old), address(next), address(denied)
	roots := x509.NewCertPool()
	roots.AddCert(next.Certificate())
	options := []grpc.DialOption{
		grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{RootCAs: roots, MinVersion: tls.VersionTLS12})),
		grpc.WithPerRPCCredentials(model.NewAuthHandler(secret, uuid, true)),
	}

	var announced atomic.Value
	announced.Store(oldAddress)
	source := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("client-secret") != "" || r.Header.Get("client-uuid") != "" {
			t.Error("discovery request carried Agent credentials")
		}
		switch server := announced.Load().(string); server {
		case "unavailable":
			w.WriteHeader(http.StatusServiceUnavailable)
		case "redirect":
			http.Redirect(w, r, "/another.json", http.StatusFound)
		default:
			fmt.Fprintf(w, `{"server":%q}`, server)
		}
	}))
	t.Cleanup(source.Close)
	discovery := newDashboardDiscovery(source.URL)
	discovery.http.Transport = source.Client().Transport
	discovery.refresh(time.Now())

	host := &pb.Host{}
	active, err := discovery.connect("old.invalid:5443", options, host)
	if err != nil || active.target != oldAddress || active.receipt.GetData() != 1 {
		t.Fatalf("initial TLS connection = %q, %v", active.target, err)
	}
	defer active.conn.Close()
	discovery.lastGood = active.target

	refresh := func(value string) (dashboardConnection, error) {
		announced.Store(value)
		discovery.nextRefresh = time.Time{}
		return discovery.replacement(active.target, options, host)
	}
	for _, value := range []string{"unavailable", "redirect", "https://invalid.example:443", deniedAddress} {
		candidate, err := refresh(value)
		if candidate.conn != nil || (value == deniedAddress && status.Code(err) != codes.Unauthenticated) {
			t.Fatalf("unexpected replacement for %q: %q, %v", value, candidate.target, err)
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if _, err := active.client.ReportSystemInfo2(ctx, host); err != nil {
		t.Fatalf("old connection lost after failed replacement: %v", err)
	}
	fallback, err := discovery.connect("old.invalid:5443", options, host)
	if err != nil || fallback.target != oldAddress {
		t.Fatalf("reconnect fallback = %q, %v", fallback.target, err)
	}
	fallback.conn.Close()
	old.Close()

	candidate, err := refresh(newAddress)
	if err != nil || candidate.target != newAddress || candidate.receipt.GetData() != 2 {
		t.Fatalf("replacement TLS connection = %q, %v", candidate.target, err)
	}
	candidate.conn.Close()
}

func TestDashboardDiscoveryTrustIsLocal(t *testing.T) {
	config := &model.AgentConfig{TLS: true, ServerDiscoveryURL: "https://discovery.example.net/panel.json"}
	if err := model.ValidateConfig(config, true); err != nil {
		t.Fatal(err)
	}
	for _, edit := range []string{
		`{"server_discovery_url":""}`,
		`{"server_discovery_url":"https://other.example.net/panel.json"}`,
		`{"insecure_tls":true}`,
		`{"tls":false}`,
	} {
		if _, ok := parseApplyConfigLocked(config, &pb.Task{Data: edit}, &pb.TaskResult{}); ok {
			t.Fatalf("unsafe remote change accepted: %s", edit)
		}
	}
	legacy := newDashboardDiscovery("")
	if got := legacy.targets("legacy:8008"); len(got) != 1 || got[0] != "legacy:8008" {
		t.Fatalf("legacy targets = %v", got)
	}
}

type discoveryTestDashboard struct {
	pb.UnimplementedShenxianServiceServer
	boot         uint64
	secret, uuid string
}

func (d discoveryTestDashboard) ReportSystemInfo2(ctx context.Context, _ *pb.Host) (*pb.Uint64Receipt, error) {
	md, _ := metadata.FromIncomingContext(ctx)
	if strings.Join(md.Get("client-secret"), "") != d.secret || strings.Join(md.Get("client-uuid"), "") != d.uuid {
		return nil, status.Error(codes.Unauthenticated, "unexpected credentials")
	}
	return &pb.Uint64Receipt{Data: d.boot}, nil
}

func startDiscoveryTestDashboard(t *testing.T, boot uint64, secret, uuid string) *httptest.Server {
	t.Helper()
	rpc := grpc.NewServer()
	pb.RegisterShenxianServiceServer(rpc, discoveryTestDashboard{boot: boot, secret: secret, uuid: uuid})
	s := httptest.NewUnstartedServer(http.HandlerFunc(rpc.ServeHTTP))
	s.EnableHTTP2 = true
	s.StartTLS()
	t.Cleanup(s.Close)
	return s
}
