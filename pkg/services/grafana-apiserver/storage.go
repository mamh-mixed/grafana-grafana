// SPDX-License-Identifier: AGPL-3.0-only
// Provenance-includes-location: https://github.com/kubernetes-sigs/apiserver-runtime/blob/main/pkg/experimental/storage/filepath/jsonfile_rest.go
// Provenance-includes-license: Apache-2.0
// Provenance-includes-copyright: The Kubernetes Authors.

package grafanaapiserver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/apiserver/pkg/storage"
	"k8s.io/apiserver/pkg/storage/storagebackend/factory"

	"github.com/grafana/grafana/pkg/services/store/entity"
)

var _ storage.Interface = (*Storage)(nil)

const MaxUpdateAttempts = 30

// Storage implements storage.Interface and storage resources as JSON files on disk.
type Storage struct {
	store entity.EntityStoreServer
	gr    schema.GroupResource
	codec runtime.Codec
	// keyFunc      func(obj runtime.Object) (string, error)
	// newFunc      func() runtime.Object
	// newListFunc  func() runtime.Object
	// getAttrsFunc storage.AttrFunc
	// trigger      storage.IndexerFuncs
	// indexers     *cache.Indexers

	// watchSet *WatchSet
}

// ErrFileNotExists means the file doesn't actually exist.
var ErrFileNotExists = fmt.Errorf("file doesn't exist")

// ErrNamespaceNotExists means the directory for the namespace doesn't actually exist.
var ErrNamespaceNotExists = errors.New("namespace does not exist")

func NewStorage(
	gr schema.GroupResource,
	store entity.EntityStoreServer,
	codec runtime.Codec,
) (storage.Interface, factory.DestroyFunc, error) {
	return &Storage{
		gr:    gr,
		codec: codec,
		store: store,
	}, nil, nil
}

// Create adds a new object at a key unless it already exists. 'ttl' is time-to-live
// in seconds (0 means forever). If no error is returned and out is not nil, out will be
// set to the read value from database.
func (s *Storage) Create(ctx context.Context, key string, obj runtime.Object, out runtime.Object, ttl uint64) error {
	return nil
}

// Delete removes the specified key and returns the value that existed at that spot.
// If key didn't exist, it will return NotFound storage error.
// If 'cachedExistingObject' is non-nil, it can be used as a suggestion about the
// current version of the object to avoid read operation from storage to get it.
// However, the implementations have to retry in case suggestion is stale.
func (s *Storage) Delete(
	ctx context.Context, key string, out runtime.Object, preconditions *storage.Preconditions,
	validateDeletion storage.ValidateObjectFunc, cachedExistingObject runtime.Object) error {
	return nil
}

// Watch begins watching the specified key. Events are decoded into API objects,
// and any items selected by 'p' are sent down to returned watch.Interface.
// resourceVersion may be used to specify what version to begin watching,
// which should be the current resourceVersion, and no longer rv+1
// (e.g. reconnecting without missing any updates).
// If resource version is "0", this interface will get current object at given key
// and send it in an "ADDED" event, before watch starts.
func (s *Storage) Watch(ctx context.Context, key string, opts storage.ListOptions) (watch.Interface, error) {
	return nil, apierrors.NewMethodNotSupported(schema.GroupResource{}, "watch")
}

// Get unmarshals object found at key into objPtr. On a not found error, will either
// return a zero object of the requested type, or an error, depending on 'opts.ignoreNotFound'.
// Treats empty responses and nil response nodes exactly like a not found error.
// The returned contents may be delayed, but it is guaranteed that they will
// match 'opts.ResourceVersion' according 'opts.ResourceVersionMatch'.
func (s *Storage) Get(ctx context.Context, key string, opts storage.GetOptions, objPtr runtime.Object) error {
	ctx, err := contextWithFakeGrafanaUser(ctx)
	if err != nil {
		return err
	}
	grn, err := keyToGRN(key, &s.gr)
	if err != nil {
		return err
	}

	rsp, err := s.store.Read(ctx, &entity.ReadEntityRequest{
		GRN:        grn,
		WithMeta:   true,
		WithBody:   true,
		WithStatus: true,
	})
	if err != nil {
		return err
	}
	if rsp.GRN == nil {
		return apierrors.NewNotFound(s.gr, grn.ResourceIdentifier)
	}

	res, err := enityToResource(rsp)
	if err != nil {
		return err
	}
	// HACK???  should be saved with the payload
	res.APIVersion = "core.kinds.grafana.com" + "/" + "v0-alpha" // << hardcoded
	res.Kind = grn.ResourceKind

	jjj, _ := json.Marshal(res)
	_, _, err = s.codec.Decode(jjj, nil, objPtr)

	fmt.Printf("k8s GET/GOT:%s (rv:%s)\n", res.Metadata.Name, res.Metadata.ResourceVersion)
	return err
}

// GetList unmarshalls objects found at key into a *List api object (an object
// that satisfies runtime.IsList definition).
// If 'opts.Recursive' is false, 'key' is used as an exact match. If `opts.Recursive'
// is true, 'key' is used as a prefix.
// The returned contents may be delayed, but it is guaranteed that they will
// match 'opts.ResourceVersion' according 'opts.ResourceVersionMatch'.
func (s *Storage) GetList(ctx context.Context, key string, opts storage.ListOptions, listObj runtime.Object) error {
	return nil
}

// GuaranteedUpdate keeps calling 'tryUpdate()' to update key 'key' (of type 'destination')
// retrying the update until success if there is index conflict.
// Note that object passed to tryUpdate may change across invocations of tryUpdate() if
// other writers are simultaneously updating it, so tryUpdate() needs to take into account
// the current contents of the object when deciding how the update object should look.
// If the key doesn't exist, it will return NotFound storage error if ignoreNotFound=false
// else `destination` will be set to the zero value of it's type.
// If the eventual successful invocation of `tryUpdate` returns an output with the same serialized
// contents as the input, it won't perform any update, but instead set `destination` to an object with those
// contents.
// If 'cachedExistingObject' is non-nil, it can be used as a suggestion about the
// current version of the object to avoid read operation from storage to get it.
// However, the implementations have to retry in case suggestion is stale.
func (s *Storage) GuaranteedUpdate(
	ctx context.Context,
	key string,
	destination runtime.Object,
	ignoreNotFound bool,
	preconditions *storage.Preconditions,
	tryUpdate storage.UpdateFunc,
	cachedExistingObject runtime.Object,
) error {
	return nil
}

// Count returns number of different entries under the key (generally being path prefix).
func (s *Storage) Count(key string) (int64, error) {
	return 0, nil
}

func (s *Storage) Versioner() storage.Versioner {
	return nil
}
