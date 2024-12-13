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

package schedule

import "errors"

var (
	ErrNoRegion       = errors.New("region not found")
	ErrRegionNoSpace  = errors.New("region no space")
	ErrNoRunner       = errors.New("runner not found")
	ErrRunnerNotReady = errors.New("runner not ready")
	ErrRunnerBusy     = errors.New("all of runners are busy")
)
