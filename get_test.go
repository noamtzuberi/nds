package nds_test

import (
	"io"
	"reflect"
	"testing"

	"github.com/qedus/nds"

	"errors"

	"appengine"
	"appengine/aetest"
	"appengine/datastore"
	"appengine/memcache"
)

func TestGetMultiStruct(t *testing.T) {
	c, err := aetest.NewContext(nil)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	type testEntity struct {
		IntVal int64
	}

	keys := []*datastore.Key{}
	entities := []testEntity{}
	for i := int64(1); i < 3; i++ {
		keys = append(keys, datastore.NewKey(c, "Entity", "", i, nil))
		entities = append(entities, testEntity{i})
	}

	if _, err := nds.PutMulti(c, keys, entities); err != nil {
		t.Fatal(err)
	}

	// Get from datastore.
	response := make([]testEntity, len(keys))
	if err := nds.GetMulti(c, keys, response); err != nil {
		t.Fatal(err)
	}
	for i := int64(0); i < 2; i++ {
		if response[i].IntVal != i+1 {
			t.Fatal("incorrect IntVal")
		}
	}

	// Get from cache.
	response = make([]testEntity, len(keys))
	if err := nds.GetMulti(c, keys, response); err != nil {
		t.Fatal(err)
	}
	for i := int64(0); i < 2; i++ {
		if response[i].IntVal != i+1 {
			t.Fatal("incorrect IntVal")
		}
	}
}

func TestGetMultiStructPtr(t *testing.T) {
	c, err := aetest.NewContext(nil)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	type testEntity struct {
		IntVal int64
	}

	keys := []*datastore.Key{}
	entities := []testEntity{}
	for i := int64(1); i < 3; i++ {
		keys = append(keys, datastore.NewKey(c, "Entity", "", i, nil))
		entities = append(entities, testEntity{i})
	}

	if _, err := nds.PutMulti(c, keys, entities); err != nil {
		t.Fatal(err)
	}

	// Get from datastore.
	response := make([]*testEntity, len(keys))
	for i := 0; i < len(response); i++ {
		response[i] = &testEntity{}
	}

	if err := nds.GetMulti(c, keys, response); err != nil {
		t.Fatal(err)
	}
	for i := int64(0); i < 2; i++ {
		if response[i].IntVal != i+1 {
			t.Fatal("incorrect IntVal")
		}
	}

	// Get from cache.
	response = make([]*testEntity, len(keys))
	for i := 0; i < len(response); i++ {
		response[i] = &testEntity{}
	}
	if err := nds.GetMulti(c, keys, response); err != nil {
		t.Fatal(err)
	}
	for i := int64(0); i < 2; i++ {
		if response[i].IntVal != i+1 {
			t.Fatal("incorrect IntVal")
		}
	}
}

func TestGetMultiInterface(t *testing.T) {
	c, err := aetest.NewContext(nil)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	type testEntity struct {
		IntVal int64
	}

	keys := []*datastore.Key{}
	entities := []testEntity{}
	for i := int64(1); i < 3; i++ {
		keys = append(keys, datastore.NewKey(c, "Entity", "", i, nil))
		entities = append(entities, testEntity{i})
	}

	if _, err := nds.PutMulti(c, keys, entities); err != nil {
		t.Fatal(err)
	}

	// Get from datastore.
	response := make([]interface{}, len(keys))
	for i := 0; i < len(response); i++ {
		response[i] = &testEntity{}
	}

	if err := nds.GetMulti(c, keys, response); err != nil {
		t.Fatal(err)
	}
	for i := int64(0); i < 2; i++ {
		if te, ok := response[i].(*testEntity); ok {
			if te.IntVal != i+1 {
				t.Fatal("incorrect IntVal")
			}
		} else {
			t.Fatal("incorrect type")
		}
	}

	// Get from cache.
	response = make([]interface{}, len(keys))
	for i := 0; i < len(response); i++ {
		response[i] = &testEntity{}
	}
	if err := nds.GetMulti(c, keys, response); err != nil {
		t.Fatal(err)
	}
	for i := int64(0); i < 2; i++ {
		if te, ok := response[i].(*testEntity); ok {
			if te.IntVal != i+1 {
				t.Fatal("incorrect IntVal")
			}
		} else {
			t.Fatal("incorrect type")
		}
	}
}

