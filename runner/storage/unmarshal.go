package storage

import (
	"fmt"

	json "github.com/bytedance/sonic"
)

func NewUnmarshaler[T any](resp *Response) *Unmarshaler[T] {
	m := &Unmarshaler[T]{
		resp: resp,
	}
	return m
}

type Unmarshaler[T any] struct {
	resp *Response
}

func (m *Unmarshaler[T]) To(target *T) error {
	resp := m.resp
	if len(resp.Kvs) == 0 {
		return fmt.Errorf("%w: no data", ErrNotFound)
	}
	kv := resp.Kvs[0]
	err := json.Unmarshal(kv.Value, target)
	if err != nil {
		return err
	}
	return nil
}

func (m *Unmarshaler[T]) SliceTo(out *[]T) error {
	resp := m.resp
	for _, kv := range resp.Kvs {
		target := new(T)
		err := json.Unmarshal(kv.Value, target)
		if err != nil {
			return err
		}
		*out = append(*out, *target)
	}
	return nil
}
