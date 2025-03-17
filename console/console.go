/*
Copyright 2025 The olive Authors

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
	"net"
	"net/http"
	urlpkg "net/url"
	"time"

	"github.com/cockroachdb/errors"
	"go.uber.org/zap"

	"github.com/olive-io/olive/client"
	"github.com/olive-io/olive/console/config"
	"github.com/olive-io/olive/console/server"
	genericserver "github.com/olive-io/olive/pkg/server"
)

type Console struct {
	genericserver.IEmbedServer

	ctx    context.Context
	cancel context.CancelFunc

	cfg *config.Config

	oct *client.Client

	serve *http.Server
}

func New(cfg *config.Config) (*Console, error) {
	lg := cfg.GetLogger()

	lg.Info("connecting to olive-mon")
	oct, err := client.New(&cfg.Client)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create client")
	}

	timeoutCtx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	if err = oct.Ping(timeoutCtx); err != nil {
		return nil, errors.Wrap(err, "failed to ping to monitor")
	}

	embedServer := genericserver.NewEmbedServer(lg)
	console := &Console{
		IEmbedServer: embedServer,

		cfg: cfg,
		oct: oct,
	}

	return console, nil
}

func (c *Console) Logger() *zap.Logger {
	return c.cfg.GetLogger()
}

func (c *Console) Start(ctx context.Context) error {
	c.ctx, c.cancel = context.WithCancel(ctx)
	if err := c.start(); err != nil {
		return err
	}
	if err := c.startServer(); err != nil {
		return err
	}

	c.Destroy(c.destroy)

	<-ctx.Done()

	return c.stop()
}

func (c *Console) stop() error {
	c.IEmbedServer.Shutdown()
	err := c.serve.Shutdown(c.ctx)
	if err != nil {
		return err
	}
	return nil
}

func (c *Console) destroy() {
	c.cancel()
}

func (c *Console) start() error {
	return nil
}

func (c *Console) startServer() error {
	lg := c.Logger()

	scheme, ts, err := c.createListener()
	if err != nil {
		return err
	}

	lg.Info("Server [grpc] Listening", zap.String("addr", ts.Addr().String()))

	c.cfg.ListenURL = scheme + ts.Addr().String()
	handler, err := server.RegisterServer(c.ctx, c.cfg, c.oct)
	if err != nil {
		return err
	}

	c.serve = &http.Server{
		Handler:        handler,
		MaxHeaderBytes: 1024 * 1024 * 20,
	}

	go func() {
		_ = c.serve.Serve(ts)
	}()

	return nil
}

func (c *Console) createListener() (string, net.Listener, error) {
	cfg := c.cfg
	lg := c.Logger()
	url, err := urlpkg.Parse(cfg.ListenURL)
	if err != nil {
		return "", nil, err
	}
	host := url.Host

	lg.Debug("listen on " + host)
	listener, err := net.Listen("tcp", host)
	if err != nil {
		return "", nil, err
	}

	return "http://", listener, nil
}