func TestGetMultiPropertyLoadSaver(t *testing.T) {
	c, err := aetest.NewContext(nil)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	type testEntity struct {
		IntVal int
	}

	keys := []*datastore.Key{}
	entities := []datastore.PropertyList{}

	for i := 1; i < 3; i++ {
		keys = append(keys, datastore.NewKey(c, "Entity", "", int64(i), nil))

		pl := datastore.PropertyList{}
		if err := nds.SaveStruct(&testEntity{i}, &pl); err != nil {
			t.Fatal(err)
		}
		entities = append(entities, pl)
	}

	if _, err := nds.PutMulti(c, keys, entities); err != nil {
		t.Fatal(err)
	}

	// Prime the cache..
	uncachedEntities := make([]datastore.PropertyList, len(keys))
	if err := nds.GetMulti(c, keys, uncachedEntities); err != nil {
		t.Fatal(err)
	}

	for i, e := range entities {
		if !reflect.DeepEqual(e, uncachedEntities[i]) {
			t.Fatal("uncachedEntities not equal", e, uncachedEntities[i])
		}
	}

	// Use cache.
	cachedEntities := make([]datastore.PropertyList, len(keys))
	if err := nds.GetMulti(c, keys, cachedEntities); err != nil {
		t.Fatal(err)
	}

	for i, e := range entities {
		if !reflect.DeepEqual(e, cachedEntities[i]) {
			t.Fatal("cachedEntities not equal", e, cachedEntities[i])
		}
	}

	// We know the datastore supports property load saver but we need to make
	// sure that memcache does by ensuring memcache does not error when we
	// change to fetching with structs.
	// Do this by making sure the datastore is not called on this following
	// GetMulti as memcache should have worked.
	nds.SetDatastoreGetMulti(func(c appengine.Context,
		keys []*datastore.Key, vals interface{}) error {
		if len(keys) != 0 {
			return errors.New("should not be called")
		}
		return nil
	})
	defer func() {
		nds.SetDatastoreGetMulti(datastore.GetMulti)
	}()
	tes := make([]testEntity, len(entities))
	if err := nds.GetMulti(c, keys, tes); err != nil {
		t.Fatal(err)
	}
}

func TestGetMultiNoKeys(t *testing.T) {
	c, err := aetest.NewContext(nil)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	type testEntity struct {
		IntVal int64
	}

	keys := []*datastore.Key{}
	entities := []testEntity{}

	if err := nds.GetMulti(c, keys, entities); err != nil {
		t.Fatal(err)
	}
}

func TestGetMultiInterfaceError(t *testing.T) {
	c, err := aetest.NewContext(nil)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	type testEntity struct {
		IntVal int64
	}

	keys := []*datastore.Key{}
	entities := []testEntity{}
	for i := int64(1); i < 3; i++ {
		keys = append(keys, datastore.NewKey(c, "Entity", "", i, nil))
		entities = append(entities, testEntity{i})
	}

	if _, err := nds.PutMulti(c, keys, entities); err != nil {
		t.Fatal(err)
	}

	// Get from datastore.
	// No errors expected.
	response := []interface{}{&testEntity{}, &testEntity{}}

	if err := nds.GetMulti(c, keys, response); err != nil {
		t.Fatal(err)
	}
	for i := int64(0); i < 2; i++ {
		if te, ok := response[i].(*testEntity); ok {
			if te.IntVal != i+1 {
				t.Fatal("incorrect IntVal")
			}
		} else {
			t.Fatal("incorrect type")
		}
	}

	// Get from cache.
	// Errors expected.
	response = []interface{}{&testEntity{}, testEntity{}}
	if err := nds.GetMulti(c, keys, response); err == nil {
		t.Fatal("expected invalid entity type error")
	}
}

// This is just used to ensure interfaces don't currently work.
type readerTestEntity struct {
	IntVal int
}

func (rte readerTestEntity) Read(p []byte) (n int, err error) {
	return 1, nil
}

var _ io.Reader = readerTestEntity{}

func newReaderTestEntity() io.Reader {
	return readerTestEntity{}
}

func TestGetArgs(t *testing.T) {
	c, err := aetest.NewContext(nil)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	type testEntity struct {
		IntVal int64
	}

	if err := nds.Get(c, nil, &testEntity{}); err == nil {
		t.Fatal("expected error for nil key")
	}

	key := datastore.NewKey(c, "Entity", "", 1, nil)
	if err := nds.Get(c, key, nil); err == nil {
		t.Fatal("expected error for nil value")
	}

	if err := nds.Get(c, key, datastore.PropertyList{}); err == nil {
		t.Fatal("expected error for datastore.PropertyList")
	}

	if err := nds.Get(c, key, testEntity{}); err == nil {
		t.Fatal("expected error for struct")
	}

	rte := newReaderTestEntity()
	if err := nds.Get(c, key, rte); err == nil {
		t.Fatal("expected error for interface")
	}
}

