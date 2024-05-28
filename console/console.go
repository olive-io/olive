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

package console

import (
	"context"
	"fmt"
	"net"
	"net/http"
	urlpkg "net/url"
	"strings"
	"sync"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"

	"github.com/olive-io/olive/apis/version"
	"github.com/olive-io/olive/client"
	"github.com/olive-io/olive/console/config"
	"github.com/olive-io/olive/console/interceptor"
	"github.com/olive-io/olive/console/routes"
	genericdaemon "github.com/olive-io/olive/pkg/daemon"
	"github.com/olive-io/olive/pkg/tonic"
	"github.com/olive-io/olive/pkg/tonic/fizz"
	"github.com/olive-io/olive/pkg/tonic/openapi"
)

type Console struct {
	genericdaemon.IDaemon

	ctx    context.Context
	cancel context.CancelFunc

	cfg *config.Config

	oct *client.Client

	engine    *fizz.Fizz
	routeTree *routes.RouteTree

	serve *http.Server

	mu      sync.RWMutex
	started chan struct{}
}

func NewConsole(cfg *config.Config) (*Console, error) {
	lg := cfg.GetLogger()
	embedDaemon := genericdaemon.NewEmbedDaemon(lg)

	lg.Debug("connect to olive-mon",
		zap.String("endpoints", strings.Join(cfg.Client.Endpoints, ",")))

	mode := gin.ReleaseMode
	if lg.Level() == zap.DebugLevel {
		mode = gin.DebugMode
	}
	gin.SetMode(mode)
	engine := gin.New()

	if cfg.EnableOpenAPI {
		swaggerFS, err := GetSwaggerStatic()
		if err != nil {
			return nil, fmt.Errorf("read swagger static dist: %w", err)
		}
		engine.StaticFS("/swagger", http.FS(swaggerFS))
		engine.GET("/metrics", gin.WrapH(promhttp.Handler()))
	}
	fiz := fizz.NewFromEngine(engine)
	fiz.Use(gin.Recovery())

	apiInfo := &openapi.Info{
		Title:          "olive document",
		Description:    "This is the openapi v3 documentation of olive workflow engine.",
		TermsOfService: "The olive term",
		Contact: &openapi.Contact{
			Name:  "olive",
			Email: "xingyys@gmail.com",
		},
		License: &openapi.License{
			Name: "AGPL license",
			URL:  "https://www.gnu.org/licenses/",
		},
		Version: version.APIVersion,
		XLogo:   &openapi.XLogo{},
	}
	fiz.GET("/openapi.json", nil, fiz.OpenAPI(apiInfo, "json"))

	bearer := interceptor.NewBearerAuth()
	cfg.Client.Interceptor = bearer

	oct, err := client.New(cfg.Client)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())
	console := &Console{
		IDaemon: embedDaemon,
		ctx:     ctx,
		cancel:  cancel,
		cfg:     cfg,
		oct:     oct,
		engine:  fiz,
		started: make(chan struct{}, 1),
	}

	root := fiz.Group("/api", "Olive.Default", "")
	root.POST("/v1/login", []fizz.OperationOption{
		fizz.Summary("User logins olive system and get token."),
	}, tonic.Handler(login(oct), 200))
	routeV1 := root.Group("/v1", "", "", bearer.Handler)
	console.routeTree, err = routes.RegisterRoutes(ctx, cfg, routeV1, oct)
	if err != nil {
		return nil, fmt.Errorf("register routes: %w", err)
	}

	fiz.Generator().SetSecuritySchemes(map[string]*openapi.SecuritySchemeOrRef{
		"Bearer": {
			SecurityScheme: &openapi.SecurityScheme{
				Type:   "http",
				Scheme: "bearer",
				Name:   "Authorization",
				In:     "header",
			},
		},
	})

	if len(fiz.Errors()) != 0 {
		return nil, fmt.Errorf("engine errors: %v", errors.Join(fiz.Errors()...))
	}

	console.serve = &http.Server{
		Handler:        fiz,
		MaxHeaderBytes: config.DefaultMaxHeaderSize,
	}

	return console, nil
}

func (c *Console) Logger() *zap.Logger {
	return c.cfg.GetLogger()
}

func (c *Console) StartNotify() <-chan struct{} {
	return c.started
}

func (c *Console) Start(stopc <-chan struct{}) error {
	if c.isStarted() {
		return nil
	}

	lg := c.Logger()

	scheme, ts, err := c.createListener()
	if err != nil {
		return err
	}

	lg.Info("Server [grpc] Listening", zap.String("addr", ts.Addr().String()))

	c.mu.Lock()
	c.cfg.ListenURL = scheme + "//" + ts.Addr().String()
	c.mu.Unlock()

	c.GoAttach(c.process)
	c.OnDestroy(c.destroy)

	ech := make(chan error, 1)
	go func() {
		if serveErr := c.serve.Serve(ts); serveErr != nil {
			ech <- serveErr
		} else {
			close(ech)
		}
	}()

	select {
	case err = <-ech:
		return err
	case <-time.After(time.Second * 2):
	}

	c.beStarted()

	<-stopc

	return c.stop()
}

func (c *Console) process() {}

func (c *Console) destroy() {
	c.routeTree.Destroy()
}

func (c *Console) stop() error {
	logger := c.Logger().Sugar()
	logger.Debugf("shutting down server")
	if err := c.serve.Shutdown(c.ctx); err != nil {
		if !errors.Is(err, http.ErrServerClosed) {
			return err
		}
	}

	c.cancel()
	c.IDaemon.Shutdown()
	return nil
}

func (c *Console) beStarted() {
	select {
	case <-c.started:
		return
	default:
		close(c.started)
	}
}

func (c *Console) isStarted() bool {
	select {
	case <-c.started:
		return true
	default:
		return false
	}
}

func (c *Console) createListener() (string, net.Listener, error) {
	lg := c.Logger()

	c.mu.RLock()
	listenURL := c.cfg.ListenURL
	c.mu.RUnlock()
	url, err := urlpkg.Parse(listenURL)
	if err != nil {
		return "", nil, err
	}
	host := url.Host

	lg.Debug("listen on " + host)
	listener, err := net.Listen("tcp", host)
	if err != nil {
		return "", nil, err
	}

	return url.Scheme, listener, nil
}
