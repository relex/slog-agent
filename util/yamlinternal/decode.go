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

// NodeDecodeKnownFields is yaml.Node.Decode with known fields set to true.
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