func TestGetMultiArgs(t *testing.T) {
	c, err := aetest.NewContext(nil)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	type testEntity struct {
		IntVal int64
	}

	key := datastore.NewKey(c, "Entity", "", 1, nil)
	keys := []*datastore.Key{key}
	val := testEntity{}
	if err := nds.GetMulti(c, keys, nil); err == nil {
		t.Fatal("expected error for nil vals")
	}
	structVals := []testEntity{val}
	if err := nds.GetMulti(c, nil, structVals); err == nil {
		t.Fatal("expected error for nil keys")
	}

	if err := nds.GetMulti(c, keys, []testEntity{}); err == nil {
		t.Fatal("expected error for unequal keys and vals")
	}

	if err := nds.GetMulti(c, keys, datastore.PropertyList{}); err == nil {
		t.Fatal("expected error for propertyList")
	}

	rte := newReaderTestEntity()
	if err := nds.GetMulti(c, keys, []io.Reader{rte}); err == nil {
		t.Fatal("expected error for interface")
	}
}

func TestGetSliceProperty(t *testing.T) {
	c, err := aetest.NewContext(nil)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	type testEntity struct {
		IntVals []int64
	}

	key := datastore.NewKey(c, "Entity", "", 1, nil)
	intVals := []int64{0, 1, 2, 3}
	val := &testEntity{intVals}

	if _, err := nds.Put(c, key, val); err != nil {
		t.Fatal(err)
	}

	// Get from datastore.
	newVal := &testEntity{}
	if err := nds.Get(c, key, newVal); err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(val.IntVals, intVals) {
		t.Fatal("slice properties not equal", val.IntVals)
	}

	// Get from memcache.
	newVal = &testEntity{}
	if err := nds.Get(c, key, newVal); err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(val.IntVals, intVals) {
		t.Fatal("slice properties not equal", val.IntVals)
	}
}

func TestGetMultiNoPropertyList(t *testing.T) {
	c, err := aetest.NewContext(nil)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	keys := []*datastore.Key{datastore.NewKey(c, "Test", "", 1, nil)}
	pl := datastore.PropertyList{datastore.Property{}}

	if err := nds.GetMulti(c, keys, pl); err == nil {
		t.Fatal("expecting no PropertyList error")
	}
}

func TestGetMultiNonStruct(t *testing.T) {
	c, err := aetest.NewContext(nil)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	keys := []*datastore.Key{datastore.NewKey(c, "Test", "", 1, nil)}
	vals := []int{12}

	if err := nds.GetMulti(c, keys, vals); err == nil {
		t.Fatal("expecting unsupported vals type")
	}
}

// TestGetMemcacheFail makes sure that memcache failure does not stop Get from
// working.
func TestGetMemcacheFail(t *testing.T) {
	c, err := aetest.NewContext(nil)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	type testEntity struct {
		IntVal int64
	}

	key := datastore.NewKey(c, "Entity", "", 1, nil)
	val := &testEntity{3}

	if _, err := nds.Put(c, key, val); err != nil {
		t.Fatal(err)
	}

	nds.SetMemcacheAddMulti(func(c appengine.Context,
		items []*memcache.Item) error {
		return errors.New("expected memcache.AddMulti error")
	})
	nds.SetMemcacheCompareAndSwapMulti(func(c appengine.Context,
		items []*memcache.Item) error {
		return errors.New("expected memcache.ComapreAndSwap error")
	})
	nds.SetMemcacheGetMulti(func(c appengine.Context,
		keys []string) (map[string]*memcache.Item, error) {
		return nil, errors.New("expected memcache.GetMulti error")
	})
	nds.SetMemcacheSetMulti(func(c appengine.Context,
		items []*memcache.Item) error {
		return errors.New("expected memcache.SetMulti error")
	})

	defer func() {
		nds.SetMemcacheAddMulti(nds.ZeroMemcacheAddMulti)
		nds.SetMemcacheCompareAndSwapMulti(nds.ZeroMemcacheCompareAndSwapMulti)
		nds.SetMemcacheGetMulti(nds.ZeroMemcacheGetMulti)
		nds.SetMemcacheSetMulti(nds.ZeroMemcacheSetMulti)
	}()

	retVal := &testEntity{}
	if err := nds.Get(c, key, retVal); err != nil {
		t.Fatal(err)
	}
	if val.IntVal != retVal.IntVal {
		t.Fatal("val and retVal not equal")
	}
}

