package sqlstash

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/grafana/grafana/pkg/infra/appcontext"
	"github.com/grafana/grafana/pkg/infra/grn"
	"github.com/grafana/grafana/pkg/infra/log"
	"github.com/grafana/grafana/pkg/infra/slugify"
	"github.com/grafana/grafana/pkg/kinds"
	"github.com/grafana/grafana/pkg/services/featuremgmt"
	"github.com/grafana/grafana/pkg/services/grpcserver"
	"github.com/grafana/grafana/pkg/services/sqlstore/migrator"
	"github.com/grafana/grafana/pkg/services/sqlstore/session"
	"github.com/grafana/grafana/pkg/services/store"
	"github.com/grafana/grafana/pkg/services/store/entity"
	entityDB "github.com/grafana/grafana/pkg/services/store/entity/db"
	"github.com/grafana/grafana/pkg/services/store/entity/migrations"
	"github.com/grafana/grafana/pkg/services/store/kind"
	"github.com/grafana/grafana/pkg/services/store/resolver"
	"github.com/grafana/grafana/pkg/setting"
	"github.com/grafana/grafana/pkg/util"
	"github.com/oklog/ulid/v2"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// Make sure we implement both store + admin
var _ entity.EntityStoreServer = &sqlEntityServer{}
var _ entity.EntityStoreAdminServer = &sqlEntityServer{}

func ProvideSQLEntityServer(db entityDB.EntityDB, cfg *setting.Cfg, grpcServerProvider grpcserver.Provider, kinds kind.KindRegistry, resolver resolver.EntityReferenceResolver, features featuremgmt.FeatureToggles) (entity.EntityStoreServer, error) {
	entityServer := &sqlEntityServer{
		db:       db,
		sess:     db.GetSession(),
		dialect:  migrator.NewDialect(db.GetEngine().DriverName()),
		log:      log.New("sql-entity-server"),
		kinds:    kinds,
		resolver: resolver,
	}

	entity.RegisterEntityStoreServer(grpcServerProvider.GetServer(), entityServer)

	if err := migrations.MigrateEntityStore(db, features); err != nil {
		return nil, err
	}

	return entityServer, nil
}

type sqlEntityServer struct {
	log      log.Logger
	db       entityDB.EntityDB // needed to keep xorm engine in scope
	sess     *session.SessionDB
	dialect  migrator.Dialect
	kinds    kind.KindRegistry
	resolver resolver.EntityReferenceResolver
}

func (s *sqlEntityServer) getReadSelect(r *entity.ReadEntityRequest) string {
	fields := []string{
		"guid",
		"tenant_id", "kind", "uid", "folder", // GRN + folder
		"version", "size", "etag", "errors", // errors are always returned
		"created_at", "created_by",
		"updated_at", "updated_by",
		"origin", "origin_key", "origin_ts"}

	if r.WithBody {
		fields = append(fields, `body`)
	}
	if r.WithMeta {
		fields = append(fields, `meta`)
	}
	if r.WithSummary {
		fields = append(fields, "name", "slug", "description", "labels", "fields")
	}
	quotedFields := make([]string, len(fields))
	for i, f := range fields {
		quotedFields[i] = s.dialect.Quote(f)
	}
	return "SELECT " + strings.Join(quotedFields, ",") +
		" FROM entity" +
		" WHERE "
}

func (s *sqlEntityServer) rowToReadEntityResponse(ctx context.Context, rows *sql.Rows, r *entity.ReadEntityRequest) (*entity.Entity, error) {
	raw := &entity.Entity{
		GRN:    &grn.GRN{},
		Origin: &entity.EntityOriginInfo{},
	}

	summaryjson := &summarySupport{}
	args := []any{
		&raw.Guid,
		&raw.GRN.TenantID, &raw.GRN.ResourceKind, &raw.GRN.ResourceIdentifier, &raw.Folder,
		&raw.Version, &raw.Size, &raw.ETag, &summaryjson.errors,
		&raw.CreatedAt, &raw.CreatedBy,
		&raw.UpdatedAt, &raw.UpdatedBy,
		&raw.Origin.Source, &raw.Origin.Key, &raw.Origin.Time,
	}
	if r.WithBody {
		args = append(args, &raw.Body)
	}
	if r.WithMeta {
		args = append(args, &raw.Meta)
	}
	if r.WithSummary {
		args = append(args, &summaryjson.name, &summaryjson.slug, &summaryjson.description, &summaryjson.labels, &summaryjson.fields)
	}

	err := rows.Scan(args...)
	if err != nil {
		return nil, err
	}

	if raw.Origin.Source == "" {
		raw.Origin = nil
	}

	if r.WithSummary || summaryjson.errors != nil {
		summary, err := summaryjson.toEntitySummary()
		if err != nil {
			return nil, err
		}

		js, err := json.Marshal(summary)
		if err != nil {
			return nil, err
		}
		raw.SummaryJson = js
	}
	return raw, nil
}

