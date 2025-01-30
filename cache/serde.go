package cache

import (
	"bytes"
	"encoding/gob"
)

type GobSerde struct{}

func (r GobSerde) ser(o any) ([]byte, error) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(o)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (r GobSerde) de(data []byte, obj any) error {
	buf := bytes.NewBuffer(data)
	dec := gob.NewDecoder(buf)
	return dec.Decode(obj)
}