func TestGetMultiDatastoreFail(t *testing.T) {
	c, err := aetest.NewContext(nil)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	type testEntity struct {
		IntVal int64
	}

	keys := []*datastore.Key{}
	entities := []testEntity{}
	for i := int64(1); i < 3; i++ {
		keys = append(keys, datastore.NewKey(c, "Entity", "", i, nil))
		entities = append(entities, testEntity{i})
	}

	if _, err := nds.PutMulti(c, keys, entities); err != nil {
		t.Fatal(err)
	}

	nds.SetDatastoreGetMulti(func(c appengine.Context,
		keys []*datastore.Key, vals interface{}) error {
		return errors.New("expected datastore.GetMulti error")
	})
	defer nds.SetDatastoreGetMulti(datastore.GetMulti)

	// Get from datastore.
	response := make([]testEntity, len(keys))
	if err := nds.GetMulti(c, keys, response); err == nil {
		t.Fatal("expected GetMulti to fail")
	}
}

func TestGetMultiMemcacheCorrupt(t *testing.T) {
	c, err := aetest.NewContext(nil)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	type testEntity struct {
		IntVal int64
	}

	keys := []*datastore.Key{}
	entities := []testEntity{}
	for i := int64(1); i < 3; i++ {
		keys = append(keys, datastore.NewKey(c, "Entity", "", i, nil))
		entities = append(entities, testEntity{i})
	}

	if _, err := nds.PutMulti(c, keys, entities); err != nil {
		t.Fatal(err)
	}

	// Charge memcache.
	if err := nds.GetMulti(c, keys, make([]testEntity, len(keys))); err != nil {
		t.Fatal(err)
	}

	nds.SetMemcacheGetMulti(func(c appengine.Context,
		keys []string) (map[string]*memcache.Item, error) {
		items, err := memcache.GetMulti(c, keys)

		// Skip over lockMemcaheKeys GetMulti.
		if len(keys) != 0 {
			// Corrupt second item.
			items[keys[1]].Value = []byte("corrupt string")
		}

		return items, err
	})
	defer nds.SetMemcacheGetMulti(nds.ZeroMemcacheGetMulti)

	// Get from datastore.
	response := make([]testEntity, len(keys))
	if err := nds.GetMulti(c, keys, response); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < len(keys); i++ {
		if entities[i].IntVal != response[i].IntVal {
			t.Fatal("IntVal not equal", entities[i].IntVal, response[i].IntVal)
		}
	}
}

func TestGetMultiMemcacheFlagCorrupt(t *testing.T) {
	c, err := aetest.NewContext(nil)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	type testEntity struct {
		IntVal int64
	}

	keys := []*datastore.Key{}
	entities := []testEntity{}
	for i := int64(1); i < 3; i++ {
		keys = append(keys, datastore.NewKey(c, "Entity", "", i, nil))
		entities = append(entities, testEntity{i})
	}

	if _, err := nds.PutMulti(c, keys, entities); err != nil {
		t.Fatal(err)
	}

	// Charge memcache.
	if err := nds.GetMulti(c, keys, make([]testEntity, len(keys))); err != nil {
		t.Fatal(err)
	}

	nds.SetMemcacheGetMulti(func(c appengine.Context,
		keys []string) (map[string]*memcache.Item, error) {
		items, err := memcache.GetMulti(c, keys)

		// Skip over lockMemcaheKeys GetMulti.
		if len(keys) != 0 {
			// Corrupt second item.
			items[keys[1]].Flags = 56
		}

		return items, err
	})
	defer nds.SetMemcacheGetMulti(nds.ZeroMemcacheGetMulti)

	// Get from datastore.
	response := make([]testEntity, len(keys))
	if err := nds.GetMulti(c, keys, response); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < len(keys); i++ {
		if entities[i].IntVal != response[i].IntVal {
			t.Fatal("IntVal not equal")
		}
	}
}

func errMemcacheGetMulti(c appengine.Context, keys []string) (
	map[string]*memcache.Item, error) {

	return nil, errors.New("test triggered memcache.GetMulti")
}

func TestGetMultiLockMemcacheFail(t *testing.T) {
	c, err := aetest.NewContext(nil)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	type testEntity struct {
		IntVal int64
	}

	keys := []*datastore.Key{}
	entities := []testEntity{}
	for i := int64(1); i < 3; i++ {
		keys = append(keys, datastore.NewKey(c, "Entity", "", i, nil))
		entities = append(entities, testEntity{i})
	}

	if _, err := nds.PutMulti(c, keys, entities); err != nil {
		t.Fatal(err)
	}

	memcacheGetChan := make(chan func(c appengine.Context, keys []string) (
		map[string]*memcache.Item, error), 2)
	memcacheGetChan <- nds.ZeroMemcacheGetMulti
	memcacheGetChan <- errMemcacheGetMulti

	nds.SetMemcacheGetMulti(func(c appengine.Context,
		keys []string) (map[string]*memcache.Item, error) {
		f := <-memcacheGetChan
		return f(c, keys)
	})
	defer nds.SetMemcacheGetMulti(nds.ZeroMemcacheGetMulti)
	response := make([]testEntity, len(keys))
	if err := nds.GetMulti(c, keys, response); err != nil {
		t.Fatal(err)
	}

	for i := 0; i < len(keys); i++ {
		if entities[i].IntVal != response[i].IntVal {
			t.Fatal("IntVal not equal")
		}
	}
}

