package yamlinternal

import (
	"reflect"

	_ "unsafe"

	"gopkg.in/yaml.v3"
)

type decoder struct{}

//go:linkname handleErr github.com/go-yaml/yaml.v3/handleErr
func handleErr(err *error)

//go:linkname newDecoder gopkg.in/yaml.v3@v3.0.1/yaml.newDecoder
func newDecoder() *decoder

//go:linkname unmarshal gopkg.in/yaml.v3.(*decoder).unmarshal
func unmarshal(d *decoder, n *yaml.Node, out reflect.Value) (good bool)

func NodeDecodeKnownFields(node *yaml.Node, v interface{}) (err error) {
	d := newDecoder()
	defer handleErr(&err)
	out := reflect.ValueOf(v)
	if out.Kind() == reflect.Ptr && !out.IsNil() {
		out = out.Elem()
	}
	unmarshal(d, node, out)

	terrors := reflect.ValueOf(d).Elem().FieldByName("terrors").Interface().([]string)

	if len(terrors) > 0 {
		return &yaml.TypeError{Errors: terrors}
	}
	return nil
}
