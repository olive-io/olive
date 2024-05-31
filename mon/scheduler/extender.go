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

package scheduler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	v1 "k8s.io/api/core/v1"
	utilnet "k8s.io/apimachinery/pkg/util/net"
	"k8s.io/apimachinery/pkg/util/sets"
	restclient "k8s.io/client-go/rest"

	configv1 "github.com/olive-io/olive/apis/config/v1"
	corev1 "github.com/olive-io/olive/apis/core/v1"
	extenderv1 "github.com/olive-io/olive/mon/scheduler/extender/v1"
	"github.com/olive-io/olive/mon/scheduler/framework"
)

const (
	// DefaultExtenderTimeout defines the default extender timeout in second.
	DefaultExtenderTimeout = 5 * time.Second
)

// HTTPExtender implements the Extender interface.
type HTTPExtender struct {
	extenderURL        string
	preemptVerb        string
	filterVerb         string
	prioritizeVerb     string
	bindVerb           string
	weight             int64
	client             *http.Client
	runnerCacheCapable bool
	managedResources   sets.Set[string]
	ignorable          bool
}

func makeTransport(config *configv1.Extender) (http.RoundTripper, error) {
	var cfg restclient.Config
	if config.TLSConfig != nil {
		cfg.TLSClientConfig.Insecure = config.TLSConfig.Insecure
		cfg.TLSClientConfig.ServerName = config.TLSConfig.ServerName
		cfg.TLSClientConfig.CertFile = config.TLSConfig.CertFile
		cfg.TLSClientConfig.KeyFile = config.TLSConfig.KeyFile
		cfg.TLSClientConfig.CAFile = config.TLSConfig.CAFile
		cfg.TLSClientConfig.CertData = config.TLSConfig.CertData
		cfg.TLSClientConfig.KeyData = config.TLSConfig.KeyData
		cfg.TLSClientConfig.CAData = config.TLSConfig.CAData
	}
	if config.EnableHTTPS {
		hasCA := len(cfg.CAFile) > 0 || len(cfg.CAData) > 0
		if !hasCA {
			cfg.Insecure = true
		}
	}
	tlsConfig, err := restclient.TLSConfigFor(&cfg)
	if err != nil {
		return nil, err
	}
	if tlsConfig != nil {
		return utilnet.SetTransportDefaults(&http.Transport{
			TLSClientConfig: tlsConfig,
		}), nil
	}
	return utilnet.SetTransportDefaults(&http.Transport{}), nil
}

// NewHTTPExtender creates an HTTPExtender object.
func NewHTTPExtender(config *configv1.Extender) (framework.Extender, error) {
	if config.HTTPTimeout.Duration.Nanoseconds() == 0 {
		config.HTTPTimeout.Duration = time.Duration(DefaultExtenderTimeout)
	}

	transport, err := makeTransport(config)
	if err != nil {
		return nil, err
	}
	client := &http.Client{
		Transport: transport,
		Timeout:   config.HTTPTimeout.Duration,
	}
	managedResources := sets.New[string]()
	for _, r := range config.ManagedResources {
		managedResources.Insert(string(r.Name))
	}
	return &HTTPExtender{
		extenderURL:        config.URLPrefix,
		preemptVerb:        config.PreemptVerb,
		filterVerb:         config.FilterVerb,
		prioritizeVerb:     config.PrioritizeVerb,
		bindVerb:           config.BindVerb,
		weight:             config.Weight,
		client:             client,
		runnerCacheCapable: config.RunnerCacheCapable,
		managedResources:   managedResources,
		ignorable:          config.Ignorable,
	}, nil
}

// Name returns extenderURL to identify the extender.
func (h *HTTPExtender) Name() string {
	return h.extenderURL
}

// IsIgnorable returns true indicates scheduling should not fail when this extender
// is unavailable
func (h *HTTPExtender) IsIgnorable() bool {
	return h.ignorable
}

// SupportsPreemption returns true if an extender supports preemption.
// An extender should have preempt verb defined and enabled its own runner cache.
func (h *HTTPExtender) SupportsPreemption() bool {
	return len(h.preemptVerb) > 0
}

// ProcessPreemption returns filtered candidate runners and victims after running preemption logic in extender.
func (h *HTTPExtender) ProcessPreemption(
	region *corev1.Region,
	runnerNameToVictims map[string]*extenderv1.Victims,
	runnerInfos framework.RunnerInfoLister,
) (map[string]*extenderv1.Victims, error) {
	var (
		result extenderv1.ExtenderPreemptionResult
		args   *extenderv1.ExtenderPreemptionArgs
	)

	if !h.SupportsPreemption() {
		return nil, fmt.Errorf("preempt verb is not defined for extender %v but run into ProcessPreemption", h.extenderURL)
	}

	if h.runnerCacheCapable {
		// If extender has cached runner info, pass RunnerNameToMetaVictims in args.
		runnerNameToMetaVictims := convertToMetaVictims(runnerNameToVictims)
		args = &extenderv1.ExtenderPreemptionArgs{
			Region:                  region,
			RunnerNameToMetaVictims: runnerNameToMetaVictims,
		}
	} else {
		args = &extenderv1.ExtenderPreemptionArgs{
			Region:              region,
			RunnerNameToVictims: runnerNameToVictims,
		}
	}

	if err := h.send(h.preemptVerb, args, &result); err != nil {
		return nil, err
	}

	// Extender will always return RunnerNameToMetaVictims.
	// So let's convert it to RunnerNameToVictims by using <runnerInfos>.
	newRunnerNameToVictims, err := h.convertToVictims(result.RunnerNameToMetaVictims, runnerInfos)
	if err != nil {
		return nil, err
	}
	// Do not override <runnerNameToVictims>.
	return newRunnerNameToVictims, nil
}

