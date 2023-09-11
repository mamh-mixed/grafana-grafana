package sqlstash

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/grafana/grafana/pkg/infra/appcontext"
	"github.com/grafana/grafana/pkg/infra/grn"
	"github.com/grafana/grafana/pkg/infra/log"
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
	"github.com/oklog/ulid/v2"
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
	return "SELECT " + strings.Join(quotedFields, ",")
}

func (s *sqlEntityServer) rowToReadEntityResponse(ctx context.Context, rows *sql.Rows, r *entity.ReadEntityRequest) (*entity.Entity, error) {
	raw := &entity.Entity{
		GRN:    &grn.GRN{},
		Origin: &entity.EntityOriginInfo{},
	}

	args := []any{
		&raw.Guid,
		&raw.GRN.TenantID, &raw.GRN.ResourceKind, &raw.GRN.ResourceIdentifier, &raw.Folder,
		&raw.Version, &raw.Size, &raw.ETag, &raw.Errors,
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
		args = append(args, &raw.Name, &raw.Slug, &raw.Description, &raw.Labels, &raw.Fields)
	}

	err := rows.Scan(args...)
	if err != nil {
		return nil, err
	}

	if raw.Origin.Source == "" {
		raw.Origin = nil
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
	return s.read(ctx, s.sess, r)
}

func (s *sqlEntityServer) read(ctx context.Context, tx session.SessionQuerier, r *entity.ReadEntityRequest) (*entity.Entity, error) {
	grn, err := s.validateGRN(ctx, r.GRN)
	if err != nil {
		return nil, err
	}

	table := "entity"
	where := " (tenant_id=? AND kind=? AND uid=?)"
	args := []any{grn.TenantID, grn.ResourceKind, grn.ResourceIdentifier}

	if r.Version != "" {
		table = "entity_history"
		where += " AND version=?"
		args = append(args, r.Version)
	}

	query := s.getReadSelect(r)

	if false { // TODO, MYSQL/PosgreSQL can lock the row " FOR UPDATE"
		query += " FOR UPDATE"
	}

	query += " FROM " + table +
		" WHERE " + where

	rows, err := tx.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	if !rows.Next() {
		return &entity.Entity{}, nil
	}

	return s.rowToReadEntityResponse(ctx, rows, r)
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
	query := s.getReadSelect(req) +
		" FROM entity" +
		" WHERE (" + strings.Join(constraints, " OR ") + ")"
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
	grn, err := s.validateGRN(ctx, r.Entity.GRN)
	if err != nil {
		s.log.Error("error validating GRN", "msg", err.Error())
		return nil, err
	}

	timestamp := time.Now().UnixMilli()
	createdAt := r.Entity.CreatedAt
	createdBy := r.Entity.CreatedBy
	updatedAt := r.Entity.UpdatedAt
	updatedBy := r.Entity.UpdatedBy
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

	rsp := &entity.WriteEntityResponse{
		Status: entity.WriteEntityResponse_CREATED, // Will be changed if not true
	}
	origin := r.Entity.Origin
	if origin == nil {
		origin = &entity.EntityOriginInfo{}
	}

	err = s.sess.WithTransaction(ctx, func(tx *session.SessionTx) error {
		current, err := s.read(ctx, tx, &entity.ReadEntityRequest{
			GRN:         grn,
			WithMeta:    true,
			WithBody:    false,
			WithStatus:  true,
			WithSummary: true,
		})
		if err != nil {
			return err
		}

		// Optimistic locking
		if r.PreviousVersion != "" {
			if r.PreviousVersion != current.Version {
				return fmt.Errorf("optimistic lock failed")
			}
		}

		isUpdate := false

		// if we found an existing entity
		if current.Guid != "" {
			if r.ClearHistory {
				// Optionally keep the original creation time information
				if createdAt < 1000 || createdBy == "" {
					createdAt = current.CreatedAt
					createdBy = current.CreatedBy
				}

				_, err = doDelete(ctx, tx, &entity.Entity{Guid: current.Guid, GRN: grn})
				if err != nil {
					s.log.Error("error removing old version", "msg", err.Error())
					return err
				}

				current = &entity.Entity{}
			} else {
				isUpdate = true

				rsp.Entity.Guid = current.Guid

				// Clear the labels+refs
				if _, err := tx.Exec(ctx, "DELETE FROM entity_labels WHERE guid=?", rsp.Entity.Guid); err != nil {
					return err
				}
				if _, err := tx.Exec(ctx, "DELETE FROM entity_ref WHERE guid=?", rsp.Entity.Guid); err != nil {
					return err
				}
			}
		} else {
			// generate guid for new entity
			current.Guid = ulid.Make().String()
		}

		// Set the comment on this write
		current.Message = r.Entity.Message
		current.Version = ulid.Make().String()

		if r.Entity.Body != nil {
			current.Body = r.Entity.Body
			current.Size = int64(len(current.Body))
		}

		if r.Entity.Meta != nil {
			current.Meta = r.Entity.Meta
		}

		if r.Entity.Status != nil {
			current.Status = r.Entity.Status
		}

		etag := createContentsHash(current.Body, current.Meta, current.Status)
		current.ETag = etag
		current.UpdatedAt = updatedAt
		current.UpdatedBy = updatedBy

		values := map[string]any{
			// below are only set at creation
			"guid":       current.Guid,
			"tenant_id":  grn.TenantID,
			"kind":       grn.ResourceKind,
			"uid":        grn.ResourceIdentifier,
			"created_at": createdAt,
			"created_by": createdBy,
			// below are set during creation and update
			"folder":      current.Folder,
			"slug":        current.Slug,
			"updated_at":  updatedAt,
			"updated_by":  updatedBy,
			"body":        current.Body,
			"meta":        current.Meta,
			"status":      current.Status,
			"size":        current.Size,
			"etag":        current.ETag,
			"version":     current.Version,
			"name":        current.Name,
			"description": current.Description,
			"labels":      current.Labels,
			"fields":      current.Fields,
			"errors":      current.Errors,
			"origin":      origin.Source,
			"origin_key":  origin.Key,
			"origin_ts":   origin.Time,
		}

		// 1. Add the `entity_history` values
		query, args, err := s.dialect.InsertQuery("entity_history", values)
		if err != nil {
			s.log.Error("error building entity history insert", "msg", err.Error())
			return err
		}

		_, err = tx.Exec(ctx, query, args...)
		if err != nil {
			s.log.Error("error writing entity history", "msg", err.Error())
			return err
		}

		// 5. Add/update the main `entity` table
		rsp.Entity = current
		if isUpdate {
			// remove values that are only set at insert
			delete(values, "guid")
			delete(values, "tenant_id")
			delete(values, "kind")
			delete(values, "uid")
			delete(values, "created_at")
			delete(values, "created_by")

			query, args, err := s.dialect.UpdateQuery(
				"entity",
				values,
				map[string]any{
					"guid": current.Guid,
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
			query, args, err := s.dialect.InsertQuery("entity", values)
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

		switch current.GRN.ResourceKind {
		case entity.StandardKindFolder:
			err = updateFolderTree(ctx, tx, r.Entity.GRN.TenantID)
			if err != nil {
				s.log.Error("error updating folder tree", "msg", err.Error())
				return err
			}
		}

		rsp.Entity = current

		return s.writeSearchInfo(ctx, tx, current)
	})
	if err != nil {
		s.log.Error("error writing entity", "msg", err.Error())
		rsp.Status = entity.WriteEntityResponse_ERROR
	}

	return rsp, err
}

func (s *sqlEntityServer) writeSearchInfo(
	ctx context.Context,
	tx *session.SessionTx,
	current *entity.Entity,
) error {
	// parent_grn := current.getParentGRN()

	// Add the labels rows
	for k, v := range current.Labels {
		query, args, err := s.dialect.InsertQuery(
			"entity_labels",
			map[string]any{
				"grn":   current.GRN.ToGRNString(),
				"label": k,
				"value": v,
				// "parent_grn": parent_grn,
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

	return nil
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

	var limit int64 = 100
	if r.Limit > 0 && r.Limit < 100 {
		limit = r.Limit
	}

	rr := &entity.ReadEntityRequest{
		GRN:         grn2,
		WithMeta:    true,
		WithBody:    false,
		WithStatus:  true,
		WithSummary: true,
	}

	query := s.getReadSelect(rr) +
		" FROM entity_history" +
		" WHERE (tenant_id=? AND kind=? AND uid=?)"
	args := []any{
		grn2.TenantID, grn2.ResourceKind, grn2.ResourceIdentifier,
	}

	if r.NextPageToken != "" {
		query += " AND version <= ?"
		args = append(args, r.NextPageToken)
	}

	query += " ORDER BY version DESC" +
		// select 1 more than we need to see if there is a next page
		" LIMIT " + fmt.Sprint(limit+1)

	rows, err := s.sess.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	rsp := &entity.EntityHistoryResponse{
		GRN: grn2,
	}
	for rows.Next() {
		v, err := s.rowToReadEntityResponse(ctx, rows, rr)
		if err != nil {
			return nil, err
		}

		// found more than requested
		if int64(len(rsp.Versions)) >= limit {
			rsp.NextPageToken = v.Version
			break
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

	s.log.Info("searching", "query", query, "args", args)

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

		var labels []byte

		args := []any{
			&token, &result.Guid,
			&result.GRN.TenantID, &result.GRN.ResourceKind, &result.GRN.ResourceIdentifier,
			&result.Version, &result.Folder, &result.Slug, &result.ErrorJson,
			&result.Size, &result.UpdatedAt, &result.UpdatedBy,
			&result.Name, &result.Description,
		}
		if r.WithBody {
			args = append(args, &result.Body, &result.Meta, &result.Status)
		}
		if r.WithLabels {
			args = append(args, &labels)
		}
		if r.WithFields {
			args = append(args, &result.FieldsJson)
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

		if labels != nil {
			err = json.Unmarshal(labels, &result.Labels)
			if err != nil {
				return rsp, err
			}
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
		"guid", "guid",
		"tenant_id", "kind", "uid",
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

		args := []any{
			&token, &result.Guid,
			&result.GRN.TenantID, &result.GRN.ResourceKind, &result.GRN.ResourceIdentifier,
			&result.Version, &result.Folder, &result.Slug, &result.ErrorJson,
			&result.Size, &result.UpdatedAt, &result.UpdatedBy,
			&result.Name, &result.Description, &result.Meta,
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

		rsp.Results = append(rsp.Results, result)
	}

	return rsp, err
}
