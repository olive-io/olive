package storage

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"time"

	json "github.com/bytedance/sonic"

	"github.com/olive-io/olive/api"
	metav1 "github.com/olive-io/olive/api/types/meta/v1"
	"github.com/olive-io/olive/runner/storage/backend"
)

type Storage struct {
	*api.Scheme

	pathPrefix string

	db backend.IBackend
}

func New(scheme *api.Scheme, db backend.IBackend) *Storage {
	storage := &Storage{
		Scheme:     scheme,
		pathPrefix: "/rt",
		db:         db,
	}

	return storage
}

func (s *Storage) GenerateKey(obj api.Object) string {
	s.Scheme.SetTypesGVK(obj)
	gvk := obj.GetObjectKind().GroupVersionKind()
	if strings.HasSuffix(gvk.Kind, "List") {
		gvk.Kind = strings.TrimSuffix(gvk.Kind, "List")
	}

	key := gvk.String()
	return key
}

func (s *Storage) GetList(ctx context.Context, key string, listObj api.Object) error {
	listPtr, err := metav1.GetItemsPtr(listObj)
	if err != nil {
		return err
	}
	v, err := metav1.EnforcePtr(listPtr)
	if err != nil || v.Kind() != reflect.Slice {
		return fmt.Errorf("need ptr to slice: %v", err)
	}

	prefix, err := s.prepareKey(key)
	if err != nil {
		return err
	}
	opts := []backend.GetOption{backend.GetPrefix()}

	resp, err := s.db.Get(ctx, prefix, opts...)
	if err != nil {
		return err
	}

	growSlice(v, 1024, len(resp.Kvs))
	getNewItem := getNewItemFunc(listObj, v)

	total := int64(0)
	for _, kv := range resp.Kvs {
		obj := getNewItem()
		if err = json.Unmarshal(kv.Value, &obj); err != nil {
			return err
		}

		v.Set(reflect.Append(v, reflect.ValueOf(obj).Elem()))

		total += 1
	}

	if v.IsNil() {
		// Ensure that we never return a nil Items pointer in the result for consistency.
		v.Set(reflect.MakeSlice(v.Type(), 0, 0))
	}

	if vv, ok := listObj.(api.ListInterface); ok {
		vv.SetTotal(total)
	}
	s.Scheme.Default(listObj)

	return nil
}

func (s *Storage) Get(ctx context.Context, key string, out api.Object) error {
	key, err := s.prepareKey(key)
	if err != nil {
		return err
	}

	resp, err := s.db.Get(ctx, key)
	if err != nil {
		return err
	}
	if err = json.Unmarshal(resp.Kvs[0].Value, out); err != nil {
		return err
	}
	s.Scheme.Default(out)

	return nil
}

func (s *Storage) Create(ctx context.Context, key string, obj api.Object, ttl int64) error {
	key, err := s.prepareKey(key)
	if err != nil {
		return err
	}

	data, err := json.Marshal(obj)
	if err != nil {
		return err
	}

	options := []backend.PutOption{}
	if ttl > 0 {
		options = append(options, backend.PutDuration(time.Duration(ttl)))
	}

	if err = s.db.Put(ctx, key, data, options...); err != nil {
		return err
	}

	return nil
}

func (s *Storage) Update(ctx context.Context, key string, obj api.Object, ttl int64) error {
	key, err := s.prepareKey(key)
	if err != nil {
		return err
	}

	data, err := json.Marshal(obj)
	if err != nil {
		return err
	}

	options := []backend.PutOption{}
	if ttl > 0 {
		options = append(options, backend.PutDuration(time.Duration(ttl)))
	}

	if err = s.db.Put(ctx, key, data, options...); err != nil {
		return err
	}

	return nil
}

func (s *Storage) Delete(ctx context.Context, key string) error {
	key, err := s.prepareKey(key)
	if err != nil {
		return err
	}

	if err := s.db.Del(ctx, key); err != nil {
		return err
	}

	return nil
}

func (s *Storage) prepareKey(key string) (string, error) {
	if key == ".." ||
		strings.HasPrefix(key, "../") ||
		strings.HasSuffix(key, "/..") ||
		strings.Contains(key, "/../") {
		return "", fmt.Errorf("invalid key: %q", key)
	}
	if key == "." ||
		strings.HasPrefix(key, "./") ||
		strings.HasSuffix(key, "/.") ||
		strings.Contains(key, "/./") {
		return "", fmt.Errorf("invalid key: %q", key)
	}
	if key == "" || key == "/" {
		return "", fmt.Errorf("empty key: %q", key)
	}
	// We ensured that pathPrefix ends in '/' in construction, so skip any leading '/' in the key now.
	startIndex := 0
	if key[0] == '/' {
		startIndex = 1
	}
	return s.pathPrefix + key[startIndex:], nil
}

func getNewItemFunc(listObj api.Object, v reflect.Value) func() api.Object {
	// Otherwise just instantiate an empty item
	elem := v.Type().Elem()
	return func() api.Object {
		return reflect.New(elem).Interface().(api.Object)
	}
}

// growSlice takes a slice value and grows its capacity up
// to the maximum of the passed sizes or maxCapacity, whichever
// is smaller. Above maxCapacity decisions about allocation are left
// to the Go runtime on append. This allows a caller to make an
// educated guess about the potential size of the total list while
// still avoiding overly aggressive initial allocation. If sizes
// is empty maxCapacity will be used as the size to grow.
func growSlice(v reflect.Value, maxCapacity int, sizes ...int) {
	cap := v.Cap()
	max := cap
	for _, size := range sizes {
		if size > max {
			max = size
		}
	}
	if len(sizes) == 0 || max > maxCapacity {
		max = maxCapacity
	}
	if max <= cap {
		return
	}
	if v.Len() > 0 {
		extra := reflect.MakeSlice(v.Type(), v.Len(), max)
		reflect.Copy(extra, v)
		v.Set(extra)
	} else {
		extra := reflect.MakeSlice(v.Type(), 0, max)
		v.Set(extra)
	}
}
