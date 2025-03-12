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
	process, err := r.oct.GetProcess(ctx, msg.Process)
	if err != nil {
		lg.Error("get process", zap.Int64("process", msg.Process), zap.Error(err))
		return
	}
	if process.Finished() {
		return
	}

	if len(process.DefinitionsContent) == 0 {
		definition, err := r.oct.GetDefinition(ctx, process.DefinitionsId, process.DefinitionsVersion)
		if err != nil {
			lg.Error("get definition",
				zap.Int64("id", process.DefinitionsId),
				zap.Uint64("version", process.DefinitionsVersion),
				zap.Error(err))
			return
		}
		process.DefinitionsContent = definition.Content
	}

	if err = r.sch.RunProcess(ctx, process); err != nil {
		lg.Error("run process", zap.Int64("process", msg.Process), zap.Error(err))
		return
	}

	lg.Info("start running process",
		zap.Int64("id", process.Id),
		zap.String("name", process.Name),
		zap.Int64("definition", process.DefinitionsId),
		zap.Uint64("version", process.DefinitionsVersion))
}
