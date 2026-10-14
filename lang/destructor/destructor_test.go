// Copyright 2026 The Flatcar Maintainers
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

package destructor

import (
	"bytes"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/coreos/pkg/capnslog"
)

type fakeCloser struct {
	err    error
	closed int
}

func (f *fakeCloser) Close() error {
	f.closed++
	return f.err
}

func TestCloserDestructorClosesUnderlyingCloser(t *testing.T) {
	fc := &fakeCloser{}
	CloserDestructor{fc}.Destroy()

	if fc.closed != 1 {
		t.Errorf("expected Close to be called once, got %d", fc.closed)
	}
}

func TestCloserDestructorLogsCloseError(t *testing.T) {
	var buf bytes.Buffer
	capnslog.SetFormatter(capnslog.NewStringFormatter(&buf))
	defer capnslog.SetFormatter(capnslog.NewDefaultFormatter(os.Stderr))

	wantErr := errors.New("close failed")
	CloserDestructor{&fakeCloser{err: wantErr}}.Destroy()

	if !strings.Contains(buf.String(), wantErr.Error()) {
		t.Errorf("expected log output to mention %q, got %q", wantErr, buf.String())
	}
}

func TestMultiDestructorDestroysAll(t *testing.T) {
	a := &fakeCloser{}
	b := &fakeCloser{}
	m := MultiDestructor{CloserDestructor{a}, CloserDestructor{b}}
	m.Destroy()

	if a.closed != 1 || b.closed != 1 {
		t.Errorf("expected both destructors to run, got a=%d b=%d", a.closed, b.closed)
	}
}

func TestMultiDestructorAddCloser(t *testing.T) {
	var m MultiDestructor
	a := &fakeCloser{}
	m.AddCloser(a)

	if len(m) != 1 {
		t.Fatalf("expected 1 destructor, got %d", len(m))
	}
	m.Destroy()
	if a.closed != 1 {
		t.Errorf("expected AddCloser's closer to be closed, got %d", a.closed)
	}
}

func TestMultiDestructorAddDestructor(t *testing.T) {
	var m MultiDestructor
	a := &fakeCloser{}
	b := &fakeCloser{}
	m.AddDestructor(CloserDestructor{a})
	m.AddDestructor(CloserDestructor{b})

	if len(m) != 2 {
		t.Fatalf("expected 2 destructors, got %d", len(m))
	}
	m.Destroy()
	if a.closed != 1 || b.closed != 1 {
		t.Errorf("expected both added destructors to run, got a=%d b=%d", a.closed, b.closed)
	}
}

func TestMultiDestructorDestroysRemainingAfterOneErrors(t *testing.T) {
	a := &fakeCloser{err: errors.New("first fails")}
	b := &fakeCloser{}
	m := MultiDestructor{CloserDestructor{a}, CloserDestructor{b}}
	m.Destroy()

	if b.closed != 1 {
		t.Errorf("expected second destructor to still run after the first errored, got %d", b.closed)
	}
}
