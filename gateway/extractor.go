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

package gateway

import (
	"net/http"
	"strings"

	dsypb "github.com/olive-io/olive/api/discoverypb"
	"github.com/olive-io/olive/x/proxy/api"
)

func extractOpenAPIDocs(svc *dsypb.OpenAPI) []*dsypb.Endpoint {
	eps := make([]*dsypb.Endpoint, 0)

	for url, pt := range svc.Paths {
		var method string
		var ep *dsypb.OpenAPIEndpoint
		if pt.Get != nil {
			method = http.MethodGet
			ep = pt.Get
		} else if pt.Post != nil {
			method = http.MethodPost
			ep = pt.Post
		} else if pt.Put != nil {
			method = http.MethodPut
			ep = pt.Put
		} else if pt.Patch != nil {
			method = http.MethodPatch
			ep = pt.Patch
		} else if pt.Delete != nil {
			method = http.MethodDelete
			ep = pt.Delete
		} else {
			continue
		}

		name := ep.OperationId
		if name == "" {
			name = url
		}

		hosts := make([]string, 0)
		for _, server := range svc.Servers {
			hosts = append(hosts, server.Url)
		}
		md := map[string]string{
			api.EndpointKey: name,
			api.MethodKey:   method,
			api.HostKey:     strings.Join(hosts, ","),
			api.URLKey:      url,
			api.HandlerKey:  api.RPCHandler,
			"summary":       ep.Summary,
			api.DescKey:     ep.Description,
		}

		req := &dsypb.Box{
			Type: dsypb.BoxType_object,
		}
		if method == http.MethodGet {
			md[api.ContentTypeKey] = "application/json"
			parameters := map[string]*dsypb.Box{}
			for _, param := range ep.Parameters {
				item := &dsypb.Box{}
				extractSchema(param.Schema, svc.Components, item)
				parameters[param.Name] = item
			}
			req.Parameters = parameters
		} else {
			content := ep.RequestBody.Content
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
		rsp := &dsypb.Box{Type: dsypb.BoxType_object}
		if resp, ok := ep.Responses["200"]; ok {
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

		for _, item := range ep.Security {
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

		for key, value := range ep.Metadata {
			md[key] = value
		}

		eps = append(eps, &dsypb.Endpoint{
			Name:     name,
			Request:  req,
			Response: rsp,
			Metadata: md,
		})
	}

	return eps
}

func extractSchema(so *dsypb.SchemaObject, components *dsypb.OpenAPIComponents, target *dsypb.Box) {
	if len(so.Example) != 0 {
		target.Data = []byte(so.Example)
	}

	switch so.Type {
	case "string":
		target.Type = dsypb.BoxType_string
		return
	case "integer":
		target.Type = dsypb.BoxType_integer
		return
	case "number":
		target.Type = dsypb.BoxType_float
		return
	case "boolean":
		target.Type = dsypb.BoxType_boolean
		return
	case "object":
		target.Type = dsypb.BoxType_object
		if so.AdditionalProperties != nil {
			target.Type = dsypb.BoxType_map
			target.Ref = so.AdditionalProperties.Ref
			return
		}
	case "array":
		extractSchema(so.Items, components, target)
		if target.Ref == "" {
			target.Ref = so.Type
		}
		target.Type = dsypb.BoxType_array
		return
	}

	ref := strings.TrimPrefix(so.Ref, "#/components/schemas/")
	model, ok := components.Schemas[ref]
	if !ok {
		return
	}
	target.Ref = so.Ref

	parameters := map[string]*dsypb.Box{}
	for name := range model.Properties {
		param := &dsypb.Box{}
		extractSchema(model.Properties[name], components, param)
		parameters[name] = param
	}
	target.Parameters = parameters
}

func extractConsumer(idt *dsypb.Consumer, service string) *dsypb.Endpoint {
	md := idt.Metadata
	if md == nil {
		md = map[string]string{}
	}

	md[api.EndpointKey] = api.DefaultTaskURL
	md[api.HandlerKey] = api.RPCHandler
	if _, ok := md[api.ContentTypeKey]; !ok {
		md[api.ContentTypeKey] = "application/json"
	}
	md[api.URLKey] = api.DefaultTaskURL
	md[api.ActivityKey] = idt.Activity.String()
	md[api.TaskTypeKey] = idt.Action
	md["service"] = service
	md["id"] = idt.Id

	ep := &dsypb.Endpoint{
		Name:     idt.Id,
		Request:  idt.Request,
		Response: idt.Response,
		Metadata: md,
	}
	return ep
}
