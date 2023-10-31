// Copyright 2023 Lack (xingyys@gmail.com).
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package server

//func TestReadonlyTxnError(t *testing.T) {
//	b, _ := betesting.NewDefaultTmpBackend(t)
//	defer betesting.Close(t, b)
//	s := mvcc.NewStore(zap.NewExample(), b, mvcc.StoreConfig{})
//	defer s.Close()
//
//	// setup minimal server to get access to applier
//	srv := &OliveServer{lgMu: new(sync.RWMutex), lg: zap.NewExample(), r: *newRaftNode(raftNodeConfig{lg: zap.NewExample(), Node: newNodeRecorder()})}
//	srv.kv = s
//	srv.be = b
//
//	a := srv.newApplierBackend()
//
//	// setup cancelled context
//	ctx, cancel := context.WithCancel(context.TODO())
//	cancel()
//
//	// put some data to prevent early termination in rangeKeys
//	// we are expecting failure on cancelled context check
//	s.Put([]byte("foo"), []byte("bar"))
//
//	txn := &api.TxnRequest{
//		Success: []*api.RequestOp{
//			{
//				Request: &api.RequestOp_RequestRange{
//					RequestRange: &api.RangeRequest{
//						Key: []byte("foo"),
//					},
//				},
//			},
//		},
//	}
//
//	_, _, err := a.Txn(ctx, txn)
//	if err == nil || !strings.Contains(err.Error(), "applyTxn: failed Range: rangeKeys: context cancelled: context canceled") {
//		t.Fatalf("Expected context canceled error, got %v", err)
//	}
//}
//
//func TestWriteTxnPanic(t *testing.T) {
//	b, _ := betesting.NewDefaultTmpBackend(t)
//	defer betesting.Close(t, b)
//	s := mvcc.NewStore(zap.NewExample(), b, mvcc.StoreConfig{})
//	defer s.Close()
//
//	// setup minimal server to get access to applier
//	srv := &OliveServer{lgMu: new(sync.RWMutex), lg: zap.NewExample(), r: *newRaftNode(raftNodeConfig{lg: zap.NewExample(), Node: newNodeRecorder()})}
//	srv.kv = s
//	srv.be = b
//
//	a := srv.newApplierBackend()
//
//	// setup cancelled context
//	ctx, cancel := context.WithCancel(context.TODO())
//	cancel()
//
//	// write txn that puts some data and then fails in range due to cancelled context
//	txn := &api.TxnRequest{
//		Success: []*api.RequestOp{
//			{
//				Request: &api.RequestOp_RequestPut{
//					RequestPut: &api.PutRequest{
//						Key:   []byte("foo"),
//						Value: []byte("bar"),
//					},
//				},
//			},
//			{
//				Request: &api.RequestOp_RequestRange{
//					RequestRange: &api.RangeRequest{
//						Key: []byte("foo"),
//					},
//				},
//			},
//		},
//	}
//
//	assert.Panics(t, func() { a.Txn(ctx, txn) }, "Expected panic in Txn with writes")
//}
