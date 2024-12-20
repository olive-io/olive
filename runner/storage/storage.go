package storage

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	json "github.com/bytedance/sonic"
	"k8s.io/apimachinery/pkg/conversion"
	krt "k8s.io/apimachinery/pkg/runtime"

	"github.com/olive-io/olive/apis"
	"github.com/olive-io/olive/runner/storage/backend"
)

var (
	errExpectFieldItems = errors.New("no Items field in this object")
	errExpectSliceItems = errors.New("items field must be a slice of objects")
)

type Storage struct {
	*apis.ReflectScheme

	pathPrefix string

	db backend.IBackend
}

func New(scheme *apis.ReflectScheme, db backend.IBackend) *Storage {
	storage := &Storage{
		ReflectScheme: scheme,
		pathPrefix:    "/rt",
		db:            db,
	}

	return storage
}

func (s *Storage) GenerateKey(obj krt.Object) string {
	s.ReflectScheme.SetTypesGVK(obj)
	gvk := obj.GetObjectKind().GroupVersionKind()
	if strings.HasSuffix(gvk.Kind, "List") {
		gvk.Kind = strings.TrimSuffix(gvk.Kind, "List")
	}

	key := gvk.String()
	return key
}

func (s *Storage) GetList(ctx context.Context, key string, listObj krt.Object) error {
	listPtr, err := GetItemsPtr(listObj)
	if err != nil {
		return err
	}
	v, err := conversion.EnforcePtr(listPtr)
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

	s.ReflectScheme.Default(listObj)

	return nil
}

func (s *Storage) Get(ctx context.Context, key string, out krt.Object) error {
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
	s.ReflectScheme.Default(out)

	return nil
}

func (s *Storage) Create(ctx context.Context, key string, obj krt.Object, ttl int64) error {
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

func (s *Storage) Update(ctx context.Context, key string, obj krt.Object, ttl int64) error {
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

// GetItemsPtr returns a pointer to the list object's Items member.
// If 'list' doesn't have an Items member, it's not really a list type
// and an error will be returned.
// This function will either return a pointer to a slice, or an error, but not both.
// TODO: this will be replaced with an interface in the future
func GetItemsPtr(list krt.Object) (interface{}, error) {
	obj, err := getItemsPtr(list)
	if err != nil {
		return nil, fmt.Errorf("%T is not a list: %v", list, err)
	}
	return obj, nil
}

// getItemsPtr returns a pointer to the list object's Items member or an error.
func getItemsPtr(list krt.Object) (interface{}, error) {
	v, err := conversion.EnforcePtr(list)
	if err != nil {
		return nil, err
	}

	items := v.FieldByName("Items")
	if !items.IsValid() {
		return nil, errExpectFieldItems
	}
	switch items.Kind() {
	case reflect.Interface, reflect.Pointer:
		target := reflect.TypeOf(items.Interface()).Elem()
		if target.Kind() != reflect.Slice {
			return nil, errExpectSliceItems
		}
		return items.Interface(), nil
	case reflect.Slice:
		return items.Addr().Interface(), nil
	default:
		return nil, errExpectSliceItems
	}
}

func getNewItemFunc(listObj krt.Object, v reflect.Value) func() krt.Object {
	// Otherwise just instantiate an empty item
	elem := v.Type().Elem()
	return func() krt.Object {
		return reflect.New(elem).Interface().(krt.Object)
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
