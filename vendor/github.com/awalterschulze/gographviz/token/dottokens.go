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

package token

var DOTTokens = NewMapFromStrings([]string{
	"ε",
	"id",
	"{",
	"}",
	";",
	"=",
	"[",
	"]",
	",",
	":",
	"->",
	"--",
	"graph",
	"Graph",
	"GRAPH",
	"strict",
	"Strict",
	"STRICT",
	"digraph",
	"Digraph",
	"DiGraph",
	"DIGRAPH",
	"node",
	"Node",
	"NODE",
	"edge",
	"Edge",
	"EDGE",
	"subgraph",
	"Subgraph",
	"SubGraph",
	"SUBGRAPH",
	"string_lit",
	"int_lit",
	"float_lit",
	"html_lit",
})
