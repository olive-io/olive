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

package metrics

import (
	"sync"
	"sync/atomic"
	"testing"
)

var _ MetricRecorder = &fakeDefinitionsRecorder{}

type fakeDefinitionsRecorder struct {
	counter int64
}

func (r *fakeDefinitionsRecorder) Inc() {
	atomic.AddInt64(&r.counter, 1)
}

func (r *fakeDefinitionsRecorder) Dec() {
	atomic.AddInt64(&r.counter, -1)
}

func (r *fakeDefinitionsRecorder) Clear() {
	atomic.StoreInt64(&r.counter, 0)
}

func TestInc(t *testing.T) {
	fakeRecorder := fakeDefinitionsRecorder{}
	var wg sync.WaitGroup
	loops := 100
	wg.Add(loops)
	for i := 0; i < loops; i++ {
		go func() {
			fakeRecorder.Inc()
			wg.Done()
		}()
	}
	wg.Wait()
	if fakeRecorder.counter != int64(loops) {
		t.Errorf("Expected %v, got %v", loops, fakeRecorder.counter)
	}
}

func TestDec(t *testing.T) {
	fakeRecorder := fakeDefinitionsRecorder{counter: 100}
	var wg sync.WaitGroup
	loops := 100
	wg.Add(loops)
	for i := 0; i < loops; i++ {
		go func() {
			fakeRecorder.Dec()
			wg.Done()
		}()
	}
	wg.Wait()
	if fakeRecorder.counter != int64(0) {
		t.Errorf("Expected %v, got %v", loops, fakeRecorder.counter)
	}
}

func TestClear(t *testing.T) {
	fakeRecorder := fakeDefinitionsRecorder{}
	var wg sync.WaitGroup
	incLoops, decLoops := 100, 80
	wg.Add(incLoops + decLoops)
	for i := 0; i < incLoops; i++ {
		go func() {
			fakeRecorder.Inc()
			wg.Done()
		}()
	}
	for i := 0; i < decLoops; i++ {
		go func() {
			fakeRecorder.Dec()
			wg.Done()
		}()
	}
	wg.Wait()
	if fakeRecorder.counter != int64(incLoops-decLoops) {
		t.Errorf("Expected %v, got %v", incLoops-decLoops, fakeRecorder.counter)
	}
	// verify Clear() works
	fakeRecorder.Clear()
	if fakeRecorder.counter != int64(0) {
		t.Errorf("Expected %v, got %v", 0, fakeRecorder.counter)
	}
}
