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

package runner

//func parseRegionKV(kv *mvccpb.KeyValue, runner string) (*corev1.Region, bool, error) {
//	region := new(corev1.Region)
//	err := region.Unmarshal(kv.Value)
//	if err != nil {
//		return nil, false, err
//	}
//	match := false
//	for _, replica := range region.Spec.Replicas {
//		if replica.Runner == runner {
//			match = true
//			break
//		}
//	}
//	return region, match, nil
//}
//
//func parseDefinitionKV(kv *mvccpb.KeyValue) (*corev1.Definition, bool, error) {
//	definition := new(pb.Definition)
//	err := proto.Unmarshal(kv.Value, definition)
//	if err != nil {
//		return nil, false, err
//	}
//	if definition.Region == 0 {
//		return definition, false, nil
//	}
//	if definition.Rev == 0 {
//		definition.Rev = kv.ModRevision
//	}
//	return definition, true, nil
//}
//
//func parseProcessInstanceKV(kv *mvccpb.KeyValue) (*pb.ProcessInstance, bool, error) {
//	process := new(pb.ProcessInstance)
//	err := proto.Unmarshal(kv.Value, process)
//	if err != nil {
//		return nil, false, err
//	}
//	if process.Status != pb.ProcessInstance_Waiting ||
//		process.DefinitionsId == "" ||
//		process.Region == 0 {
//		return process, false, nil
//	}
//	if process.Rev == 0 {
//		process.Rev = kv.ModRevision
//	}
//	return process, true, nil
//}
//
//func commitProcessInstance(ctx context.Context, lg *zap.Logger, client *clientgo.Client, process *pb.ProcessInstance) {
//	if lg == nil {
//		lg = zap.NewNop()
//	}
//
//	if process.Status == pb.ProcessInstance_Waiting {
//		process.Status = pb.ProcessInstance_Prepare
//	}
//	key := path.Join(ort.DefaultRunnerProcessInstance,
//		process.DefinitionsId, fmt.Sprintf("%d", process.DefinitionsVersion),
//		process.Id)
//	data, _ := proto.Marshal(process)
//	_, err := client.Put(ctx, key, string(data))
//	if err != nil {
//		lg.Error("update process instance",
//			zap.String("definition", process.DefinitionsId),
//			zap.Uint64("version", process.DefinitionsVersion),
//			zap.String("id", process.Id),
//			zap.Error(err))
//		return
//	}
//}
//
//func sliceEqual[T string | int64](a, b []T) bool {
//	if len(a) != len(b) {
//		return false
//	}
//	sort.Slice(a, func(i, j int) bool {
//		return a[i] < a[j]
//	})
//	sort.Slice(b, func(i, j int) bool {
//		return b[i] < b[j]
//	})
//	for idx, v := range a {
//		if b[idx] != v {
//			return false
//		}
//	}
//	return true
//}