func TestGetMultiLockItemValueFail(t *testing.T) {
	c, err := aetest.NewContext(nil)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	type testEntity struct {
		IntVal int64
	}

	keys := []*datastore.Key{}
	entities := []testEntity{}
	for i := int64(1); i < 3; i++ {
		keys = append(keys, datastore.NewKey(c, "Entity", "", i, nil))
		entities = append(entities, testEntity{i})
	}

	if _, err := nds.PutMulti(c, keys, entities); err != nil {
		t.Fatal(err)
	}

	memcacheGetChan := make(chan func(c appengine.Context, keys []string) (
		map[string]*memcache.Item, error), 2)
	memcacheGetChan <- nds.ZeroMemcacheGetMulti
	memcacheGetChan <- func(c appengine.Context,
		keys []string) (map[string]*memcache.Item, error) {
		items, err := nds.ZeroMemcacheGetMulti(c, keys)
		if err != nil {
			return nil, err
		}
		items[keys[0]].Value = []byte("random1")
		items[keys[1]].Value = []byte("random2")
		return items, nil
	}
	nds.SetMemcacheGetMulti(func(c appengine.Context,
		keys []string) (map[string]*memcache.Item, error) {
		f := <-memcacheGetChan
		return f(c, keys)
	})
	defer nds.SetMemcacheGetMulti(nds.ZeroMemcacheGetMulti)
	response := make([]testEntity, len(keys))
	if err := nds.GetMulti(c, keys, response); err != nil {
		t.Fatal(err)
	}

	for i := 0; i < len(keys); i++ {
		if entities[i].IntVal != response[i].IntVal {
			t.Fatal("IntVal not equal")
		}
	}
}

func TestGetMultiLockItemNoneFlags(t *testing.T) {
	c, err := aetest.NewContext(nil)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	type testEntity struct {
		IntVal int64
	}

	keys := []*datastore.Key{}
	entities := []testEntity{}
	for i := int64(1); i < 3; i++ {
		keys = append(keys, datastore.NewKey(c, "Entity", "", i, nil))
		entities = append(entities, testEntity{i})
	}

	if _, err := nds.PutMulti(c, keys, entities); err != nil {
		t.Fatal(err)
	}

	memcacheGetChan := make(chan func(c appengine.Context, keys []string) (
		map[string]*memcache.Item, error), 2)
	memcacheGetChan <- nds.ZeroMemcacheGetMulti
	memcacheGetChan <- func(c appengine.Context,
		keys []string) (map[string]*memcache.Item, error) {
		items, err := nds.ZeroMemcacheGetMulti(c, keys)
		if err != nil {
			return nil, err
		}
		items[keys[0]].Flags = nds.NoneItem
		items[keys[1]].Flags = nds.NoneItem
		return items, nil
	}
	nds.SetMemcacheGetMulti(func(c appengine.Context,
		keys []string) (map[string]*memcache.Item, error) {
		f := <-memcacheGetChan
		return f(c, keys)
	})
	defer nds.SetMemcacheGetMulti(nds.ZeroMemcacheGetMulti)

	response := make([]testEntity, len(keys))
	if err := nds.GetMulti(c, keys, response); err == nil {
		t.Fatal("expected error")
	} else if me, ok := err.(appengine.MultiError); !ok {
		t.Fatal("expected appengine.MultiError")
	} else {
		for _, e := range me {
			if e != datastore.ErrNoSuchEntity {
				t.Fatal("expected datastore.ErrNoSuchEntity")
			}
		}
	}
}