func (s *sqlEntityServer) validateGRN(ctx context.Context, grn *grn.GRN) (*grn.GRN, error) {
	if grn == nil {
		return nil, fmt.Errorf("missing GRN")
	}
	user, err := appcontext.User(ctx)
	if err != nil {
		return nil, err
	}
	if grn.TenantID == 0 {
		grn.TenantID = user.OrgID
	} else if grn.TenantID != user.OrgID {
		return nil, fmt.Errorf("tenant ID does not match userID")
	}

	if grn.ResourceKind == "" {
		return nil, fmt.Errorf("GRN missing ResourceKind")
	}
	if grn.ResourceIdentifier == "" {
		return nil, fmt.Errorf("GRN missing ResourceIdentifier")
	}
	if len(grn.ResourceIdentifier) > 64 {
		return nil, fmt.Errorf("GRN ResourceIdentifier is too long (>64)")
	}
	if strings.ContainsAny(grn.ResourceIdentifier, "/#$@?") {
		return nil, fmt.Errorf("invalid character in GRN")
	}
	return grn, nil
}

func (s *sqlEntityServer) Read(ctx context.Context, r *entity.ReadEntityRequest) (*entity.Entity, error) {
	if r.Version != "" {
		return s.readFromHistory(ctx, r)
	}
	grn, err := s.validateGRN(ctx, r.GRN)
	if err != nil {
		return nil, err
	}

	where := " (tenant_id=? AND kind=? AND uid=?)"
	args := []interface{}{grn.TenantID, grn.ResourceKind, grn.ResourceIdentifier}

	rows, err := s.sess.Query(ctx, s.getReadSelect(r)+where, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	if !rows.Next() {
		return &entity.Entity{}, nil
	}

	return s.rowToReadEntityResponse(ctx, rows, r)
}

func (s *sqlEntityServer) readFromHistory(ctx context.Context, r *entity.ReadEntityRequest) (*entity.Entity, error) {
	grn, err := s.validateGRN(ctx, r.GRN)
	if err != nil {
		return nil, err
	}

	fields := []string{
		"guid", "body", "size", "etag", "version",
		"created_at", "created_by", "updated_at", "updated_by",
	}

	query := "SELECT " + strings.Join(fields, ",") +
		" FROM entity_history" +
		" WHERE (tenant_id=? AND kind=? AND uid=? AND version=?)"
	args := []interface{}{grn.TenantID, grn.ResourceKind, grn.ResourceIdentifier, r.Version}

	rows, err := s.sess.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	// Version or key not found
	if !rows.Next() {
		return &entity.Entity{}, nil
	}

	raw := &entity.Entity{
		GRN: r.GRN,
	}
	err = rows.Scan(
		&raw.Guid, &raw.Body, &raw.Size, &raw.ETag, &raw.Version,
		&raw.CreatedAt, &raw.CreatedBy, &raw.UpdatedAt, &raw.UpdatedBy,
	)
	if err != nil {
		return nil, err
	}

	// Dynamically create the summary
	if r.WithSummary {
		builder := s.kinds.GetSummaryBuilder(r.GRN.ResourceKind)
		if builder != nil {
			val, out, err := builder(ctx, r.GRN.ResourceIdentifier, raw.Body)
			if err == nil {
				raw.Body = out // cleaned up
				raw.SummaryJson, err = json.Marshal(val)
				if err != nil {
					return nil, err
				}
			}
		}
	}

	// Clear the body if not requested
	if !r.WithBody {
		raw.Body = nil
	}

	return raw, err
}

func (s *sqlEntityServer) BatchRead(ctx context.Context, b *entity.BatchReadEntityRequest) (*entity.BatchReadEntityResponse, error) {
	if len(b.Batch) < 1 {
		return nil, fmt.Errorf("missing querires")
	}

	first := b.Batch[0]
	args := []any{}
	constraints := []string{}

	for _, r := range b.Batch {
		if r.WithBody != first.WithBody || r.WithSummary != first.WithSummary {
			return nil, fmt.Errorf("requests must want the same things")
		}

		grn, err := s.validateGRN(ctx, r.GRN)
		if err != nil {
			return nil, err
		}

		constraints = append(constraints, "(tenant_id=? AND kind=? AND uid=?)")
		args = append(args, grn.TenantID, grn.ResourceKind, grn.ResourceIdentifier)
		if r.Version != "" {
			return nil, fmt.Errorf("version not supported for batch read (yet?)")
		}
	}

	req := b.Batch[0]
	query := s.getReadSelect(req) + "(" + strings.Join(constraints, " OR ") + ")"
	rows, err := s.sess.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	// TODO? make sure the results are in order?
	rsp := &entity.BatchReadEntityResponse{}
	for rows.Next() {
		r, err := s.rowToReadEntityResponse(ctx, rows, req)
		if err != nil {
			return nil, err
		}
		rsp.Results = append(rsp.Results, r)
	}
	return rsp, nil
}

func (s *sqlEntityServer) Write(ctx context.Context, r *entity.WriteEntityRequest) (*entity.WriteEntityResponse, error) {
	return s.AdminWrite(ctx, entity.ToAdminWriteEntityRequest(r))
}

//nolint:gocyclo
func (s *sqlEntityServer) AdminWrite(ctx context.Context, r *entity.AdminWriteEntityRequest) (*entity.WriteEntityResponse, error) {
	grn, err := s.validateGRN(ctx, r.GRN)
	if err != nil {
		s.log.Error("error validating GRN", "msg", err.Error())
		return nil, err
	}

	timestamp := time.Now().UnixMilli()
	createdAt := r.CreatedAt
	createdBy := r.CreatedBy
	updatedAt := r.UpdatedAt
	updatedBy := r.UpdatedBy
	if updatedBy == "" {
		modifier, err := appcontext.User(ctx)
		if err != nil {
			return nil, err
		}
		if modifier == nil {
			return nil, fmt.Errorf("can not find user in context")
		}
		updatedBy = store.GetUserIDString(modifier)
	}
	if updatedAt < 1000 {
		updatedAt = timestamp
	}

	summary, body, err := s.prepare(ctx, r)
	if err != nil {
		return nil, err
	}

	var access []byte //
	etag := createContentsHash(body, r.Meta, r.Status)
	rsp := &entity.WriteEntityResponse{
		GRN:    grn,
		Status: entity.WriteEntityResponse_CREATED, // Will be changed if not true
	}
	origin := r.Origin
	if origin == nil {
		origin = &entity.EntityOriginInfo{}
	}

	err = s.sess.WithTransaction(ctx, func(tx *session.SessionTx) error {
		isUpdate := false

		versionInfo, err := s.selectForUpdate(ctx, tx, grn)
		if err != nil {
			if err.Error() != "not found" {
				s.log.Error("error getting current version", "msg", err.Error())
				return err
			}
			versionInfo = &entity.EntityVersionInfo{}
			rsp.Status = entity.WriteEntityResponse_CREATED
		} else if r.ClearHistory {
			// Optionally keep the original creation time information
			if createdAt < 1000 || createdBy == "" {
				createdAt = versionInfo.CreatedAt
				createdBy = versionInfo.CreatedBy
			}

			_, err = doDelete(ctx, tx, &entity.Entity{Guid: versionInfo.Guid, GRN: grn})
			if err != nil {
				s.log.Error("error removing old version", "msg", err.Error())
				return err
			}
			versionInfo = &entity.EntityVersionInfo{}
		} else {
			// Same entity
			if versionInfo.ETag == etag {
				rsp.Entity = versionInfo
				rsp.Status = entity.WriteEntityResponse_UNCHANGED
				return nil
			}

			isUpdate = true

			rsp.GUID = versionInfo.Guid
			rsp.Status = entity.WriteEntityResponse_UPDATED

			// Clear the labels+refs
			if _, err := tx.Exec(ctx, "DELETE FROM entity_labels WHERE guid=?", rsp.GUID); err != nil {
				return err
			}
			if _, err := tx.Exec(ctx, "DELETE FROM entity_ref WHERE guid=?", rsp.GUID); err != nil {
				return err
			}
		}

		// Optimistic locking
		if r.PreviousVersion != "" {
			if r.PreviousVersion != versionInfo.Version {
				return fmt.Errorf("optimistic lock failed")
			}
		}

		// Set the comment on this write
		versionInfo.Comment = r.Comment
		versionInfo.Version = ulid.Make().String()

		// 1. Add the `entity_history` values
		versionInfo.Size = int64(len(body))
		versionInfo.ETag = etag
		versionInfo.UpdatedAt = updatedAt
		versionInfo.UpdatedBy = updatedBy

		query, args, err := s.dialect.InsertQuery("entity_history", map[string]interface{}{
			"guid":       versionInfo.Guid,
			"version":    versionInfo.Version,
			"message":    versionInfo.Comment,
			"size":       versionInfo.Size,
			"body":       body,
			"etag":       versionInfo.ETag,
			"folder":     r.Folder,
			"access":     access,
			"updated_at": versionInfo.UpdatedAt,
			"updated_by": versionInfo.UpdatedBy,
		})
		if err != nil {
			s.log.Error("error building entity history insert", "msg", err.Error())
			return err
		}

		_, err = tx.Exec(ctx, query, args...)
		if err != nil {
			s.log.Error("error writing entity history", "msg", err.Error())
			return err
		}

		meta := &kinds.GrafanaResourceMetadata{}
		if len(r.Meta) > 0 {
			err = json.Unmarshal(r.Meta, meta)
			if err != nil {
				return err
			}
		}
		meta.Name = grn.ResourceIdentifier
		if meta.Namespace == "" {
			meta.Namespace = "default" // USE tenant id
		}

		if meta.UID == "" {
			meta.UID = types.UID(uuid.New().String())
		}
		if meta.Annotations == nil {
			meta.Annotations = make(map[string]string)
		}

		meta.ResourceVersion = versionInfo.Version
		meta.Namespace = util.OrgIdToNamespace(grn.TenantID)

		meta.SetFolder(r.Folder)

		if !isUpdate {
			if createdAt < 1000 {
				createdAt = updatedAt
			}
			if createdBy == "" {
				createdBy = updatedBy
			}
		}
		if createdAt > 0 {
			meta.CreationTimestamp = v1.NewTime(time.UnixMilli(createdAt))
		}
		if updatedAt > 0 {
			meta.SetUpdatedTimestamp(util.Pointer(time.UnixMilli(updatedAt)))
		}

		if origin != nil {
			var ts *time.Time
			if origin.Time > 0 {
				ts = util.Pointer(time.UnixMilli(origin.Time))
			}
			meta.SetOriginInfo(&kinds.ResourceOriginInfo{
				Name:      origin.Source,
				Key:       origin.Key,
				Timestamp: ts,
			})
		}

		if len(meta.Labels) > 0 {
			if summary.model.Labels == nil {
				summary.model.Labels = make(map[string]string)
			}
			for k, v := range meta.Labels {
				summary.model.Labels[k] = v
			}
		}
		r.Meta, err = json.Marshal(meta)
		if err != nil {
			return err
		}
		rsp.GUID = string(meta.UID)

		// 5. Add/update the main `entity` table
		rsp.Entity = versionInfo
		if isUpdate {
			query, args, err := s.dialect.UpdateQuery(
				"entity",
				map[string]interface{}{
					"folder":      r.Folder,
					"slug":        summary.model.Slug,
					"updated_at":  updatedAt,
					"updated_by":  updatedBy,
					"body":        body,
					"meta":        r.Meta,
					"status":      r.Status,
					"size":        versionInfo.Size,
					"etag":        versionInfo.ETag,
					"version":     versionInfo.Version,
					"name":        summary.model.Name,
					"description": summary.model.Description,
					"labels":      summary.labels,
					"fields":      summary.fields,
					"errors":      summary.errors,
					"origin":      origin.Source,
					"origin_key":  origin.Key,
					"origin_ts":   timestamp,
				},
				map[string]interface{}{
					"guid": versionInfo.Guid,
				},
			)
			if err != nil {
				s.log.Error("error building entity update sql", "msg", err.Error())
				return err
			}
			_, err = tx.Exec(ctx, query, args...)
			if err != nil {
				s.log.Error("error updating entity", "msg", err.Error())
				return err
			}
			rsp.Status = entity.WriteEntityResponse_UPDATED
		} else {
			query, args, err := s.dialect.InsertQuery(
				"entity",
				map[string]interface{}{
					// below are only set at creation
					"guid":       versionInfo.Guid,
					"tenant_id":  grn.TenantID,
					"kind":       grn.ResourceKind,
					"uid":        grn.ResourceIdentifier,
					"created_at": createdAt,
					"created_by": createdBy,
					// below are the same as update
					"folder":      r.Folder,
					"slug":        summary.model.Slug,
					"updated_at":  updatedAt,
					"updated_by":  updatedBy,
					"body":        body,
					"meta":        r.Meta,
					"status":      r.Status,
					"size":        versionInfo.Size,
					"etag":        versionInfo.ETag,
					"version":     versionInfo.Version,
					"name":        summary.model.Name,
					"description": summary.model.Description,
					"labels":      summary.labels,
					"fields":      summary.fields,
					"errors":      summary.errors,
					"origin":      origin.Source,
					"origin_key":  origin.Key,
					"origin_ts":   timestamp,
				},
			)
			if err != nil {
				s.log.Error("error building entity insert sql", "msg", err.Error())
				return err
			}

			_, err = tx.Exec(ctx, query, args...)
			if err != nil {
				s.log.Error("error inserting entity", "msg", err.Error())
				return err
			}
			rsp.Status = entity.WriteEntityResponse_CREATED
		}

		switch r.GRN.ResourceKind {
		case entity.StandardKindFolder:
			err = updateFolderTree(ctx, tx, r.GRN.TenantID)
			if err != nil {
				s.log.Error("error updating folder tree", "msg", err.Error())
				return err
			}
		}

		summary.folder = r.Folder
		summary.parent_grn = grn

		return s.writeSearchInfo(ctx, tx, grn.String(), summary)
	})
	if err != nil {
		s.log.Error("error writing entity", "msg", err.Error())
		rsp.Status = entity.WriteEntityResponse_ERROR
	}

	rsp.Body = body           // k8s
	rsp.MetaJson = r.Meta     // k8s
	rsp.StatusJson = r.Status // k8s
	rsp.SummaryJson = summary.marshaled

	return rsp, err
}

func (s *sqlEntityServer) selectForUpdate(ctx context.Context, tx *session.SessionTx, grn *grn.GRN) (*entity.EntityVersionInfo, error) {
	q := "SELECT guid,etag,version," +
		"created_at,created_by,updated_at,updated_by," +
		"size" +
		" FROM entity" +
		" WHERE (tenant_id=? AND kind=? AND uid=?)"
	if false { // TODO, MYSQL/PosgreSQL can lock the row " FOR UPDATE"
		q += " FOR UPDATE"
	}
	args := []interface{}{grn.TenantID, grn.ResourceKind, grn.ResourceIdentifier}

	rows, err := tx.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	current := &entity.EntityVersionInfo{}
	if rows.Next() {
		err = rows.Scan(
			&current.Guid, &current.ETag, &current.Version,
			&current.CreatedAt, &current.CreatedBy, &current.UpdatedAt, &current.UpdatedBy,
			&current.Size,
		)
	} else {
		err = fmt.Errorf("not found")
	}

	if err != nil {
		return nil, err
	}

	return current, nil
}

func (s *sqlEntityServer) writeSearchInfo(
	ctx context.Context,
	tx *session.SessionTx,
	grn2 string,
	summary *summarySupport,
) error {
	parent_grn := summary.getParentGRN()

	// Add the labels rows
	for k, v := range summary.model.Labels {
		query, args, err := s.dialect.InsertQuery(
			"entity_labels",
			map[string]interface{}{
				"grn":        grn2,
				"label":      k,
				"value":      v,
				"parent_grn": parent_grn,
			},
		)
		if err != nil {
			return err
		}

		_, err = tx.Exec(ctx, query, args...)
		if err != nil {
			return err
		}
	}

	// Resolve references
	for _, ref := range summary.model.References {
		resolved, err := s.resolver.Resolve(ctx, ref)
		if err != nil {
			return err
		}
		query, args, err := s.dialect.InsertQuery(
			"entity_ref",
			map[string]interface{}{
				"grn":              grn2,
				"parent_grn":       parent_grn,
				"family":           ref.Family,
				"type":             ref.Type,
				"id":               ref.Identifier,
				"resolved_ok":      resolved.OK,
				"resolved_to":      resolved.Key,
				"resolved_warning": resolved.Warning,
				"resolved_time":    resolved.Timestamp,
			},
		)
		if err != nil {
			return err
		}

		_, err = tx.Exec(ctx, query, args...)
		if err != nil {
			return err
		}
	}

	// Traverse entities and insert refs
	if summary.model.Nested != nil {
		for _, childModel := range summary.model.Nested {
			grn2 = (&grn.GRN{
				TenantID:           summary.parent_grn.TenantID,
				ResourceKind:       childModel.Kind,
				ResourceIdentifier: childModel.UID, // append???
			}).ToGRNString()

			child, err := newSummarySupport(childModel)
			if err != nil {
				return err
			}
			child.isNested = true
			child.folder = summary.folder
			child.parent_grn = summary.parent_grn
			parent_grn := child.getParentGRN()

			query, args, err := s.dialect.InsertQuery(
				"entity_nested",
				map[string]interface{}{
					"parent_grn":  parent_grn,
					"grn":         grn2,
					"tenant_id":   summary.parent_grn.TenantID,
					"kind":        childModel.Kind,
					"uid":         childModel.UID,
					"folder":      summary.folder,
					"name":        child.name,
					"description": child.description,
					"labels":      child.labels,
					"fields":      child.fields,
					"errors":      child.errors,
				},
			)
			if err != nil {
				return err
			}

			_, err = tx.Exec(ctx, query, args...)
			if err != nil {
				return err
			}

			err = s.writeSearchInfo(ctx, tx, grn2, child)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (s *sqlEntityServer) prepare(ctx context.Context, r *entity.AdminWriteEntityRequest) (*summarySupport, []byte, error) {
	builder := s.kinds.GetSummaryBuilder(r.GRN.ResourceKind)
	if builder == nil {
		return nil, nil, fmt.Errorf("unsupported kind")
	}

	summary, body, err := builder(ctx, r.GRN.ResourceIdentifier, r.Body)
	if err != nil {
		return nil, nil, err
	}

	// Update a summary based on the name (unless the root suggested one)
	if summary.Slug == "" {
		t := summary.Name
		if t == "" {
			t = r.GRN.ResourceIdentifier
		}
		summary.Slug = slugify.Slugify(t)
	}

	summaryjson, err := newSummarySupport(summary)
	if err != nil {
		return nil, nil, err
	}

	return summaryjson, body, nil
}

func (s *sqlEntityServer) Delete(ctx context.Context, r *entity.DeleteEntityRequest) (*entity.DeleteEntityResponse, error) {
	rsp := &entity.DeleteEntityResponse{}

	err := s.sess.WithTransaction(ctx, func(tx *session.SessionTx) error {
		entity, err := s.Read(ctx, &entity.ReadEntityRequest{
			GRN: r.GRN,
		})
		if err != nil {
			return err
		}

		rsp.OK, err = doDelete(ctx, tx, entity)
		return err
	})

	return rsp, err
}

func doDelete(ctx context.Context, tx *session.SessionTx, ent *entity.Entity) (bool, error) {
	_, err := tx.Exec(ctx, "DELETE FROM entity WHERE guid=?", ent.Guid)
	if err != nil {
		return false, err
	}

	// TODO: keep history? would need current version bump, and the "write" would have to get from history
	_, err = tx.Exec(ctx, "DELETE FROM entity_history WHERE guid=?", ent.Guid)
	if err != nil {
		return false, err
	}
	_, err = tx.Exec(ctx, "DELETE FROM entity_labels WHERE guid=?", ent.Guid)
	if err != nil {
		return false, err
	}
	_, err = tx.Exec(ctx, "DELETE FROM entity_ref WHERE guid=?", ent.Guid)
	if err != nil {
		return false, err
	}

	if ent.GRN.ResourceKind == entity.StandardKindFolder {
		err = updateFolderTree(ctx, tx, ent.GRN.TenantID)
	}

	return true, err
}

func (s *sqlEntityServer) History(ctx context.Context, r *entity.EntityHistoryRequest) (*entity.EntityHistoryResponse, error) {
	grn2, err := s.validateGRN(ctx, r.GRN)
	if err != nil {
		return nil, err
	}

	page := ""
	args := []interface{}{grn2.TenantID, grn2.ResourceKind, grn2.ResourceIdentifier}
	if r.NextPageToken != "" {
		// args = append(args, r.NextPageToken) // TODO, need to get time from the version
		// page = " AND updated <= ?"
		return nil, fmt.Errorf("next page not supported yet")
	}

	query := "SELECT version,size,etag,updated_at,updated_by,message" +
		" FROM entity_history" +
		" WHERE (tenant_id=? AND kind=? AND uid=?)" +
		page +
		" ORDER BY updated_at DESC" +
		" LIMIT 100"

	rows, err := s.sess.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	rsp := &entity.EntityHistoryResponse{
		GRN: r.GRN,
	}
	for rows.Next() {
		v := &entity.EntityVersionInfo{}
		err := rows.Scan(&v.Version, &v.Size, &v.ETag, &v.UpdatedAt, &v.UpdatedBy, &v.Comment)
		if err != nil {
			return nil, err
		}
		rsp.Versions = append(rsp.Versions, v)
	}
	return rsp, err
}

func (s *sqlEntityServer) Search(ctx context.Context, r *entity.EntitySearchRequest) (*entity.EntitySearchResponse, error) {
	user, err := appcontext.User(ctx)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, fmt.Errorf("missing user in context")
	}

	if r.NextPageToken != "" || len(r.Sort) > 0 {
		return nil, fmt.Errorf("not yet supported")
	}

	fields := []string{
		"guid", "guid",
		"tenant_id", "kind", "uid",
		"version", "folder", "slug", "errors", // errors are always returned
		"size", "updated_at", "updated_by",
		"name", "description", // basic summary
	}

	if r.WithBody {
		fields = append(fields, "body", "meta", "status")
	}

	if r.WithLabels {
		fields = append(fields, "labels")
	}
	if r.WithFields {
		fields = append(fields, "fields")
	}

	entityQuery := selectQuery{
		dialect:  migrator.NewDialect(s.sess.DriverName()),
		fields:   fields,
		from:     "entity", // the table
		args:     []any{},
		limit:    r.Limit,
		oneExtra: true, // request one more than the limit (and show next token if it exists)
	}
	entityQuery.addWhere("tenant_id", user.OrgID)

	if len(r.Kind) > 0 {
		entityQuery.addWhereIn("kind", r.Kind)
	}

	// Folder guid
	if r.Folder != "" {
		entityQuery.addWhere("folder", r.Folder)
	}

	if r.NextPageToken != "" {
		entityQuery.addWhere("guid>?", r.NextPageToken)
	}

	if len(r.Labels) > 0 {
		var args []any
		var conditions []string
		for labelKey, labelValue := range r.Labels {
			args = append(args, labelKey)
			args = append(args, labelValue)
			conditions = append(conditions, "(label = ? AND value = ?)")
		}
		query := "SELECT guid FROM entity_labels" +
			" WHERE (" + strings.Join(conditions, " OR ") + ")" +
			" GROUP BY guid" +
			" HAVING COUNT(label) = ?"
		args = append(args, len(r.Labels))

		entityQuery.addWhereInSubquery("guid", query, args)
	}

	query, args := entityQuery.toQuery()

	rows, err := s.sess.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	token := ""
	rsp := &entity.EntitySearchResponse{}
	for rows.Next() {
		result := &entity.EntitySearchResult{
			GRN: &grn.GRN{},
		}
		summaryjson := summarySupport{}

		args := []any{
			&token, &result.Guid,
			&result.GRN.TenantID, &result.GRN.ResourceKind, &result.GRN.ResourceIdentifier,
			&result.Version, &result.Folder, &result.Slug, &summaryjson.errors,
			&result.Size, &result.UpdatedAt, &result.UpdatedBy,
			&result.Name, &summaryjson.description,
		}
		if r.WithBody {
			args = append(args, &result.Body, &result.Meta, &result.Status)
		}
		if r.WithLabels {
			args = append(args, &summaryjson.labels)
		}
		if r.WithFields {
			args = append(args, &summaryjson.fields)
		}

		err = rows.Scan(args...)
		if err != nil {
			return rsp, err
		}

		// found more than requested
		if int64(len(rsp.Results)) >= entityQuery.limit {
			// TODO? this only works if we sort by guid
			rsp.NextPageToken = token
			break
		}

		if summaryjson.description != nil {
			result.Description = *summaryjson.description
		}

		if summaryjson.labels != nil {
			b := []byte(*summaryjson.labels)
			err = json.Unmarshal(b, &result.Labels)
			if err != nil {
				return rsp, err
			}
		}

		if summaryjson.fields != nil {
			result.FieldsJson = []byte(*summaryjson.fields)
		}

		if summaryjson.errors != nil {
			result.ErrorJson = []byte(*summaryjson.errors)
		}

		rsp.Results = append(rsp.Results, result)
	}

	return rsp, err
}

func (s *sqlEntityServer) Watch(*entity.EntityWatchRequest, entity.EntityStore_WatchServer) error {
	return fmt.Errorf("unimplemented")
}

func (s *sqlEntityServer) FindReferences(ctx context.Context, r *entity.ReferenceRequest) (*entity.EntitySearchResponse, error) {
	user, err := appcontext.User(ctx)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, fmt.Errorf("missing user in context")
	}

	if r.NextPageToken != "" {
		return nil, fmt.Errorf("not yet supported")
	}

	fields := []string{
		"guid", "guid", "tenant_id", "kind", "uid",
		"version", "folder", "slug", "errors", // errors are always returned
		"size", "updated_at", "updated_by",
		"name", "description", "meta",
	}

	// SELECT entity_ref.* FROM entity_ref
	// 	JOIN entity ON entity_ref.grn = entity.grn
	// 	WHERE family='librarypanel' AND resolved_to='a7975b7a-fb53-4ab7-951d-15810953b54f';

	sql := strings.Builder{}
	_, _ = sql.WriteString("SELECT ")
	for i, f := range fields {
		if i > 0 {
			_, _ = sql.WriteString(",")
		}
		_, _ = sql.WriteString(fmt.Sprintf("entity.%s", f))
	}
	_, _ = sql.WriteString(" FROM entity_ref JOIN entity ON entity_ref.grn = entity.grn")
	_, _ = sql.WriteString(" WHERE family=? AND resolved_to=?") // TODO tenant ID!!!!

	rows, err := s.sess.Query(ctx, sql.String(), r.Kind, r.Uid)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	token := ""
	rsp := &entity.EntitySearchResponse{}
	for rows.Next() {
		result := &entity.EntitySearchResult{
			GRN: &grn.GRN{},
		}
		summaryjson := summarySupport{}

		args := []interface{}{
			&token, &result.Guid, &result.GRN.TenantID, &result.GRN.ResourceKind, &result.GRN.ResourceIdentifier,
			&result.Version, &result.Folder, &result.Slug, &summaryjson.errors,
			&result.Size, &result.UpdatedAt, &result.UpdatedBy,
			&result.Name, &summaryjson.description, &result.Meta,
		}

		err = rows.Scan(args...)
		if err != nil {
			return rsp, err
		}

		// // found one more than requested
		// if int64(len(rsp.Results)) >= entityQuery.limit {
		// 	// TODO? should this encode start+offset?
		// 	rsp.NextPageToken = token
		// 	break
		// }

		if summaryjson.description != nil {
			result.Description = *summaryjson.description
		}

		if summaryjson.labels != nil {
			b := []byte(*summaryjson.labels)
			err = json.Unmarshal(b, &result.Labels)
			if err != nil {
				return rsp, err
			}
		}

		if summaryjson.fields != nil {
			result.FieldsJson = []byte(*summaryjson.fields)
		}

		if summaryjson.errors != nil {
			result.ErrorJson = []byte(*summaryjson.errors)
		}

		rsp.Results = append(rsp.Results, result)
	}

	return rsp, err
}