// convertToVictims converts "runnerNameToMetaVictims" from object identifiers,
// such as UIDs and names, to object pointers.
func (h *HTTPExtender) convertToVictims(
	runnerNameToMetaVictims map[string]*extenderv1.MetaVictims,
	runnerInfos framework.RunnerInfoLister,
) (map[string]*extenderv1.Victims, error) {
	runnerNameToVictims := map[string]*extenderv1.Victims{}
	for runnerName, metaVictims := range runnerNameToMetaVictims {
		runnerInfo, err := runnerInfos.Get(runnerName)
		if err != nil {
			return nil, err
		}
		victims := &extenderv1.Victims{
			Regions:          []*corev1.Region{},
			NumPDBViolations: metaVictims.NumPDBViolations,
		}
		for _, metaRegion := range metaVictims.Regions {
			region, err := h.convertRegionUIDToRegion(metaRegion, runnerInfo)
			if err != nil {
				return nil, err
			}
			victims.Regions = append(victims.Regions, region)
		}
		runnerNameToVictims[runnerName] = victims
	}
	return runnerNameToVictims, nil
}

// convertRegionUIDToRegion returns v1.Region object for given MetaRegion and runner info.
// The v1.Region object is restored by runnerInfo.Regions().
// It returns an error if there's cache inconsistency between default scheduler
// and extender, i.e. when the region is not found in runnerInfo.Regions.
func (h *HTTPExtender) convertRegionUIDToRegion(
	metaRegion *extenderv1.MetaRegion,
	runnerInfo *framework.RunnerInfo) (*corev1.Region, error) {
	for _, p := range runnerInfo.Regions {
		if string(p.Region.UID) == metaRegion.UID {
			return p.Region, nil
		}
	}
	return nil, fmt.Errorf("extender: %v claims to preempt region (UID: %v) on runner: %v, but the region is not found on that runner",
		h.extenderURL, metaRegion, runnerInfo.Runner().Name)
}

// convertToMetaVictims converts from struct type to meta types.
func convertToMetaVictims(
	runnerNameToVictims map[string]*extenderv1.Victims,
) map[string]*extenderv1.MetaVictims {
	runnerNameToMetaVictims := map[string]*extenderv1.MetaVictims{}
	for runner, victims := range runnerNameToVictims {
		metaVictims := &extenderv1.MetaVictims{
			Regions:          []*extenderv1.MetaRegion{},
			NumPDBViolations: victims.NumPDBViolations,
		}
		for _, region := range victims.Regions {
			metaRegion := &extenderv1.MetaRegion{
				UID: string(region.UID),
			}
			metaVictims.Regions = append(metaVictims.Regions, metaRegion)
		}
		runnerNameToMetaVictims[runner] = metaVictims
	}
	return runnerNameToMetaVictims
}

// Filter based on extender implemented predicate functions. The filtered list is
// expected to be a subset of the supplied list; otherwise the function returns an error.
// The failedRunners and failedAndUnresolvableRunners optionally contains the list
// of failed runners and failure reasons, except runners in the latter are
// unresolvable.
func (h *HTTPExtender) Filter(
	region *corev1.Region,
	runners []*framework.RunnerInfo,
) (filteredList []*framework.RunnerInfo, failedRunners, failedAndUnresolvableRunners extenderv1.FailedRunnersMap, err error) {
	var (
		result       extenderv1.ExtenderFilterResult
		runnerList   *corev1.RunnerList
		runnerNames  *[]string
		runnerResult []*framework.RunnerInfo
		args         *extenderv1.ExtenderArgs
	)
	fromRunnerName := make(map[string]*framework.RunnerInfo)
	for _, n := range runners {
		fromRunnerName[n.Runner().Name] = n
	}

	if h.filterVerb == "" {
		return runners, extenderv1.FailedRunnersMap{}, extenderv1.FailedRunnersMap{}, nil
	}

	if h.runnerCacheCapable {
		runnerNameSlice := make([]string, 0, len(runners))
		for _, runner := range runners {
			runnerNameSlice = append(runnerNameSlice, runner.Runner().Name)
		}
		runnerNames = &runnerNameSlice
	} else {
		runnerList = &corev1.RunnerList{}
		for _, runner := range runners {
			runnerList.Items = append(runnerList.Items, *runner.Runner())
		}
	}

	args = &extenderv1.ExtenderArgs{
		Region:      region,
		Runners:     runnerList,
		RunnerNames: runnerNames,
	}

	if err := h.send(h.filterVerb, args, &result); err != nil {
		return nil, nil, nil, err
	}
	if result.Error != "" {
		return nil, nil, nil, fmt.Errorf(result.Error)
	}

	if h.runnerCacheCapable && result.RunnerNames != nil {
		runnerResult = make([]*framework.RunnerInfo, len(*result.RunnerNames))
		for i, runnerName := range *result.RunnerNames {
			if n, ok := fromRunnerName[runnerName]; ok {
				runnerResult[i] = n
			} else {
				return nil, nil, nil, fmt.Errorf(
					"extender %q claims a filtered runner %q which is not found in the input runner list",
					h.extenderURL, runnerName)
			}
		}
	} else if result.Runners != nil {
		runnerResult = make([]*framework.RunnerInfo, len(result.Runners.Items))
		for i := range result.Runners.Items {
			runnerResult[i] = framework.NewRunnerInfo()
			runnerResult[i].SetRunner(&result.Runners.Items[i])
		}
	}

	return runnerResult, result.FailedRunners, result.FailedAndUnresolvableRunners, nil
}

