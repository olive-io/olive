package backend

import (
	"fmt"

	json "github.com/bytedance/sonic"
)

func NewUnmarshaler[T any](rsp *Response) *Unmarshaler[T] {
	m := &Unmarshaler[T]{
		rsp: rsp,
	}
	return m
}

type Unmarshaler[T any] struct {
	rsp *Response
}

func (m *Unmarshaler[T]) To(target *T) error {
	rsp := m.rsp
	if len(rsp.Kvs) == 0 {
		return fmt.Errorf("%w: no data", ErrNotFound)
	}
	kv := rsp.Kvs[0]
	err := json.Unmarshal(kv.Value, target)
	if err != nil {
		return err
	}
	return nil
}

func (m *Unmarshaler[T]) SliceTo(out *[]T) error {
	rsp := m.rsp
	for _, kv := range rsp.Kvs {
		target := new(T)
		err := json.Unmarshal(kv.Value, target)
		if err != nil {
			return err
		}
		*out = append(*out, *target)
	}
	return nil
}
