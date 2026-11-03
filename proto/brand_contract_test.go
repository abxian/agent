package proto

import (
	"google.golang.org/protobuf/proto"
	"testing"
)

func TestShenxianDescriptorAndGPULegacyWire(t *testing.T) {
	if got := File_proto_shenxian_proto.Services().Get(0).FullName(); got != "proto.ShenxianService" {
		t.Fatal(got)
	}
	state := &State{Gpu: []float64{50}, Gpus: []*State_GPU{{Utilization: 50, MemoryUsed: 128, MemoryTotal: 1024}}}
	b, err := proto.Marshal(state)
	if err != nil {
		t.Fatal(err)
	}
	var out State
	if err := proto.Unmarshal(b, &out); err != nil {
		t.Fatal(err)
	}
	if len(out.Gpu) != 1 || len(out.Gpus) != 1 || out.Gpus[0].MemoryTotal != 1024 {
		t.Fatal("GPU wire roundtrip failed")
	}
	if ShenxianService_ServiceDesc.ServiceName != "proto.ShenxianService" {
		t.Fatal("RPC brand mismatch")
	}
}
