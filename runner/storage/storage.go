package storage

import (
	"context"
	"path"

	json "github.com/bytedance/sonic"

	"github.com/olive-io/olive/api"
	metav1 "github.com/olive-io/olive/api/types/meta/v1"
	"github.com/olive-io/olive/runner/storage/backend"
)

type Storage struct {
	scheme *api.Scheme

	db backend.IBackend
}

func New(scheme *api.Scheme, db backend.IBackend) *Storage {
	storage := &Storage{
		scheme: scheme,
		db:     db,
	}

	return storage
}

func (s *Storage) List(ctx context.Context, options *metav1.ListOptions) ([]api.Object, error) {
	gvk := api.FromGVK(options.GVK)

	prefix := path.Join(gvk.String())
	opts := []backend.GetOption{backend.GetPrefix()}

	resp, err := s.db.Get(ctx, prefix, opts...)
	if err != nil {
		return nil, err
	}

	out := make([]api.Object, 0)
	for _, kv := range resp.Kvs {
		obj, err := s.scheme.New(gvk)
		if err != nil {
			return nil, err
		}
		if err = json.Unmarshal(kv.Value, obj); err != nil {
			return nil, err
		}
		out = append(out, obj)
	}

	return out, nil
}

func (s *Storage) Get(ctx context.Context, options *metav1.GetOptions) (api.Object, error) {
	gvk := api.FromGVK(options.GVK)

	obj, err := s.scheme.New(gvk)
	if err != nil {
		return nil, err
	}

	key := path.Join(gvk.String(), options.UID)
	resp, err := s.db.Get(ctx, key)
	if err != nil {
		return nil, err
	}
	if err = json.Unmarshal(resp.Kvs[0].Value, obj); err != nil {
		return nil, err
	}

	return obj, nil
}

func (s *Storage) Post(ctx context.Context, options *metav1.PostOptions) error {
	gvk := api.FromGVK(options.GVK)

	obj, err := s.scheme.New(gvk)
	if err != nil {
		return err
	}
	if err = json.Unmarshal(options.Data, obj); err != nil {
		return err
	}

	key := path.Join(gvk.String(), options.UID)
	if err = s.db.Put(ctx, key, obj); err != nil {
		return err
	}

	return nil
}

func (s *Storage) Patch(ctx context.Context, options *metav1.PatchOptions) error {
	gvk := api.FromGVK(options.GVK)

	obj, err := s.scheme.New(gvk)
	if err != nil {
		return err
	}
	if err = json.Unmarshal(options.Data, obj); err != nil {
		return err
	}

	key := path.Join(gvk.String(), options.UID)
	if err := s.db.Put(ctx, key, obj); err != nil {
		return err
	}

	return nil
}

func (s *Storage) Delete(ctx context.Context, options *metav1.DeleteOptions) error {
	gvk := api.FromGVK(options.GVK)

	_, err := s.scheme.New(gvk)
	if err != nil {
		return err
	}

	key := path.Join(gvk.String(), options.UID)
	if err := s.db.Del(ctx, key); err != nil {
		return err
	}

	return nil
}
