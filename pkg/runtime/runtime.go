// Copyright 2023 Lack (xingyys@gmail.com).
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package runtime

import "path"

const (
	DefaultOlivePrefix = "_olive"
)

var (
	DefaultMetaPrefix           = path.Join(DefaultOlivePrefix, "meta")
	DefaultMetaRunnerRegistryId = path.Join(DefaultMetaPrefix, "runnerIds")
	DefaultMetaRegionRegistryId = path.Join(DefaultMetaPrefix, "regionsIds")
	DefaultMetaRunnerPrefix     = path.Join(DefaultMetaPrefix, "runner")
	DefaultMetaDefinitionPrefix = path.Join(DefaultMetaPrefix, "definitions")
	DefaultMetaDefinitionMeta   = path.Join(DefaultMetaDefinitionPrefix, "meta")
	DefaultMetaDefinitionWait   = path.Join(DefaultMetaDefinitionPrefix, "wait")
	DefaultMetaRunnerRegistry   = path.Join(DefaultMetaRunnerPrefix, "registry")
	DefaultMetaRunnerStat       = path.Join(DefaultMetaRunnerPrefix, "stat")
	DefaultMetaRegionStat       = path.Join(DefaultMetaRunnerPrefix, "region", "stat")
)

var (
	DefaultRunnerPrefix      = path.Join(DefaultOlivePrefix, "runner")
	DefaultRunnerDefinitions = path.Join(DefaultRunnerPrefix, "definitions")
	DefaultRunnerRegion      = path.Join(DefaultRunnerPrefix, "regions")
)
