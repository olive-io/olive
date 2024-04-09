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

package runtime

import "path"

const (
	DefaultOlivePrefix = "_olive"
)

var (
	DefaultMetaPrefix            = path.Join(DefaultOlivePrefix, "meta")
	DefaultMetaRunnerRegistrarId = path.Join(DefaultMetaPrefix, "runnerIds")
	DefaultMetaRegionRegistrarId = path.Join(DefaultMetaPrefix, "regionsIds")
	DefaultMetaRunnerPrefix      = path.Join(DefaultMetaPrefix, "runner")
	DefaultMetaDefinitionPrefix  = path.Join(DefaultMetaPrefix, "definitions")
	DefaultMetaDefinitionMeta    = path.Join(DefaultMetaDefinitionPrefix, "meta")
	DefaultMetaRunnerRegistrar   = path.Join(DefaultMetaRunnerPrefix, "registry")
	DefaultMetaRunnerStat        = path.Join(DefaultMetaRunnerPrefix, "stat")
	DefaultMetaRegionStat        = path.Join(DefaultMetaRunnerPrefix, "region", "stat")
)

var (
	DefaultRunnerPrefix          = path.Join(DefaultOlivePrefix, "runner")
	DefaultRunnerDefinitions     = path.Join(DefaultRunnerPrefix, "definitions")
	DefaultRunnerProcessInstance = path.Join(DefaultRunnerPrefix, "processInstances")
	DefaultRunnerRegion          = path.Join(DefaultRunnerPrefix, "regions")
	DefaultRunnerDiscoveryPrefix = path.Join(DefaultRunnerPrefix, "discovery")
	DefaultRunnerDiscoveryNode   = path.Join(DefaultRunnerDiscoveryPrefix, "nodes")
)
