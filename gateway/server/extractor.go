// Copyright 2024 The olive Authors
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

import (
	"net/http"

	pb "github.com/olive-io/olive/api/discoverypb"
	"github.com/olive-io/olive/pkg/proxy/api"
)

func extractEndpoints(svc *pb.OpenAPI) []*pb.Endpoint {
	eps := make([]*pb.Endpoint, 0)

	for url, pt := range svc.Paths {
		var method string
		var apiEp *pb.OpenAPIEndpoint
		if pt.Get != nil {
			method = http.MethodGet
			apiEp = pt.Get
		} else if pt.Post != nil {
			method = http.MethodPost
			apiEp = pt.Post
		} else if pt.Put != nil {
			method = http.MethodPut
			apiEp = pt.Put
		} else if pt.Patch != nil {
			method = http.MethodPatch
			apiEp = pt.Patch
		} else if pt.Delete != nil {
			method = http.MethodDelete
			apiEp = pt.Delete
		} else {
			continue
		}

		req := &pb.Box{
			Type: pb.BoxType_Object,
		}
		if method == http.MethodGet {
			//for _, param := range apiEp.Parameters {
			//
			//}
		}
		rsp := &pb.Box{}

		ep := &pb.Endpoint{
			Name:     url,
			Request:  req,
			Response: rsp,
			Metadata: map[string]string{
				api.MethodKey: method,
			},
		}

		for key, value := range apiEp.Metadata {
			ep.Metadata[key] = value
		}

		eps = append(eps, ep)
	}

	return eps
}
