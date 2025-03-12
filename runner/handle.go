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

package runner

import (
	"context"

	"go.uber.org/zap"

	"github.com/olive-io/olive/api/types"
)

func (r *Runner) handleEvent(ctx context.Context, event *types.RunnerEvent) {
	switch {
	case event.ExecuteProcess != nil:
		r.handleExecuteProcess(ctx, event.ExecuteProcess)
	}
}

func (r *Runner) handleExecuteProcess(ctx context.Context, msg *types.ExecuteProcessMsg) {
	lg := r.Logger()
	instance, err := r.oct.GetProcess(ctx, msg.Process)
	if err != nil {
		lg.Error("get process", zap.Int64("process", msg.Process), zap.Error(err))
		return
	}
	if instance.Finished() {
		return
	}

	if len(instance.DefinitionsContent) == 0 {
		definition, err := r.oct.GetDefinition(ctx, instance.DefinitionsId, instance.DefinitionsVersion)
		if err != nil {
			lg.Error("get definition",
				zap.Int64("id", instance.DefinitionsId),
				zap.Uint64("version", instance.DefinitionsVersion),
				zap.Error(err))
			return
		}
		instance.DefinitionsContent = definition.Content
	}

	if err = r.sch.RunProcess(ctx, instance); err != nil {
		lg.Error("run process", zap.Int64("process", msg.Process), zap.Error(err))
		return
	}

	lg.Info("start running process",
		zap.Int64("id", instance.Id),
		zap.String("name", instance.Name),
		zap.Int64("definition", instance.DefinitionsId),
		zap.Uint64("version", instance.DefinitionsVersion))
}
