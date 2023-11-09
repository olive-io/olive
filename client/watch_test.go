package client

import (
	"testing"

	pb "github.com/olive-io/olive/api/serverpb"
)

func TestEvent(t *testing.T) {
	tests := []struct {
		ev       *Event
		isCreate bool
		isModify bool
	}{{
		ev: &Event{
			Type: EventTypePut,
			Kv: &pb.KeyValue{
				CreateRevision: 3,
				ModRevision:    3,
			},
		},
		isCreate: true,
	}, {
		ev: &Event{
			Type: EventTypePut,
			Kv: &pb.KeyValue{
				CreateRevision: 3,
				ModRevision:    4,
			},
		},
		isModify: true,
	}}
	for i, tt := range tests {
		if tt.isCreate && !tt.ev.IsCreate() {
			t.Errorf("#%d: event should be Create event", i)
		}
		if tt.isModify && !tt.ev.IsModify() {
			t.Errorf("#%d: event should be Modify event", i)
		}
	}
}