// Prioritize based on extender implemented priority functions. Weight*priority is added
// up for each such priority function. The returned score is added to the score computed
// by Kubernetes scheduler. The total score is used to do the host selection.
func (h *HTTPExtender) Prioritize(region *corev1.Region, runners []*framework.RunnerInfo) (*extenderv1.HostPriorityList, int64, error) {
	var (
		result      extenderv1.HostPriorityList
		runnerList  *corev1.RunnerList
		runnerNames *[]string
		args        *extenderv1.ExtenderArgs
	)

	if h.prioritizeVerb == "" {
		result := extenderv1.HostPriorityList{}
		for _, runner := range runners {
			result = append(result, extenderv1.HostPriority{Host: runner.Runner().Name, Score: 0})
		}
		return &result, 0, nil
	}

	if h.runnerCacheCapable {
		runnerNameSlice := make([]string, 0, len(runners))
		for _, runner := range runners {
			runnerNameSlice = append(runnerNameSlice, runner.Runner().Name)
		}
		runnerNames = &runnerNameSlice
	} else {
		runnerList = &corev1.RunnerList{}
		for _, runner := range runners {
			runnerList.Items = append(runnerList.Items, *runner.Runner())
		}
	}

	args = &extenderv1.ExtenderArgs{
		Region:      region,
		Runners:     runnerList,
		RunnerNames: runnerNames,
	}

	if err := h.send(h.prioritizeVerb, args, &result); err != nil {
		return nil, 0, err
	}
	return &result, h.weight, nil
}

// Bind delegates the action of binding a region to a runner to the extender.
func (h *HTTPExtender) Bind(binding *corev1.Binding) error {
	var result extenderv1.ExtenderBindingResult
	if !h.IsBinder() {
		// This shouldn't happen as this extender wouldn't have become a Binder.
		return fmt.Errorf("unexpected empty bindVerb in extender")
	}
	req := &extenderv1.ExtenderBindingArgs{
		RegionName:      binding.Name,
		RegionNamespace: binding.Namespace,
		RegionUID:       binding.UID,
		Runner:          binding.Target.Names[0],
	}
	if err := h.send(h.bindVerb, req, &result); err != nil {
		return err
	}
	if result.Error != "" {
		return fmt.Errorf(result.Error)
	}
	return nil
}

// IsBinder returns whether this extender is configured for the Bind method.
func (h *HTTPExtender) IsBinder() bool {
	return h.bindVerb != ""
}

// IsPrioritizer returns whether this extender is configured for the Prioritize method.
func (h *HTTPExtender) IsPrioritizer() bool {
	return h.prioritizeVerb != ""
}

// IsFilter returns whether this extender is configured for the Filter method.
func (h *HTTPExtender) IsFilter() bool {
	return h.filterVerb != ""
}

// Helper function to send messages to the extender
func (h *HTTPExtender) send(action string, args interface{}, result interface{}) error {
	out, err := json.Marshal(args)
	if err != nil {
		return err
	}

	url := strings.TrimRight(h.extenderURL, "/") + "/" + action

	req, err := http.NewRequest("POST", url, bytes.NewReader(out))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := h.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed %v with extender at URL %v, code %v", action, url, resp.StatusCode)
	}

	return json.NewDecoder(resp.Body).Decode(result)
}

// IsInterested returns true if at least one extended resource requested by
// this region is managed by this extender.
func (h *HTTPExtender) IsInterested(region *corev1.Region) bool {
	if h.managedResources.Len() == 0 {
		return true
	}
	return false
}

func (h *HTTPExtender) hasManagedResources(containers []v1.Container) bool {
	for i := range containers {
		container := &containers[i]
		for resourceName := range container.Resources.Requests {
			if h.managedResources.Has(string(resourceName)) {
				return true
			}
		}
		for resourceName := range container.Resources.Limits {
			if h.managedResources.Has(string(resourceName)) {
				return true
			}
		}
	}
	return false
}
