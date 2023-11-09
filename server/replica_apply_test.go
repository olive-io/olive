package server

//func TestReadonlyTxnError(t *testing.T) {
//	b, _ := betesting.NewDefaultTmpBackend(t)
//	defer betesting.Close(t, b)
//	s := mvcc.NewStore(zap.NewExample(), b, mvcc.StoreConfig{})
//	defer s.Close()
//
//	// setup minimal server to get access to applier
//	srv := &KVServer{lgMu: new(sync.RWMutex), lg: zap.NewExample(), r: *newRaftNode(raftNodeConfig{lg: zap.NewExample(), Node: newNodeRecorder()})}
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
//	txn := &pb.TxnRequest{
//		Success: []*pb.RequestOp{
//			{
//				Request: &pb.RequestOp_RequestRange{
//					RequestRange: &pb.RangeRequest{
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
//	srv := &KVServer{lgMu: new(sync.RWMutex), lg: zap.NewExample(), r: *newRaftNode(raftNodeConfig{lg: zap.NewExample(), Node: newNodeRecorder()})}
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
//	txn := &pb.TxnRequest{
//		Success: []*pb.RequestOp{
//			{
//				Request: &pb.RequestOp_RequestPut{
//					RequestPut: &pb.PutRequest{
//						Key:   []byte("foo"),
//						Value: []byte("bar"),
//					},
//				},
//			},
//			{
//				Request: &pb.RequestOp_RequestRange{
//					RequestRange: &pb.RangeRequest{
//						Key: []byte("foo"),
//					},
//				},
//			},
//		},
//	}
//
//	assert.Panics(t, func() { a.Txn(ctx, txn) }, "Expected panic in Txn with writes")
//}
