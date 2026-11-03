package main

import (
	"context"
	"errors"
	"io"
	"net"
	"testing"

	pb "github.com/shenxianhq/agent/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type grpcTCPService struct {
	pb.UnimplementedShenxianServiceServer
	requestTask       func(pb.ShenxianService_RequestTaskServer) error
	reportSystemState func(pb.ShenxianService_ReportSystemStateServer) error
	ioStream          func(pb.ShenxianService_IOStreamServer) error
}

func (s *grpcTCPService) RequestTask(stream pb.ShenxianService_RequestTaskServer) error {
	return s.requestTask(stream)
}

func (s *grpcTCPService) ReportSystemState(stream pb.ShenxianService_ReportSystemStateServer) error {
	return s.reportSystemState(stream)
}

func (s *grpcTCPService) IOStream(stream pb.ShenxianService_IOStreamServer) error {
	return s.ioStream(stream)
}

type grpcTCPFixture struct {
	client pb.ShenxianServiceClient
}

func newGRPCTCPFixture(t *testing.T, service *grpcTCPService) *grpcTCPFixture {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen grpc tcp: %v", err)
	}
	server := grpc.NewServer()
	pb.RegisterShenxianServiceServer(server, service)
	serveResult := make(chan error, 1)
	go func() { serveResult <- server.Serve(listener) }()

	dialContext, cancelDial := context.WithTimeout(context.Background(), streamFixtureDeadline)
	defer cancelDial()
	connection, err := grpc.DialContext(
		dialContext,
		listener.Addr().String(),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		server.Stop()
		_ = listener.Close()
		t.Fatalf("dial grpc tcp: %v", err)
	}

	t.Cleanup(func() {
		if err := connection.Close(); err != nil {
			t.Errorf("close grpc tcp client: %v", err)
		}
		server.Stop()
		if err := listener.Close(); err != nil && !errors.Is(err, net.ErrClosed) {
			t.Errorf("close grpc tcp listener: %v", err)
		}
		serveErr := awaitStreamOperationResult(t, serveResult)
		if serveErr != nil && !errors.Is(serveErr, grpc.ErrServerStopped) {
			t.Errorf("serve grpc tcp: %v", serveErr)
		}
	})

	return &grpcTCPFixture{client: pb.NewShenxianServiceClient(connection)}
}

func grpcTCPDrainIOStream(stream pb.ShenxianService_IOStreamServer, frames chan<- []byte) error {
	for {
		message, err := stream.Recv()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}
		frames <- append([]byte(nil), message.GetData()...)
	}
}
