// Copyright (c) 2024 RELEX Oy
// Copyright (c) 2011-2019 Canonical Ltd
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

package yamlinternal

import (
	"fmt"
	"reflect"

	_ "unsafe"

	"gopkg.in/yaml.v3"
)

// decoder is a copy from gopkg.in/yaml.v3/decode.go
//
// FIXME: Can we get the actual type definition of the returned valued from newDecoder()? Attempts resulted in "<invalid reflect.Value>"
//
//lint:ignore U1000 all fields until the last used field must be kept for proper offsets
type decoder struct {
	doc     *yaml.Node
	aliases map[*yaml.Node]bool
	terrors []string

	stringMapType  reflect.Type
	generalMapType reflect.Type

	knownFields bool
	uniqueKeys  bool
	decodeCount int
	aliasCount  int
	aliasDepth  int

	mergedFields map[interface{}]bool
}

//go:linkname handleErr gopkg.in/yaml%2ev3.handleErr
func handleErr(err *error)

//go:linkname newDecoder gopkg.in/yaml%2ev3.newDecoder
func newDecoder() *decoder

//go:linkname unmarshal gopkg.in/yaml%2ev3.(*decoder).unmarshal
func unmarshal(d *decoder, n *yaml.Node, out reflect.Value) (good bool)

// NodeDecodeKnownFields is yaml.Node.Decode with KnownFields=true, to disallow unknown fields in YAML source.
func NodeDecodeKnownFields(node *yaml.Node, v interface{}) (err error) {
	d := newDecoder()
	d.knownFields = true
	defer handleErr(&err)
	out := reflect.ValueOf(v)
	if out.Kind() == reflect.Ptr && !out.IsNil() {
		out = out.Elem()
	}
	good := unmarshal(d, node, out)

	if len(d.terrors) > 0 {
		return &yaml.TypeError{Errors: d.terrors}
	}

	if !good {
		return fmt.Errorf("not good")
	}
	return nil
}
