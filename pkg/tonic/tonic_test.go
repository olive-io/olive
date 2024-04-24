package tonic_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/olive-io/olive/pkg/tonic"
)

var r http.Handler

func errorHook(c *gin.Context, e error) (int, interface{}) {
	var bindError tonic.BindError
	if errors.As(e, &bindError) {
		return 400, e.Error()
	}
	return 500, e.Error()
}

func TestMain(m *testing.M) {

	tonic.SetErrorHook(errorHook)

	g := gin.Default()
	g.GET("/simple", tonic.Handler(simpleHandler, 200))
	g.GET("/scalar", tonic.Handler(scalarHandler, 200))
	g.GET("/error", tonic.Handler(errorHandler, 200))
	g.GET("/path/:param", tonic.Handler(pathHandler, 200))
	g.GET("/query", tonic.Handler(queryHandler, 200))
	g.GET("/query-old", tonic.Handler(queryHandlerOld, 200))
	g.POST("/body", tonic.Handler(bodyHandler, 200))

	r = g

	m.Run()
}

func errorHandler(c *gin.Context) error {
	return errors.New("error")
}

func simpleHandler(c *gin.Context) error {
	return nil
}

func scalarHandler(c *gin.Context) (string, error) {
	return "", nil
}

type pathIn struct {
	Param string `path:"param" json:"param"`
}

func pathHandler(c *gin.Context, in *pathIn) (*pathIn, error) {
	return in, nil
}

type queryIn struct {
	Param                       string    `query:"param" json:"param" validate:"required"`
	ParamOptional               string    `query:"param-optional" json:"param-optional"`
	Params                      []string  `query:"params" json:"params"`
	ParamInt                    int       `query:"param-int" json:"param-int"`
	ParamBool                   bool      `query:"param-bool" json:"param-bool"`
	ParamDefault                string    `query:"param-default" json:"param-default" default:"default" validate:"required"`
	ParamPtr                    *string   `query:"param-ptr" json:"param-ptr"`
	ParamComplex                time.Time `query:"param-complex" json:"param-complex"`
	ParamExplode                []string  `query:"param-explode" json:"param-explode" explode:"true"`
	ParamExplodeDisabled        []string  `query:"param-explode-disabled" json:"param-explode-disabled" explode:"false"`
	ParamExplodeString          string    `query:"param-explode-string" json:"param-explode-string" explode:"true"`
	ParamExplodeDefault         []string  `query:"param-explode-default" json:"param-explode-default" default:"1,2,3" explode:"true"`
	ParamExplodeDefaultDisabled []string  `query:"param-explode-disabled-default" json:"param-explode-disabled-default" default:"1,2,3" explode:"false"`
	*DoubleEmbedded
}

// XXX: deprecated, but ensure backwards compatibility
type queryInOld struct {
	ParamRequired string `query:"param, required" json:"param"`
	ParamDefault  string `query:"param-default,required,default=default" json:"param-default"`
}

type Embedded struct {
	ParamEmbed string `query:"param-embed" json:"param-embed"`
}

type DoubleEmbedded struct {
	Embedded
}

func queryHandler(c *gin.Context, in *queryIn) (*queryIn, error) {
	return in, nil
}

func queryHandlerOld(c *gin.Context, in *queryInOld) (*queryInOld, error) {
	return in, nil
}

type bodyIn struct {
	Param                  string `json:"param" validate:"required"`
	ParamOptional          string `json:"param-optional"`
	ValidatedParamOptional string `json:"param-optional-validated" validate:"eq=|eq=foo|gt=10"`
}

func bodyHandler(c *gin.Context, in *bodyIn) (*bodyIn, error) {
	return in, nil
}

func expectEmptyBody(r *http.Response, body string, obj interface{}) error {
	if len(body) != 0 {
		return fmt.Errorf("Body '%s' should be empty", body)
	}
	return nil
}

func expectString(paramName, value string) func(*http.Response, string, interface{}) error {

	return func(r *http.Response, body string, obj interface{}) error {

		var i map[string]interface{}

		err := json.Unmarshal([]byte(body), &i)
		if err != nil {
			return err
		}
		s, ok := i[paramName]
		if !ok {
			return fmt.Errorf("%s missing", paramName)
		}
		if s != value {
			return fmt.Errorf("%s: expected %s got %s", paramName, value, s)
		}
		return nil
	}
}

func expectBool(paramName string, value bool) func(*http.Response, string, interface{}) error {

	return func(r *http.Response, body string, obj interface{}) error {

		i := map[string]interface{}{paramName: 0}

		err := json.Unmarshal([]byte(body), &i)
		if err != nil {
			return err
		}
		v, ok := i[paramName]
		if !ok {
			return fmt.Errorf("%s missing", paramName)
		}
		vb, ok := v.(bool)
		if !ok {
			return fmt.Errorf("%s not a number", paramName)
		}
		if vb != value {
			return fmt.Errorf("%s: expected %v got %v", paramName, value, vb)
		}
		return nil
	}
}

func expectStringArr(paramName string, value ...string) func(*http.Response, string, interface{}) error {

	return func(r *http.Response, body string, obj interface{}) error {

		var i map[string]interface{}

		err := json.Unmarshal([]byte(body), &i)
		if err != nil {
			return fmt.Errorf("failed to unmarshal json: %s", body)
		}
		s, ok := i[paramName]
		if !ok {
			return fmt.Errorf("%s missing", paramName)
		}
		sArr, ok := s.([]interface{})
		if !ok {
			return fmt.Errorf("%s not a string arr", paramName)
		}
		for n := 0; n < len(value); n++ {
			if n >= len(sArr) {
				return fmt.Errorf("%s too short", paramName)
			}
			if sArr[n] != value[n] {
				return fmt.Errorf("%s: %s does not match", paramName, sArr[n])
			}
		}
		return nil
	}
}

func expectStringInBody(value string) func(*http.Response, string, interface{}) error {

	return func(r *http.Response, body string, obj interface{}) error {
		if !strings.Contains(body, value) {
			return fmt.Errorf("body doesn't contain '%s' (%s)", value, body)
		}
		return nil
	}
}

func expectInt(paramName string, value int) func(*http.Response, string, interface{}) error {

	return func(r *http.Response, body string, obj interface{}) error {

		i := map[string]interface{}{paramName: 0}

		err := json.Unmarshal([]byte(body), &i)
		if err != nil {
			return err
		}
		v, ok := i[paramName]
		if !ok {
			return fmt.Errorf("%s missing", paramName)
		}
		vf, ok := v.(float64)
		if !ok {
			return fmt.Errorf("%s not a number", paramName)
		}
		vInt := int(vf)
		if vInt != value {
			return fmt.Errorf("%s: expected %v got %v", paramName, value, vInt)
		}
		return nil
	}
}
