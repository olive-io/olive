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
	"strings"

	pb "github.com/olive-io/olive/api/discoverypb"
	"github.com/olive-io/olive/pkg/proxy/api"
)

func extractOpenAPIDocs(svc *pb.OpenAPI) []*pb.Endpoint {
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

		name := apiEp.OperationId
		if name == "" {
			name = url
		}

		hosts := make([]string, 0)
		for _, server := range svc.Servers {
			hosts = append(hosts, server.Url)
		}
		md := map[string]string{
			api.MethodKey:  method,
			api.HostKey:    strings.Join(hosts, ","),
			api.URLKey:     url,
			api.HandlerKey: api.RPCHandler,
			"summary":      apiEp.Summary,
			api.DescKey:    apiEp.Description,
		}

		req := &pb.Box{
			Type: pb.BoxType_object,
		}
		if method == http.MethodGet {
			md[api.ContentTypeKey] = "application/json"
			parameters := map[string]*pb.Box{}
			for _, param := range apiEp.Parameters {
				item := &pb.Box{}
				extractSchema(param.Schema, svc.Components, item)
				parameters[param.Name] = item
			}
			req.Parameters = parameters
		} else {
			content := apiEp.RequestBody.Content
			if content.ApplicationJson != nil {
				extractSchema(content.ApplicationJson.Schema, svc.Components, req)
				md[api.ContentTypeKey] = "application/json"
			}
			if content.ApplicationXml != nil {
				extractSchema(content.ApplicationXml.Schema, svc.Components, req)
				md[api.ContentTypeKey] = "application/xml"
			}
			if content.ApplicationYaml != nil {
				extractSchema(content.ApplicationYaml.Schema, svc.Components, req)
				md[api.ContentTypeKey] = "application/yaml"
			}
		}
		rsp := &pb.Box{Type: pb.BoxType_object}
		if resp, ok := apiEp.Responses["200"]; ok {
			content := resp.Content
			if content.ApplicationJson != nil {
				extractSchema(content.ApplicationJson.Schema, svc.Components, rsp)
			}
			if content.ApplicationXml != nil {
				extractSchema(content.ApplicationXml.Schema, svc.Components, rsp)
			}
			if content.ApplicationYaml != nil {
				extractSchema(content.ApplicationYaml.Schema, svc.Components, rsp)
			}
		}

		for key, value := range apiEp.Metadata {
			md[key] = value
		}

		for _, item := range apiEp.Security {
			if item.Basic != nil {
				md[api.SecurityKey] = strings.Join(append([]string{"basic"}, strings.Join(item.Basic, ",")), "::")
			} else if item.ApiKeys != nil {
				md[api.SecurityKey] = strings.Join(append([]string{"api_keys"}, strings.Join(item.ApiKeys, ",")), "::")
			} else if item.Bearer != nil {
				md[api.SecurityKey] = strings.Join(append([]string{"bearer"}, strings.Join(item.Basic, ",")), "::")
			} else if item.OAuth2 != nil {
				md[api.SecurityKey] = strings.Join(append([]string{"oauth2"}, strings.Join(item.OAuth2, ",")), "::")
			} else if item.OpenId != nil {
				md[api.SecurityKey] = strings.Join(append([]string{"openId"}, strings.Join(item.OpenId, ",")), "::")
			} else if item.CookieAuth != nil {
				md[api.SecurityKey] = strings.Join(append([]string{"cookieAuth"}, strings.Join(item.CookieAuth, ",")), "::")
			}
		}

		ep := &pb.Endpoint{
			Name:     name,
			Request:  req,
			Response: rsp,
			Metadata: md,
		}

		eps = append(eps, ep)
	}

	return eps
}

func extractSchema(so *pb.SchemaObject, components *pb.OpenAPIComponents, target *pb.Box) {
	if len(so.Example) != 0 {
		target.Data = []byte(so.Example)
	}

	switch so.Type {
	case "string":
		target.Type = pb.BoxType_string
		return
	case "integer":
		target.Type = pb.BoxType_integer
		return
	case "number":
		target.Type = pb.BoxType_float
		return
	case "boolean":
		target.Type = pb.BoxType_boolean
		return
	case "object":
		target.Type = pb.BoxType_object
		if so.AdditionalProperties != nil {
			target.Type = pb.BoxType_map
			target.Ref = so.AdditionalProperties.Ref
			return
		}
	case "array":
		extractSchema(so.Items, components, target)
		if target.Ref == "" {
			target.Ref = so.Type
		}
		target.Type = pb.BoxType_array
		return
	}

	ref := strings.TrimPrefix(so.Ref, "#/components/schemas/")
	model, ok := components.Schemas[ref]
	if !ok {
		return
	}
	target.Ref = so.Ref

	parameters := map[string]*pb.Box{}
	for name := range model.Properties {
		param := &pb.Box{}
		extractSchema(model.Properties[name], components, param)
		parameters[name] = param
	}
	target.Parameters = parameters
}

//func extractEndpoint(method reflect.Method) *pb.Endpoint {
//	if method.PkgPath != "" {
//		return nil
//	}
//
//	var rspType, reqType reflect.Type
//	var stream bool
//	mt := method.Type
//
//	in, out := mt.NumIn(), mt.NumOut()
//	if in == 3 && out == 2 {
//		reqType = mt.In(2)
//		rspType = mt.Out(0)
//	} else if in == 3 && out == 1 {
//		reqType = mt.In(1)
//		rspType = mt.In(2)
//	} else if in == 2 && out == 1 {
//		reqType = mt.In(1)
//		rspType = mt.In(1)
//	} else {
//		panic("invalid grpc endpoint")
//	}
//
//	// are we dealing with a stream?
//	switch rspType.Kind() {
//	case reflect.Func, reflect.Interface:
//		stream = true
//	default:
//	}
//
//	request := extractBox(reqType, 0)
//	response := extractBox(rspType, 0)
//
//	ep := &pb.Endpoint{
//		Name:     method.Name,
//		Request:  request,
//		Response: response,
//		Metadata: make(map[string]string),
//	}
//
//	if stream {
//		if _, exists := ep.Metadata["stream"]; !exists {
//			ep.Metadata["stream"] = fmt.Sprintf("%v", stream)
//		}
//	}
//
//	return ep
//}
//
//func extractBox(v reflect.Type, d int) *pb.Box {
//	if d == 3 {
//		return nil
//	}
//	if v == nil {
//		return nil
//	}
//
//	if v.Kind() == reflect.Ptr {
//		v = v.Elem()
//	}
//
//	name := v.Name()
//	pkgPath := v.PkgPath()
//	_ = pkgPath
//	_ = name
//
//	return &pb.Box{}
//}
