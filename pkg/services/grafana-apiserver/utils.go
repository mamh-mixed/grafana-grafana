package grafanaapiserver

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apiserver/pkg/endpoints/request"

	"github.com/grafana/grafana/pkg/infra/appcontext"
	"github.com/grafana/grafana/pkg/infra/grn"
	"github.com/grafana/grafana/pkg/services/store/entity"
	"github.com/grafana/grafana/pkg/services/user"
	"github.com/grafana/grafana/pkg/util"
)

type Key struct {
	Group     string
	Kind      string
	Namespace string
	Name      string
}

func ParseKey(key string) (*Key, error) {
	// /<group>/<kind plural lowercase>/<namespace>/<name>
	parts := strings.Split(key, "/")
	if len(parts) != 5 {
		return nil, fmt.Errorf("invalid key (expecting 4 parts) " + key)
	}

	return &Key{
		Group:     parts[1],
		Kind:      parts[2],
		Namespace: parts[3],
		Name:      parts[4],
	}, nil
}

func (k *Key) String() string {
	return fmt.Sprintf("/%s/%s/%s/%s", k.Group, k.Kind, k.Namespace, k.Name)
}

func (k *Key) IsEqual(other *Key) bool {
	return k.Group == other.Group && k.Kind == other.Kind && k.Namespace == other.Namespace && k.Name == other.Name
}

func (k *Key) TenantID() (int64, error) {
	if k.Namespace == "default" {
		return 1, nil
	}
	tid := strings.Split(k.Namespace, "-")
	if len(tid) != 2 || !(tid[0] == "org" || tid[0] == "tenant") {
		return 0, fmt.Errorf("invalid namespace, expected org|tenant-${#}")
	}
	intVar, err := strconv.ParseInt(tid[1], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid namespace, expected number")
	}
	return intVar, nil
}

func (k *Key) ToGRN(kindName string) (*grn.GRN, error) {
	tid, err := k.TenantID()
	if err != nil {
		return nil, err
	}

	return &grn.GRN{
		ResourceGroup:      k.Group,
		ResourceKind:       kindName,
		ResourceIdentifier: k.Name,
		TenantID:           tid,
	}, nil
}

// Convert an etcd key to GRN style
func keyToGRN(key string, kindName string) (*grn.GRN, error) {
	k, err := ParseKey(key)
	if err != nil {
		return nil, err
	}
	return k.ToGRN(kindName)
}