func TestGetMultiLockReturnEntityFail(t *testing.T) {
	c, err := aetest.NewContext(nil)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	type testEntity struct {
		IntVal int64
	}

	keys := []*datastore.Key{}
	entities := []testEntity{}
	for i := int64(1); i < 3; i++ {
		keys = append(keys, datastore.NewKey(c, "Entity", "", i, nil))
		entities = append(entities, testEntity{i})
	}

	if _, err := nds.PutMulti(c, keys, entities); err != nil {
		t.Fatal(err)
	}

	// Fail to unmarshal test.
	memcacheGetChan := make(chan func(c appengine.Context, keys []string) (
		map[string]*memcache.Item, error), 2)
	memcacheGetChan <- nds.ZeroMemcacheGetMulti
	memcacheGetChan <- func(c appengine.Context,
		keys []string) (map[string]*memcache.Item, error) {
		items, err := nds.ZeroMemcacheGetMulti(c, keys)
		if err != nil {
			return nil, err
		}
		items[keys[0]].Flags = nds.EntityItem
		items[keys[1]].Flags = nds.EntityItem
		return items, nil
	}
	nds.SetMemcacheGetMulti(func(c appengine.Context,
		keys []string) (map[string]*memcache.Item, error) {
		f := <-memcacheGetChan
		return f(c, keys)
	})

	response := make([]testEntity, len(keys))
	if err := nds.GetMulti(c, keys, response); err != nil {
		t.Fatal(err)
	}
	defer nds.SetMemcacheGetMulti(nds.ZeroMemcacheGetMulti)

	for i := 0; i < len(keys); i++ {
		if entities[i].IntVal != response[i].IntVal {
			t.Fatal("IntVal not equal")
		}
	}
}

func TestGetMultiLockReturnEntitySetValueFail(t *testing.T) {
	c, err := aetest.NewContext(nil)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	type testEntity struct {
		IntVal int64
	}

	keys := []*datastore.Key{}
	entities := []testEntity{}
	for i := int64(1); i < 3; i++ {
		keys = append(keys, datastore.NewKey(c, "Entity", "", i, nil))
		entities = append(entities, testEntity{i})
	}

	if _, err := nds.PutMulti(c, keys, entities); err != nil {
		t.Fatal(err)
	}

	// Fail to unmarshal test.
	memcacheGetChan := make(chan func(c appengine.Context, keys []string) (
		map[string]*memcache.Item, error), 2)
	memcacheGetChan <- nds.ZeroMemcacheGetMulti
	memcacheGetChan <- func(c appengine.Context,
		keys []string) (map[string]*memcache.Item, error) {
		items, err := nds.ZeroMemcacheGetMulti(c, keys)
		if err != nil {
			return nil, err
		}
		pl := datastore.PropertyList{
			datastore.Property{"One", 1, false, false},
		}
		value, err := nds.MarshalPropertyList(pl)
		if err != nil {
			return nil, err
		}
		items[keys[0]].Flags = nds.EntityItem
		items[keys[0]].Value = value
		items[keys[1]].Flags = nds.EntityItem
		items[keys[1]].Value = value
		return items, nil
	}
	nds.SetMemcacheGetMulti(func(c appengine.Context,
		keys []string) (map[string]*memcache.Item, error) {
		f := <-memcacheGetChan
		return f(c, keys)
	})

	response := make([]testEntity, len(keys))
	if err := nds.GetMulti(c, keys, response); err != nil {
		t.Fatal(err)
	}
	defer nds.SetMemcacheGetMulti(nds.ZeroMemcacheGetMulti)

	for i := 0; i < len(keys); i++ {
		if entities[i].IntVal != response[i].IntVal {
			t.Fatal("IntVal not equal")
		}
	}
}

func TestGetMultiLockReturnEntity(t *testing.T) {
	c, err := aetest.NewContext(nil)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	type testEntity struct {
		IntVal int64
	}

	keys := []*datastore.Key{}
	entities := []testEntity{}
	for i := int64(1); i < 3; i++ {
		keys = append(keys, datastore.NewKey(c, "Entity", "", i, nil))
		entities = append(entities, testEntity{i})
	}

	if _, err := nds.PutMulti(c, keys, entities); err != nil {
		t.Fatal(err)
	}

	memcacheGetChan := make(chan func(c appengine.Context, keys []string) (
		map[string]*memcache.Item, error), 2)
	memcacheGetChan <- nds.ZeroMemcacheGetMulti
	memcacheGetChan <- func(c appengine.Context,
		keys []string) (map[string]*memcache.Item, error) {
		items, err := nds.ZeroMemcacheGetMulti(c, keys)
		if err != nil {
			return nil, err
		}
		pl := datastore.PropertyList{
			datastore.Property{"IntVal", int64(5), false, false},
		}
		value, err := nds.MarshalPropertyList(pl)
		if err != nil {
			return nil, err
		}
		items[keys[0]].Flags = nds.EntityItem
		items[keys[0]].Value = value
		items[keys[1]].Flags = nds.EntityItem
		items[keys[1]].Value = value
		return items, nil
	}
	nds.SetMemcacheGetMulti(func(c appengine.Context,
		keys []string) (map[string]*memcache.Item, error) {
		f := <-memcacheGetChan
		return f(c, keys)
	})

	response := make([]testEntity, len(keys))
	if err := nds.GetMulti(c, keys, response); err != nil {
		t.Fatal(err)
	}
	defer nds.SetMemcacheGetMulti(nds.ZeroMemcacheGetMulti)

	for i := 0; i < len(keys); i++ {
		if 5 != response[i].IntVal {
			t.Fatal("IntVal not equal")
		}
	}
}

