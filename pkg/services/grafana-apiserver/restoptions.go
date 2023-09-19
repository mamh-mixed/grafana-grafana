// SPDX-License-Identifier: AGPL-3.0-only

package grafanaapiserver

import (
	"encoding/json"
	"path"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apiserver/pkg/registry/generic"
	"k8s.io/apiserver/pkg/storage"
	"k8s.io/apiserver/pkg/storage/storagebackend"
	"k8s.io/apiserver/pkg/storage/storagebackend/factory"
	flowcontrolrequest "k8s.io/apiserver/pkg/util/flowcontrol/request"
	"k8s.io/client-go/tools/cache"

	"github.com/grafana/grafana/pkg/services/featuremgmt"
	entityDB "github.com/grafana/grafana/pkg/services/store/entity/db"
	"github.com/grafana/grafana/pkg/services/store/entity/sqlstash"
	"github.com/grafana/grafana/pkg/setting"
)

var _ generic.RESTOptionsGetter = (*RESTOptionsGetter)(nil)

type RESTOptionsGetter struct {
	cfg      *setting.Cfg
	features featuremgmt.FeatureToggles
	Codec    runtime.Codec
}

func NewRESTOptionsGetter(cfg *setting.Cfg, features featuremgmt.FeatureToggles, codec runtime.Codec) *RESTOptionsGetter {
	return &RESTOptionsGetter{
		cfg:      cfg,
		features: features,
		Codec:    codec,
	}
}

func (f *RESTOptionsGetter) GetRESTOptions(resource schema.GroupResource) (generic.RESTOptions, error) {
	connectionInfo, err := json.Marshal(f.cfg.SectionWithEnvOverrides("entity_api").KeysHash())
	if err != nil {
		return generic.RESTOptions{}, err
	}

	storageConfig := &storagebackend.ConfigForResource{
		Config: storagebackend.Config{
			Type:   "custom",
			Prefix: "",
			Transport: storagebackend.TransportConfig{
				ServerList: []string{
					string(connectionInfo),
				},
			},
			Paging:                    false,
			Codec:                     f.Codec,
			EncodeVersioner:           nil,
			Transformer:               nil,
			CompactionInterval:        0,
			CountMetricPollPeriod:     0,
			DBMetricPollInterval:      0,
			HealthcheckTimeout:        0,
			ReadycheckTimeout:         0,
			StorageObjectCountTracker: nil,
		},
		GroupResource: resource,
	}

	ret := generic.RESTOptions{
		StorageConfig: storageConfig,
		Decorator: func(
			config *storagebackend.ConfigForResource,
			resourcePrefix string,
			keyFunc func(obj runtime.Object) (string, error),
			newFunc func() runtime.Object,
			newListFunc func() runtime.Object,
			getAttrsFunc storage.AttrFunc,
			trigger storage.IndexerFuncs,
			indexers *cache.Indexers,
		) (storage.Interface, factory.DestroyFunc, error) {
			eDB, err := entityDB.ProvideEntityDB(nil, f.cfg, f.features)
			if err != nil {
				return nil, nil, err
			}

			store, err := sqlstash.ProvideSQLEntityServer(eDB)
			if err != nil {
				return nil, nil, err
			}

			return NewStorage(resource, store, f.Codec)
		},
		DeleteCollectionWorkers:   0,
		EnableGarbageCollection:   false,
		ResourcePrefix:            path.Join(storageConfig.Prefix, resource.Group, resource.Resource),
		CountMetricPollPeriod:     1 * time.Second,
		StorageObjectCountTracker: flowcontrolrequest.NewStorageObjectCountTracker(),
	}

	return ret, nil
}