// this is terrible... but just making it work!!!!
func entityToResource(rsp *entity.Entity, res runtime.Object) error {
	var err error

	metaAccessor, err := meta.Accessor(res)
	if err != nil {
		return err
	}

	if rsp.GRN == nil {
		return fmt.Errorf("invalid entity, missing GRN")
	}

	if len(rsp.Meta) > 0 {
		raw := []byte("{\"metadata\":" + string(rsp.Meta) + "}") // HACK
		err = json.Unmarshal(raw, res)
		if err != nil {
			return err
		}
	}

	metaAccessor.SetName(rsp.GRN.ResourceIdentifier)
	if rsp.GRN.TenantID != 1 {
		metaAccessor.SetNamespace(fmt.Sprintf("tenant-%d", rsp.GRN.TenantID))
	} else {
		metaAccessor.SetNamespace("default") // org 1
	}
	// metaAccessor.SetKind(rsp.GRN.ResourceKind)
	metaAccessor.SetUID(types.UID(rsp.Guid))
	metaAccessor.SetResourceVersion(rsp.Version)
	metaAccessor.SetCreationTimestamp(metav1.Unix(rsp.CreatedAt/1000, rsp.CreatedAt%1000*1000000))

	annotations := metaAccessor.GetAnnotations()
	if annotations == nil {
		annotations = map[string]string{}
	}

	if rsp.Folder != "" {
		annotations["grafana.com/folder"] = rsp.Folder
	}
	if rsp.CreatedBy != "" {
		annotations["grafana.com/createdBy"] = rsp.CreatedBy
	}
	if rsp.UpdatedBy != "" {
		annotations["grafana.com/updatedBy"] = rsp.UpdatedBy
	}
	if rsp.UpdatedAt != 0 {
		updatedAt := time.UnixMilli(rsp.UpdatedAt).UTC()
		annotations["grafana.com/updatedTimestamp"] = updatedAt.Format(time.RFC3339)
	}
	annotations["grafana.com/slug"] = rsp.Slug

	if rsp.Origin != nil {
		originTime := time.UnixMilli(rsp.Origin.Time).UTC()
		annotations["grafana.com/originName"] = rsp.Origin.Source
		annotations["grafana.com/originKey"] = rsp.Origin.Key
		annotations["grafana.com/originTime"] = originTime.Format(time.RFC3339)
		annotations["grafana.com/originPath"] = "" // rsp.Origin.Path
	}

	metaAccessor.SetAnnotations(annotations)

	if len(rsp.Labels) > 0 {
		metaAccessor.SetLabels(rsp.Labels)
	}

	// TODO fields?

	if len(rsp.Body) > 0 {
		err = json.Unmarshal(rsp.Body, res)
		if err != nil {
			return err
		}
	}
	if len(rsp.Status) > 0 {
		status := reflect.ValueOf(res).Elem().FieldByName("Status")
		if status != (reflect.Value{}) && status.CanSet() {
			err = json.Unmarshal(rsp.Status, status.Addr().Interface())
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func resourceToEntity(key string, res runtime.Object) (*entity.Entity, error) {
	metaAccessor, err := meta.Accessor(res)
	if err != nil {
		return nil, err
	}

	fmt.Printf("RESOURCE: %+v\n", res)

	g, err := keyToGRN(key, res.GetObjectKind().GroupVersionKind().Kind)
	if err != nil {
		return nil, err
	}

	rsp := &entity.Entity{
		GRN:       g,
		Name:      metaAccessor.GetName(),
		Guid:      string(metaAccessor.GetUID()),
		Version:   metaAccessor.GetResourceVersion(),
		Folder:    metaAccessor.GetAnnotations()["grafana.com/folder"],
		CreatedAt: metaAccessor.GetCreationTimestamp().Time.UnixMilli(),
		CreatedBy: metaAccessor.GetAnnotations()["grafana.com/createdBy"],
		UpdatedBy: metaAccessor.GetAnnotations()["grafana.com/updatedBy"],
		Slug:      metaAccessor.GetAnnotations()["grafana.com/slug"],
		Origin: &entity.EntityOriginInfo{
			Source: metaAccessor.GetAnnotations()["grafana.com/originName"],
			Key:    metaAccessor.GetAnnotations()["grafana.com/originKey"],
			// Path: metaAccessor.GetAnnotations()["grafana.com/originPath"],
		},
		Labels: metaAccessor.GetLabels(),
	}

	if metaAccessor.GetAnnotations()["grafana.com/updatedTimestamp"] != "" {
		t, err := time.Parse(time.RFC3339, metaAccessor.GetAnnotations()["grafana.com/updatedTimestamp"])
		if err != nil {
			return nil, err
		}
		rsp.UpdatedAt = t.UnixMilli()
	}

	if metaAccessor.GetAnnotations()["grafana.com/originTime"] != "" {
		t, err := time.Parse(time.RFC3339, metaAccessor.GetAnnotations()["grafana.com/originTime"])
		if err != nil {
			return nil, err
		}
		rsp.Origin.Time = t.UnixMilli()
	}

	/*
		rsp.Meta, err = json.Marshal(rrr.Object["metadata"])
		if err != nil {
			return nil, err
		}

		rsp.Body, err = json.Marshal(rrr.Object["spec"])
		if err != nil {
			return nil, err
		}
	*/

	status := reflect.ValueOf(res).Elem().FieldByName("Status")
	if status != (reflect.Value{}) {
		rsp.Status, err = json.Marshal(status.Interface())
		if err != nil {
			return nil, err
		}
	}

	fmt.Printf("ENTITY: %+v\n", rsp)
	return rsp, nil
}

func contextWithFakeGrafanaUser(ctx context.Context) (context.Context, error) {
	info, ok := request.UserFrom(ctx)
	if !ok {
		return ctx, fmt.Errorf("could not find k8s user info in context")
	}

	var err error
	user := &user.SignedInUser{
		UserID: -1,
		OrgID:  -1,
		Name:   info.GetName(),
	}
	if user.Name == "system:apiserver" {
		user.OrgID = 1
		user.UserID = 1
	}

	v, ok := info.GetExtra()["user-id"]
	if ok && len(v) > 0 {
		user.UserID, err = strconv.ParseInt(v[0], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("couldn't determine the Grafana user id from extras map")
		}
	}
	v, ok = info.GetExtra()["org-id"]
	if ok && len(v) > 0 {
		user.OrgID, err = strconv.ParseInt(v[0], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("couldn't determine the Grafana org id from extras map")
		}
	}

	if user.OrgID < 0 || user.UserID < 0 {
		// Aggregated mode.... need to map this to a real user somehow
		user.OrgID = 1
		user.UserID = 1
		// return nil, fmt.Errorf("insufficient information on user context, couldn't determine UserID and OrgID")
	}

	// HACK alert... change to the requested org
	// TODO: should validate that user has access to that org/tenant
	ns, ok := request.NamespaceFrom(ctx)
	if ok && ns != "" {
		nsorg, err := util.NamespaceToOrgID(ns)
		if err != nil {
			return nil, err
		}
		user.OrgID = nsorg
	}

	return appcontext.WithUser(ctx, user), nil
}
