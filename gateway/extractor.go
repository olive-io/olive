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

	json "github.com/json-iterator/go"

	dsypb "github.com/olive-io/olive/api/pb/discovery"
	"github.com/olive-io/olive/pkg/proxy/api"
	"github.com/olive-io/olive/pkg/tonic/openapi"
)

func extractOpenAPIDocs(svc *openapi.OpenAPI) []*dsypb.Endpoint {
	eps := make([]*dsypb.Endpoint, 0)

	for url, pt := range svc.Paths {
		var method string
		var ep *openapi.Operation
		if pt.GET != nil {
			method = http.MethodGet
			ep = pt.GET
		} else if pt.POST != nil {
			method = http.MethodPost
			ep = pt.POST
		} else if pt.PUT != nil {
			method = http.MethodPut
			ep = pt.PUT
		} else if pt.Parameters != nil {
			method = http.MethodPatch
			ep = pt.PATCH
		} else if pt.DELETE != nil {
			method = http.MethodDelete
			ep = pt.DELETE
		} else {
			continue
		}

		name := ep.ID
		if name == "" {
			name = url
		}

		hosts := make([]string, 0)
		for _, server := range svc.Servers {
			hosts = append(hosts, server.URL)
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
			if mt, ok := content["application/json"]; ok {
				extractSchema(mt.Schema, svc.Components, req)
				md[api.ContentTypeKey] = "application/json"
			}
			if mt, ok := content["application/xml"]; ok {
				extractSchema(mt.Schema, svc.Components, req)
				md[api.ContentTypeKey] = "application/xml"
			}
			if mt, ok := content["application/yaml"]; ok {
				extractSchema(mt.Schema, svc.Components, req)
				md[api.ContentTypeKey] = "application/yaml"
			}
		}
		rsp := &dsypb.Box{Type: dsypb.BoxType_object}
		if resp, ok := ep.Responses["200"]; ok {
			content := resp.Content
			if mt, ok := content["application/json"]; ok {
				extractSchema(mt.Schema, svc.Components, req)
			}
			if mt, ok := content["application/xml"]; ok {
				extractSchema(mt.Schema, svc.Components, req)
			}
			if mt, ok := content["application/yaml"]; ok {
				extractSchema(mt.Schema, svc.Components, req)
			}
		}

		if len(ep.Security) != 0 {
			security := ep.Security[0]
			for key, values := range *security {
				securityItem := strings.Join(values, ",")
				securityText := key + "::" + securityItem
				md[api.SecurityKey] = securityText
			}
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

func extractSchema(so *openapi.SchemaOrRef, components *openapi.Components, target *dsypb.Box) {
	if schema := so.Schema; schema != nil {
		if schema.Example != nil {
			data, _ := json.Marshal(so.Example)
			target.Data = string(data)
		}

		switch schema.Type {
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
			if schema.AdditionalProperties != nil {
				target.Type = dsypb.BoxType_map
				ap := schema.AdditionalProperties
				if ap.Schema != nil {
					target.Ref = ap.Schema.Type
				}
				if ap.Reference != nil {
					target.Ref = ap.Reference.Ref
				}
				return
			}
		case "array":
			extractSchema(schema.Items, components, target)
			if target.Ref == "" {
				target.Ref = schema.Type
			}
			target.Type = dsypb.BoxType_array
			return
		}
	}

	if reference := so.Reference; reference != nil {
		ref := strings.TrimPrefix(reference.Ref, "#/components/schemas/")
		model, ok := components.Schemas[ref]
		if !ok {
			return
		}
		target.Ref = reference.Ref

		parameters := map[string]*dsypb.Box{}
		for name := range model.Properties {
			param := &dsypb.Box{}
			extractSchema(model.Properties[name], components, param)
			parameters[name] = param
		}
		target.Parameters = parameters
	}
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
