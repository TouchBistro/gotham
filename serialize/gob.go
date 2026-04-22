// Package serialize provides serialization helpers for encoding Go values to
// bytes and decoding bytes back into values.
package serialize

import (
	"bytes"
	"encoding/gob"
)

// GobSerde is a gob-backed Serde.
type GobSerde struct{}

// Encode encodes the supplied value to bytes using encoding/gob.
func (GobSerde) Encode(o any) ([]byte, error) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(o); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// Decode decodes the supplied bytes into obj using encoding/gob.
func (GobSerde) Decode(data []byte, obj any) error {
	buf := bytes.NewBuffer(data)
	dec := gob.NewDecoder(buf)
	return dec.Decode(obj)
}
