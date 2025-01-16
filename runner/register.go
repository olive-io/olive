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

import (
	"errors"
	"fmt"
	"time"

	"github.com/dustin/go-humanize"
	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	corev1 "github.com/olive-io/olive/apis/core/v1"
	clientgo "github.com/olive-io/olive/client-go"
)

type clientSet struct {
	logger *zap.Logger
	cfg    *Config
	conn   *clientgo.Client
}

func newClientSet(config *Config) *clientSet {
	logger := config.GetLogger()
	lg := logger.Sugar()

	client := &clientSet{
		logger: logger,
	}

	cfg, err := clientgo.NewConfig(config.OliveConfig, logger)
	if err != nil {
		lg.Warnf("ClientSet not initialized: %v", err)
		return client
	}

	oct, err := clientgo.New(cfg)
	if err != nil {
		lg.Warnf("ClientSet not initialized: %v", err)
		return client
	}

	client.conn = oct

	lg.Infof("ClientSet initialized")

	return client
}

func (c *clientSet) Get() (*clientgo.Client, error) {
	if c.conn == nil {
		return nil, errors.New("ClientSet not initialized")
	}
	return c.conn, nil
}

func (r *Runner) process() {

	conn, err := r.clientSet.Get()
	if err != nil {
		return
	}

	ctx := r.ctx
	runner := r.gather.GetRunner()

	runner.Status.Phase = corev1.RunnerActive

	remote, err := conn.CoreV1().Runners().Get(ctx, runner.Name, metav1.GetOptions{})
	if err != nil {
		remote = runner
		runner, err = conn.CoreV1().Runners().Create(ctx, remote, metav1.CreateOptions{})
	} else {
		runner.Spec.DeepCopyInto(&remote.Spec)
		runner.Status.DeepCopyInto(&remote.Status)
		runner, err = conn.CoreV1().Runners().Update(ctx, remote, metav1.UpdateOptions{})
	}

	lg := r.Logger().Sugar()
	if err != nil {
		lg.Warnf("registry runner: %v", err)
	}

	r.Logger().Info("olive-runner registered",
		zap.String("listen-url", runner.Spec.ListenURL),
		zap.String("cpu", fmt.Sprintf("%dx%dMhz", runner.Status.CpuSocket, uint64(runner.Status.CpuTotal))),
		zap.String("memory", humanize.IBytes(uint64(runner.Status.MemoryTotal))),
		zap.String("version", runner.Spec.Version))

	heartbeatMs := r.cfg.HeartbeatMs
	ticker := time.NewTicker(time.Duration(heartbeatMs) * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-r.StoppingNotify():
			return
		case <-ticker.C:
			statistics, err := r.gather.GetStat()
			if err != nil {
				continue
			}

			runner.Status.Stat = *statistics
			runner, err = conn.CoreV1().Runners().UpdateStatus(ctx, runner, metav1.UpdateOptions{})
		}
	}
}