func TestGetMultiLockReturnUnknown(t *testing.T) {
	c, err := aetest.NewContext(nil)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	type testEntity struct {
		IntVal int64
	}

	keys := []*datastore.Key{}
	entities := []testEntity{}
	for i := int64(1); i < 3; i++ {
		keys = append(keys, datastore.NewKey(c, "Entity", "", i, nil))
		entities = append(entities, testEntity{i})
	}

	if _, err := nds.PutMulti(c, keys, entities); err != nil {
		t.Fatal(err)
	}

	memcacheGetChan := make(chan func(c appengine.Context, keys []string) (
		map[string]*memcache.Item, error), 2)
	memcacheGetChan <- nds.ZeroMemcacheGetMulti
	memcacheGetChan <- func(c appengine.Context,
		keys []string) (map[string]*memcache.Item, error) {
		items, err := nds.ZeroMemcacheGetMulti(c, keys)
		if err != nil {
			return nil, err
		}

		// Unknown lock values.
		items[keys[0]].Flags = 23
		items[keys[1]].Flags = 24
		return items, nil
	}
	nds.SetMemcacheGetMulti(func(c appengine.Context,
		keys []string) (map[string]*memcache.Item, error) {
		f := <-memcacheGetChan
		return f(c, keys)
	})

	response := make([]testEntity, len(keys))
	if err := nds.GetMulti(c, keys, response); err != nil {
		t.Fatal(err)
	}
	defer nds.SetMemcacheGetMulti(nds.ZeroMemcacheGetMulti)

	for i := 0; i < len(keys); i++ {
		if entities[i].IntVal != response[i].IntVal {
			t.Fatal("IntVal not equal")
		}
	}
}

func TestGetMultiLockReturnMiss(t *testing.T) {
	c, err := aetest.NewContext(nil)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	type testEntity struct {
		IntVal int64
	}

	keys := []*datastore.Key{}
	entities := []testEntity{}
	for i := int64(1); i < 3; i++ {
		keys = append(keys, datastore.NewKey(c, "Entity", "", i, nil))
		entities = append(entities, testEntity{i})
	}

	if _, err := nds.PutMulti(c, keys, entities); err != nil {
		t.Fatal(err)
	}

	memcacheGetChan := make(chan func(c appengine.Context, keys []string) (
		map[string]*memcache.Item, error), 2)
	memcacheGetChan <- nds.ZeroMemcacheGetMulti
	memcacheGetChan <- func(c appengine.Context,
		keys []string) (map[string]*memcache.Item, error) {
		items, err := nds.ZeroMemcacheGetMulti(c, keys)
		if err != nil {
			return nil, err
		}

		// Remove one item between memcache Add and Get.
		delete(items, keys[0])
		return items, nil
	}
	nds.SetMemcacheGetMulti(func(c appengine.Context,
		keys []string) (map[string]*memcache.Item, error) {
		f := <-memcacheGetChan
		return f(c, keys)
	})

	response := make([]testEntity, len(keys))
	if err := nds.GetMulti(c, keys, response); err != nil {
		t.Fatal(err)
	}
	defer nds.SetMemcacheGetMulti(nds.ZeroMemcacheGetMulti)

	for i := 0; i < len(keys); i++ {
		if entities[i].IntVal != response[i].IntVal {
			t.Fatal("IntVal not equal")
		}
	}
}

func TestGetMultiDatastoreMarshalFail(t *testing.T) {
	c, err := aetest.NewContext(nil)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	type testEntity struct {
		IntVal int64
	}

	keys := []*datastore.Key{}
	entities := []testEntity{}
	for i := int64(1); i < 3; i++ {
		keys = append(keys, datastore.NewKey(c, "Entity", "", i, nil))
		entities = append(entities, testEntity{i})
	}

	if _, err := nds.PutMulti(c, keys, entities); err != nil {
		t.Fatal(err)
	}

	nds.SetMarshal(func(pl datastore.PropertyList) ([]byte, error) {
		return nil, errors.New("expected marshal error")
	})

	response := make([]testEntity, len(keys))
	if err := nds.GetMulti(c, keys, response); err != nil {
		t.Fatal(err)
	}
	nds.SetMarshal(nds.MarshalPropertyList)

	for i := 0; i < len(keys); i++ {
		if entities[i].IntVal != response[i].IntVal {
			t.Fatal("IntVal not equal")
		}
	}
}

