/*
Copyright 2023 The olive Authors

This program is offered under a commercial and under the AGPL license.
For AGPL licensing, see below.

AGPL licensing:
This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

package scheduler

//
//var (
//	r1 = &pb.Runner{
//		Id:              1,
//		ListenPeerURL:   "http://127.0.0.1:15280",
//		ListenClientURL: "http://127.0.0.1:15239",
//		HeartbeatMs:     2000,
//		Hostname:        "r1",
//		Cpu:             4 * 2500,
//		Memory:          16 * 1024 * 1024 * 1024,
//		Version:         "0.1.0",
//	}
//	r2 = &pb.Runner{
//		Id:              2,
//		ListenPeerURL:   "http://127.0.0.1:25280",
//		ListenClientURL: "http://127.0.0.1:25239",
//		HeartbeatMs:     2000,
//		Hostname:        "r2",
//		Cpu:             4 * 2500,
//		Memory:          16 * 1024 * 1024 * 1024,
//		Version:         "0.1.0",
//	}
//	r3 = &pb.Runner{
//		Id:              3,
//		ListenPeerURL:   "http://127.0.0.1:35280",
//		ListenClientURL: "http://127.0.0.1:35239",
//		HeartbeatMs:     2000,
//		Hostname:        "r3",
//		Cpu:             4 * 2500,
//		Memory:          16 * 1024 * 1024 * 1024,
//		Version:         "0.1.0",
//	}
//)
//
//func randInt(n int) int {
//	rn := rand.New(rand.NewSource(time.Now().UnixNano()))
//	return rn.Intn(n)
//}
//
//func newScheduler(t *testing.T) (*Scheduler, *clientv3.Client, func()) {
//	cfg := embed.NewConfig()
//	cfg.Dir = "testdata"
//	peerPort := 11000 + randInt(500)
//	peerURL, _ := url.Parse("http://localhost:" + fmt.Sprintf("%d", peerPort))
//	cfg.ListenPeerUrls = []url.URL{*peerURL}
//	clientPort := 12000 + randInt(500)
//	clientURL, _ := url.Parse("http://localhost:" + fmt.Sprintf("%d", clientPort))
//	cfg.ListenClientUrls = []url.URL{*clientURL}
//	etcd, err := embed.StartEtcd(cfg)
//	if !assert.NoError(t, err) {
//		return nil, nil, nil
//	}
//
//	<-etcd.Server.ReadyNotify()
//
//	ctx := context.Background()
//	lg := zap.NewExample()
//	client := v3client.New(etcd.Server)
//	notifier := leader.NewNotify(etcd.Server)
//	stopping := make(chan struct{}, 1)
//	limit := Limit{
//		RegionLimit:     50,
//		DefinitionLimit: 100,
//	}
//	cancel := func() {
//		close(stopping)
//		etcd.Server.HardStop()
//		<-etcd.Server.StopNotify()
//		_ = os.RemoveAll(cfg.Dir)
//	}
//	scheduler := New(ctx, lg, client, notifier, limit, stopping)
//	return scheduler, client, cancel
//}
//
//func injectRunners(t *testing.T, client *clientv3.Client, n int) {
//	ctx := context.TODO()
//	if n > 0 {
//		key := path.Join(runtime.DefaultMetaRunnerRegistrar, fmt.Sprintf("%d", r1.Id))
//		data, _ := proto.Marshal(r1)
//		_, err := client.Put(ctx, key, string(data))
//		if !assert.NoError(t, err) {
//			return
//		}
//	}
//
//	if n > 1 {
//		key := path.Join(runtime.DefaultMetaRunnerRegistrar, fmt.Sprintf("%d", r2.Id))
//		data, _ := proto.Marshal(r2)
//		_, err := client.Put(ctx, key, string(data))
//		if !assert.NoError(t, err) {
//			return
//		}
//	}
//
//	if n > 2 {
//		key := path.Join(runtime.DefaultMetaRunnerRegistrar, fmt.Sprintf("%d", r3.Id))
//		data, _ := proto.Marshal(r3)
//		_, err := client.Put(ctx, key, string(data))
//		if !assert.NoError(t, err) {
//			return
//		}
//	}
//}
//
//func runnerHeartbeat(t *testing.T, client *clientv3.Client, runner *pb.Runner) {
//	stat := &pb.RunnerStat{
//		Id:        runner.Id,
//		CpuPer:    float64(randInt(50)),
//		MemoryPer: float64(randInt(70)),
//		Timestamp: time.Now().Unix(),
//	}
//
//	ctx := context.Background()
//	data, _ := proto.Marshal(stat)
//	key := path.Join(runtime.DefaultMetaRunnerStat, fmt.Sprintf("%d", stat.Id))
//	_, err := client.Put(ctx, key, string(data))
//	if !assert.NoError(t, err) {
//		return
//	}
//}
//
//func regionHeartbeat(t *testing.T, client *clientv3.Client, region *pb.Region) {
//	stat := &pb.RegionStat{
//		Id:          region.Id,
//		Leader:      region.Leader,
//		Replicas:    int32(len(region.Replicas)),
//		Definitions: uint64(randInt(100)),
//		Timestamp:   time.Now().Unix(),
//	}
//
//	ctx := context.Background()
//	data, _ := proto.Marshal(stat)
//	key := path.Join(runtime.DefaultMetaRegionStat, fmt.Sprintf("%d", stat.Id))
//	_, err := client.Put(ctx, key, string(data))
//	if !assert.NoError(t, err) {
//		return
//	}
//}
//
//func TestScheduler_New(t *testing.T) {
//	sc, _, cancel := newScheduler(t)
//	defer cancel()
//	err := sc.Start()
//	if !assert.NoError(t, err) {
//		return
//	}
//}
//
//func TestScheduler_AllocRegion_one_replica(t *testing.T) {
//	sc, client, cancel := newScheduler(t)
//	defer cancel()
//
//	injectRunners(t, client, 1)
//
//	err := sc.Start()
//	if !assert.NoError(t, err) {
//		return
//	}
//
//	runnerHeartbeat(t, client, r1)
//
//	time.Sleep(time.Second)
//
//	ctx := context.Background()
//	region, err := sc.AllocRegion(ctx)
//	if !assert.NoError(t, err) {
//		return
//	}
//
//	t.Logf("%+v\n", region)
//}
//
//func TestScheduler_AllocRegion_three_replicas(t *testing.T) {
//	sc, client, cancel := newScheduler(t)
//	defer cancel()
//
//	injectRunners(t, client, 3)
//
//	err := sc.Start()
//	if !assert.NoError(t, err) {
//		return
//	}
//
//	runnerHeartbeat(t, client, r1)
//	runnerHeartbeat(t, client, r2)
//	runnerHeartbeat(t, client, r3)
//
//	time.Sleep(time.Second)
//
//	ctx := context.Background()
//	region, err := sc.AllocRegion(ctx)
//	if !assert.NoError(t, err) {
//		return
//	}
//
//	t.Logf("%+v\n", region)
//}
//
//func TestScheduler_BindRegion(t *testing.T) {
//	sc, client, cancel := newScheduler(t)
//	defer cancel()
//
//	injectRunners(t, client, 3)
//
//	err := sc.Start()
//	if !assert.NoError(t, err) {
//		return
//	}
//
//	runnerHeartbeat(t, client, r1)
//	runnerHeartbeat(t, client, r2)
//	runnerHeartbeat(t, client, r3)
//
//	time.Sleep(time.Second)
//
//	ctx := context.Background()
//	region, err := sc.AllocRegion(ctx)
//	if !assert.NoError(t, err) {
//		return
//	}
//
//	regionHeartbeat(t, client, region)
//	time.Sleep(time.Second)
//
//	dm := &pb.DefinitionMeta{
//		Id:      "xx",
//		Version: 1,
//		Region:  0,
//	}
//	region, has, err := sc.BindRegion(ctx, dm)
//	if !assert.True(t, has) {
//		return
//	}
//	if !assert.NoError(t, err) {
//		return
//	}
//	t.Logf("%+v\n", dm)
//}
