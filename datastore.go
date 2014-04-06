package nds

import (
	"appengine"
	"appengine/datastore"
	"errors"
	"reflect"
)

const (
	// multiLimit is the App Engine datastore limit for the number of entities
	// that can be PutMulti or GetMulti in one call.
	multiLimit = 1000
)

var (
	// milMultiError is a convenience slice used to represent a nil error when
	// grouping erros in GetMulti.
	nilMultiError = make(appengine.MultiError, multiLimit)
)

func Get(c appengine.Context, key *datastore.Key, dst interface{}) error {
	return datastore.Get(c, key, dst)
}

// GetMulti works just like datastore.GetMulti except it calls
// datastore.GetMulti as many times as required to complete a request of over
// 1000 entities.
func GetMulti(c appengine.Context,
	keys []*datastore.Key, dst interface{}) error {

	v := reflect.ValueOf(dst)
	if v.Kind() != reflect.Slice {
		return errors.New("nds: dst is not a slice")
	}

	if len(keys) != v.Len() {
		return errors.New("nds: key and dst slices have different length")
	}

	if len(keys) == 0 {
		return nil
	}

	p := len(keys) / multiLimit
	errs := make([]error, 0, p+1)
	for i := 0; i < p; i++ {
		keySlice := keys[i*multiLimit : (i+1)*multiLimit]
		dstSlice := v.Slice(i*multiLimit, (i+1)*multiLimit)
		err := datastore.GetMulti(c, keySlice, dstSlice.Interface())
		errs = append(errs, err)
	}

	if len(keys)%multiLimit != 0 {
		keySlice := keys[p*multiLimit : len(keys)]
		dstSlice := v.Slice(p*multiLimit, len(keys))
		err := datastore.GetMulti(c, keySlice, dstSlice.Interface())
		errs = append(errs, err)
	}

	// Quick escape if all errors are nil.
	errsNil := true
	for _, err := range errs {
		if err != nil {
			errsNil = false
		}
	}
	if errsNil {
		return nil
	}

	groupedErrs := make(appengine.MultiError, 0, len(keys))
	for _, err := range errs {
		if err == nil {
			groupedErrs = append(groupedErrs, nilMultiError...)
		} else if me, ok := err.(appengine.MultiError); ok {
			groupedErrs = append(groupedErrs, me...)
		} else {
			return err
		}
	}
	return groupedErrs[:len(keys)]
}

func Put(c appengine.Context,
	key *datastore.Key, src interface{}) (*datastore.Key, error) {
	return datastore.Put(c, key, src)
}