func TestGetMultiPaths(t *testing.T) {

	type memcacheGetMultiFunc func(c appengine.Context,
		keys []string) (map[string]*memcache.Item, error)

	type memcacheAddMultiFunc func(c appengine.Context,
		items []*memcache.Item) error

	type memcacheCompareAndSwapMultiFunc func(c appengine.Context,
		items []*memcache.Item) error

	type datastoreGetMultiFunc func(c appengine.Context,
		keys []*datastore.Key, vals interface{}) error

	expectedErr := errors.New("expected error")

	c, err := aetest.NewContext(nil)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	type testEntity struct {
		IntVal int64
	}

	keysVals := func(c appengine.Context, count int64) (
		[]*datastore.Key, []testEntity) {

		keys, vals := make([]*datastore.Key, count), make([]testEntity, count)
		for i := int64(0); i < count; i++ {
			keys[i] = datastore.NewKey(c, "Entity", "", i+1, nil)
			vals[i] = testEntity{i + 1}
		}
		return keys, vals
	}

	tests := []struct {
		description string

		count int64

		memcacheGetMultis           [2]memcacheGetMultiFunc
		memcacheAddMulti            memcacheAddMultiFunc
		memcacheCompareAndSwapMulti memcacheCompareAndSwapMultiFunc

		datastoreGetMulti datastoreGetMultiFunc

		expectedErr error
	}{
		{
			"no errors",
			20,
			[2]memcacheGetMultiFunc{
				nds.ZeroMemcacheGetMulti,
				nds.ZeroMemcacheGetMulti,
			},
			nds.ZeroMemcacheAddMulti,
			nds.ZeroMemcacheCompareAndSwapMulti,
			datastore.GetMulti,

			nil,
		},
		{
			"datastore unknown multierror",
			2,
			[2]memcacheGetMultiFunc{
				nds.ZeroMemcacheGetMulti,
				nds.ZeroMemcacheGetMulti,
			},
			nds.ZeroMemcacheAddMulti,
			nds.ZeroMemcacheCompareAndSwapMulti,
			func(c appengine.Context,
				keys []*datastore.Key, vals interface{}) error {

				me := make(appengine.MultiError, len(keys))
				for i := range me {
					me[i] = expectedErr
				}
				return me
			},

			appengine.MultiError{expectedErr, expectedErr},
		},
	}

	for _, test := range tests {
		t.Log("Start", test.description)

		keys, putVals := keysVals(c, test.count)
		if _, err := nds.PutMulti(c, keys, putVals); err != nil {
			t.Fatal(err)
		}

		memcacheGetChan := make(chan memcacheGetMultiFunc,
			len(test.memcacheGetMultis))

		for _, fn := range test.memcacheGetMultis {
			memcacheGetChan <- fn
		}

		nds.SetMemcacheGetMulti(func(c appengine.Context, keys []string) (
			map[string]*memcache.Item, error) {
			fn := <-memcacheGetChan
			return fn(c, keys)
		})

		nds.SetMemcacheAddMulti(test.memcacheAddMulti)
		nds.SetMemcacheCompareAndSwapMulti(test.memcacheCompareAndSwapMulti)

		nds.SetDatastoreGetMulti(test.datastoreGetMulti)

		getVals := make([]testEntity, test.count)
		err := nds.GetMulti(c, keys, getVals)

		// Reset App Engine API calls.
		nds.SetMemcacheGetMulti(nds.ZeroMemcacheGetMulti)
		nds.SetMemcacheAddMulti(nds.ZeroMemcacheAddMulti)
		nds.SetMemcacheCompareAndSwapMulti(nds.ZeroMemcacheCompareAndSwapMulti)
		nds.SetDatastoreGetMulti(datastore.GetMulti)

		if test.expectedErr == nil {
			if err != nil {
				t.Fatal(err)
			}

			for i := range getVals {
				if getVals[i].IntVal != putVals[i].IntVal {
					t.Fatal("incorrect IntVal")
				}
			}
		} else {
			if err == nil {
				t.Fatal("expected error")
			}
			expectedMultiErr, isMultiErr := test.expectedErr.(appengine.MultiError)

			if isMultiErr {
				me, ok := err.(appengine.MultiError)
				if !ok {
					t.Fatal("expected appengine.MultiError but got", err)
				}

				if len(me) != len(expectedMultiErr) {
					t.Fatal("appengine.MultiError length incorrect")
				}

				for i, e := range me {
					if e != expectedMultiErr[i] {
						t.Fatal("non matching errors", e, expectedMultiErr[i])
					}
				}
			}
		}

		if err := nds.DeleteMulti(c, keys); err != nil {
			t.Fatal(err)
		}
		t.Log("End", test.description)
	}
}
