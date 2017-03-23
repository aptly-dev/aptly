//Copyright 2013 GoGraphviz Authors
//
//Licensed under the Apache License, Version 2.0 (the "License");
//you may not use this file except in compliance with the License.
//You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
//Unless required by applicable law or agreed to in writing, software
//distributed under the License is distributed on an "AS IS" BASIS,
//WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//See the License for the specific language governing permissions and
//limitations under the License.

package gographviz

import (
	"reflect"
	"testing"
)

func TestEdges_Sorted(t *testing.T) {
	var tts = map[string]struct {
		edges    []*Edge
		expected []*Edge
	}{
		"empty": {
			edges:    []*Edge{},
			expected: []*Edge{},
		},
		"one edge": {
			edges: []*Edge{
				{Src: "0", Dst: "1", Attrs: map[string]string{"label": "abc"}},
			},
			expected: []*Edge{
				{Src: "0", Dst: "1", Attrs: map[string]string{"label": "abc"}},
			},
		},
		"two non parallel edges": {
			edges: []*Edge{
				{Src: "0", Dst: "2", Attrs: map[string]string{"label": "hello"}},
				{Src: "0", Dst: "1", Attrs: map[string]string{"label": "abc"}},
			},
			expected: []*Edge{
				{Src: "0", Dst: "1", Attrs: map[string]string{"label": "abc"}},
				{Src: "0", Dst: "2", Attrs: map[string]string{"label": "hello"}},
			},
		},
		"two parallel edges": {
			edges: []*Edge{
				{Src: "0", Dst: "1", Attrs: map[string]string{"label": "hello"}},
				{Src: "0", Dst: "1", Attrs: map[string]string{"label": "abc"}},
			},
			expected: []*Edge{
				{Src: "0", Dst: "1", Attrs: map[string]string{"label": "abc"}},
				{Src: "0", Dst: "1", Attrs: map[string]string{"label": "hello"}},
			},
		},
		"two parallel edges - one without label": {
			edges: []*Edge{
				{Src: "0", Dst: "1", Attrs: map[string]string{"label": "abc"}},
				{Src: "0", Dst: "1"},
			},
			expected: []*Edge{
				{Src: "0", Dst: "1"},
				{Src: "0", Dst: "1", Attrs: map[string]string{"label": "abc"}},
			},
		},
		"several non parallel edges": {
			edges: []*Edge{
				{Src: "0", Dst: "2", Attrs: map[string]string{"label": "hello"}},
				{Src: "1", Dst: "1", Attrs: map[string]string{"label": "world"}},
				{Src: "0", Dst: "1", Attrs: map[string]string{"label": "abc"}},
				{Src: "1", Dst: "0", Attrs: map[string]string{"label": "golang"}},
			},
			expected: []*Edge{
				{Src: "0", Dst: "1", Attrs: map[string]string{"label": "abc"}},
				{Src: "0", Dst: "2", Attrs: map[string]string{"label": "hello"}},
				{Src: "1", Dst: "0", Attrs: map[string]string{"label": "golang"}},
				{Src: "1", Dst: "1", Attrs: map[string]string{"label": "world"}},
			},
		},
		"several with parallel edges": {
			edges: []*Edge{
				{Src: "0", Dst: "2", Attrs: map[string]string{"label": "hello"}},
				{Src: "0", Dst: "1", Attrs: map[string]string{"label": "cba"}},
				{Src: "1", Dst: "1", Attrs: map[string]string{"label": "world"}},
				{Src: "0", Dst: "1", Attrs: map[string]string{"label": "abc"}},
				{Src: "1", Dst: "0", Attrs: map[string]string{"label": "gopher"}},
				{Src: "1", Dst: "0", Attrs: map[string]string{"label": "golang"}},
			},
			expected: []*Edge{
				{Src: "0", Dst: "1", Attrs: map[string]string{"label": "abc"}},
				{Src: "0", Dst: "1", Attrs: map[string]string{"label": "cba"}},
				{Src: "0", Dst: "2", Attrs: map[string]string{"label": "hello"}},
				{Src: "1", Dst: "0", Attrs: map[string]string{"label": "golang"}},
				{Src: "1", Dst: "0", Attrs: map[string]string{"label": "gopher"}},
				{Src: "1", Dst: "1", Attrs: map[string]string{"label": "world"}},
			},
		},
		"edges with ports": {
			edges: []*Edge{
				{Src: "0", Dst: "1", SrcPort: "a", DstPort: "b"},
				{Src: "0", Dst: "1", SrcPort: "a", DstPort: "a"},
				{Src: "0", Dst: "1", SrcPort: "b", DstPort: "a"},
			},
			expected: []*Edge{
				{Src: "0", Dst: "1", SrcPort: "a", DstPort: "a"},
				{Src: "0", Dst: "1", SrcPort: "a", DstPort: "b"},
				{Src: "0", Dst: "1", SrcPort: "b", DstPort: "a"},
			},
		},
		"directed edges before non directed edges": {
			edges: []*Edge{
				{Src: "0", Dst: "1", Dir: false},
				{Src: "0", Dst: "1", Dir: true},
			},
			expected: []*Edge{
				{Src: "0", Dst: "1", Dir: true},
				{Src: "0", Dst: "1", Dir: false},
			},
		},
		"the theory of everything": {
			edges: []*Edge{
				{Src: "0", Dst: "1", Attrs: map[string]string{"label": "cba"}},
				{Src: "1", Dst: "0", SrcPort: "a", Dir: false, Attrs: map[string]string{"label": "gopher"}},
				{Src: "0", Dst: "1", Attrs: map[string]string{"label": "abc"}},
				{Src: "0", Dst: "2", Attrs: map[string]string{"label": "hello"}},
				{Src: "1", Dst: "0", Attrs: map[string]string{"label": "gopher"}},
				{Src: "1", Dst: "0", Attrs: map[string]string{"label": "golang"}},
				{Src: "1", Dst: "0", SrcPort: "b", Attrs: map[string]string{"label": "gopher"}},
				{Src: "1", Dst: "0", SrcPort: "a", DstPort: "b", Attrs: map[string]string{"label": "golang"}},
				{Src: "1", Dst: "1", Attrs: map[string]string{"comment": "test", "label": "world"}},
				{Src: "1", Dst: "0", SrcPort: "a", DstPort: "a", Attrs: map[string]string{"label": "golang"}},
				{Src: "1", Dst: "0", SrcPort: "a", Attrs: map[string]string{"label": "golang"}},
				{Src: "1", Dst: "0", SrcPort: "b", Dir: false, Attrs: map[string]string{"label": "gopher"}},
				{Src: "1", Dst: "1", Attrs: map[string]string{"label": "world"}},
				{Src: "1", Dst: "0", SrcPort: "a", DstPort: "b", Dir: true, Attrs: map[string]string{"label": "golang"}},
				{Src: "1", Dst: "1", Attrs: map[string]string{"comment": "test", "label": "hello"}},
				{Src: "1", Dst: "0", SrcPort: "a", Dir: true, Attrs: map[string]string{"label": "golang"}},
				{Src: "1", Dst: "0", SrcPort: "a", DstPort: "b", Dir: true, Attrs: map[string]string{"label": "graphviz"}},
			},
			expected: []*Edge{
				{Src: "0", Dst: "1", Attrs: map[string]string{"label": "abc"}},
				{Src: "0", Dst: "1", Attrs: map[string]string{"label": "cba"}},
				{Src: "0", Dst: "2", Attrs: map[string]string{"label": "hello"}},
				{Src: "1", Dst: "0", Attrs: map[string]string{"label": "golang"}},
				{Src: "1", Dst: "0", Attrs: map[string]string{"label": "gopher"}},
				{Src: "1", Dst: "0", SrcPort: "a", Dir: true, Attrs: map[string]string{"label": "golang"}},
				{Src: "1", Dst: "0", SrcPort: "a", Attrs: map[string]string{"label": "golang"}},
				{Src: "1", Dst: "0", SrcPort: "a", Dir: false, Attrs: map[string]string{"label": "gopher"}},
				{Src: "1", Dst: "0", SrcPort: "a", DstPort: "a", Attrs: map[string]string{"label": "golang"}},
				{Src: "1", Dst: "0", SrcPort: "a", DstPort: "b", Dir: true, Attrs: map[string]string{"label": "golang"}},
				{Src: "1", Dst: "0", SrcPort: "a", DstPort: "b", Dir: true, Attrs: map[string]string{"label": "graphviz"}},
				{Src: "1", Dst: "0", SrcPort: "a", DstPort: "b", Attrs: map[string]string{"label": "golang"}},
				{Src: "1", Dst: "0", SrcPort: "b", Dir: false, Attrs: map[string]string{"label": "gopher"}},
				{Src: "1", Dst: "0", SrcPort: "b", Attrs: map[string]string{"label": "gopher"}},
				{Src: "1", Dst: "1", Attrs: map[string]string{"label": "world"}},
				{Src: "1", Dst: "1", Attrs: map[string]string{"comment": "test", "label": "hello"}},
				{Src: "1", Dst: "1", Attrs: map[string]string{"comment": "test", "label": "world"}},
			},
		},
	}

	for name, tt := range tts {
		edges := NewEdges()
		for _, e := range tt.edges {
			edges.Add(e)
		}
		s := edges.Sorted()
		if !reflect.DeepEqual(tt.expected, s) {
			t.Errorf("%s - Sorted invalid: expected %v got %v", name, tt.expected, s)
		} else if !reflect.DeepEqual(edges.Edges, tt.edges) {
			t.Errorf("%s - Sorted should not have changed original order: expected %v got %v", name, tt.edges, edges.Edges)
		}
	}
}
