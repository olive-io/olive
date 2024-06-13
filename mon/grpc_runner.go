/*
Copyright 2024 The olive Authors

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

package mon

//
//type runnerServer struct {
//	pb.UnsafeMetaRunnerRPCServer
//	pb.UnsafeMetaRegionRPCServer
//
//	*MonitorServer
//}
//
//func newRunnerServer(s *MonitorServer) (*runnerServer, error) {
//	rs := &runnerServer{MonitorServer: s}
//	return rs, nil
//}
//
//func (s *runnerServer) ListRunner(ctx context.Context, req *pb.ListRunnerRequest) (resp *pb.ListRunnerResponse, err error) {
//
//	lg := s.lg
//	resp = &pb.ListRunnerResponse{}
//	key := ort.DefaultMetaRunnerRegistrar
//
//	runners := make([]*pb.Runner, 0)
//	var continueToken string
//	continueToken, err = s.pageList(ctx, key, req.Limit, req.Continue, func(kv *mvccpb.KeyValue) error {
//		runner := new(pb.Runner)
//		if e1 := proto.Unmarshal(kv.Value, runner); e1 != nil {
//			lg.Error("unmarshal runner", zap.String("key", string(kv.Key)), zap.Error(err))
//			return e1
//		}
//		runners = append(runners, runner)
//
//		return nil
//	})
//	if err != nil {
//		return
//	}
//
//	resp.Header = s.responseHeader()
//	resp.Runners = runners
//	resp.ContinueToken = continueToken
//
//	return resp, nil
//}
//
//func (s *runnerServer) GetRunner(ctx context.Context, req *pb.GetRunnerRequest) (resp *pb.GetRunnerResponse, err error) {
//
//	resp = &pb.GetRunnerResponse{}
//	resp.Header = s.responseHeader()
//	resp.Runner, err = s.getRunner(ctx, req.Id)
//	return
//}
//
//func (s *runnerServer) ListRegion(ctx context.Context, req *pb.ListRegionRequest) (resp *pb.ListRegionResponse, err error) {
//
//	lg := s.lg
//	resp = &pb.ListRegionResponse{}
//	key := ort.DefaultRunnerRegion
//
//	regions := make([]*pb.Region, 0)
//	var continueToken string
//	continueToken, err = s.pageList(ctx, key, req.Limit, req.Continue, func(kv *mvccpb.KeyValue) error {
//		region := new(pb.Region)
//		if e1 := proto.Unmarshal(kv.Value, region); e1 != nil {
//			lg.Error("unmarshal region", zap.String("key", string(kv.Key)), zap.Error(err))
//			return e1
//		}
//		regions = append(regions, region)
//
//		return nil
//	})
//	if err != nil {
//		return
//	}
//
//	resp.Header = s.responseHeader()
//	resp.Regions = regions
//	resp.ContinueToken = continueToken
//
//	return resp, nil
//}
//
//func (s *runnerServer) GetRegion(ctx context.Context, req *pb.GetRegionRequest) (resp *pb.GetRegionResponse, err error) {
//
//	resp = &pb.GetRegionResponse{}
//	resp.Header = s.responseHeader()
//	resp.Region, err = s.getRegion(ctx, req.Id)
//	return
//}
