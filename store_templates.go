package cmsstore

import (
	"context"
	"errors"
	"log"
	"strconv"
	"strings"

	"github.com/doug-martin/goqu/v9"
	"github.com/dromara/carbon/v2"
	"github.com/gouniverse/base/database"
	"github.com/gouniverse/sb"
	"github.com/samber/lo"
)

func (store *store) TemplateCount(ctx context.Context, options TemplateQueryInterface) (int64, error) {
	options.SetCountOnly(true)

	q, _, err := store.templateSelectQuery(options)

	if err != nil {
		return -1, err
	}

	sqlStr, params, errSql := q.Prepared(true).
		Limit(1).
		Select(goqu.COUNT(goqu.Star()).As("count")).
		ToSQL()

	if errSql != nil {
		return -1, nil
	}

	if store.debugEnabled {
		log.Println(sqlStr)
	}

	mapped, err := database.SelectToMapString(store.toQuerableContext(ctx), sqlStr, params...)

	if err != nil {
		return -1, err
	}

	if len(mapped) < 1 {
		return -1, nil
	}

	countStr := mapped[0]["count"]

	i, err := strconv.ParseInt(countStr, 10, 64)

	if err != nil {
		return -1, err

	}

	return i, nil
}

func (store *store) TemplateCreate(ctx context.Context, template TemplateInterface) error {
	if template == nil {
		return errors.New("template is nil")
	}
	if template.CreatedAt() == "" {
		template.SetCreatedAt(carbon.Now(carbon.UTC).ToDateTimeString(carbon.UTC))
	}

	if template.UpdatedAt() == "" {
		template.SetUpdatedAt(carbon.Now(carbon.UTC).ToDateTimeString(carbon.UTC))
	}

	data := template.Data()

	sqlStr, params, errSql := goqu.Dialect(store.dbDriverName).
		Insert(store.templateTableName).
		Prepared(true).
		Rows(data).
		ToSQL()

	if errSql != nil {
		return errSql
	}

	if store.debugEnabled {
		log.Println(sqlStr)
	}

	if store.db == nil {
		return errors.New("templatestore: database is nil")
	}

	_, err := database.Execute(store.toQuerableContext(ctx), sqlStr, params...)

	if err != nil {
		return err
	}

	template.MarkAsNotDirty()

	return nil
}

func (store *store) TemplateDelete(ctx context.Context, template TemplateInterface) error {
	if template == nil {
		return errors.New("template is nil")
	}

	return store.TemplateDeleteByID(ctx, template.ID())
}

func (store *store) TemplateDeleteByID(ctx context.Context, id string) error {
	if id == "" {
		return errors.New("template id is empty")
	}

	sqlStr, params, errSql := goqu.Dialect(store.dbDriverName).
		Delete(store.templateTableName).
		Prepared(true).
		Where(goqu.C("id").Eq(id)).
		ToSQL()

	if errSql != nil {
		return errSql
	}

	if store.debugEnabled {
		log.Println(sqlStr)
	}

	_, err := database.Execute(store.toQuerableContext(ctx), sqlStr, params...)

	return err
}

func (store *store) TemplateFindByHandle(ctx context.Context, handle string) (template TemplateInterface, err error) {
	if handle == "" {
		return nil, errors.New("template handle is empty")
	}

	list, err := store.TemplateList(ctx, TemplateQuery().
		SetHandle(handle).
		SetLimit(1))

	if err != nil {
		return nil, err
	}

	if len(list) > 0 {
		return list[0], nil
	}

	return nil, nil
}

func (store *store) TemplateFindByID(ctx context.Context, id string) (template TemplateInterface, err error) {
	if id == "" {
		return nil, errors.New("template id is empty")
	}

	list, err := store.TemplateList(ctx, TemplateQuery().SetID(id).SetLimit(1))

	if err != nil {
		return nil, err
	}

	if len(list) > 0 {
		return list[0], nil
	}

	return nil, nil
}

func (store *store) TemplateList(ctx context.Context, query TemplateQueryInterface) ([]TemplateInterface, error) {
	q, columns, err := store.templateSelectQuery(query)

	if err != nil {
		return []TemplateInterface{}, err
	}

	sqlStr, _, errSql := q.Select(columns...).ToSQL()

	if errSql != nil {
		return []TemplateInterface{}, nil
	}

	if store.debugEnabled {
		log.Println(sqlStr)
	}

	if store.db == nil {
		return []TemplateInterface{}, errors.New("templatestore: database is nil")
	}

	modelMaps, err := database.SelectToMapString(store.toQuerableContext(ctx), sqlStr)

	if err != nil {
		return []TemplateInterface{}, err
	}

	list := []TemplateInterface{}

	lo.ForEach(modelMaps, func(modelMap map[string]string, index int) {
		model := NewTemplateFromExistingData(modelMap)
		list = append(list, model)
	})

	return list, nil
}

func (store *store) TemplateSoftDelete(ctx context.Context, template TemplateInterface) error {
	if template == nil {
		return errors.New("template is nil")
	}

	template.SetSoftDeletedAt(carbon.Now(carbon.UTC).ToDateTimeString(carbon.UTC))

	return store.TemplateUpdate(ctx, template)
}

func (store *store) TemplateSoftDeleteByID(ctx context.Context, id string) error {
	template, err := store.TemplateFindByID(ctx, id)

	if err != nil {
		return err
	}

	return store.TemplateSoftDelete(ctx, template)
}

func (store *store) TemplateUpdate(ctx context.Context, template TemplateInterface) error {
	if store.db == nil {
		return errors.New("templatestore: database is nil")
	}

	if template == nil {
		return errors.New("template is nil")
	}

	template.SetUpdatedAt(carbon.Now(carbon.UTC).ToDateTimeString())

	dataChanged := template.DataChanged()

	delete(dataChanged, COLUMN_ID) // ID is not updateable

	if len(dataChanged) < 1 {
		return nil
	}

	sqlStr, params, errSql := goqu.Dialect(store.dbDriverName).
		Update(store.templateTableName).
		Prepared(true).
		Set(dataChanged).
		Where(goqu.C(COLUMN_ID).Eq(template.ID())).
		ToSQL()

	if errSql != nil {
		return errSql
	}

	if store.debugEnabled {
		log.Println(sqlStr)
	}

	_, err := database.Execute(store.toQuerableContext(ctx), sqlStr, params...)

	if err != nil {
		return err
	}

	template.MarkAsNotDirty()

	return nil
}

func (store *store) templateSelectQuery(options TemplateQueryInterface) (selectDataset *goqu.SelectDataset, columns []any, err error) {
	if options == nil {
		return nil, nil, errors.New("template query cannot be nil")
	}

	if err := options.Validate(); err != nil {
		return nil, nil, err
	}

	q := goqu.Dialect(store.dbDriverName).From(store.templateTableName)

	if options.HasCreatedAtGte() && options.HasCreatedAtLte() {
		q = q.Where(
			goqu.C(COLUMN_CREATED_AT).Gte(options.CreatedAtGte()),
			goqu.C(COLUMN_CREATED_AT).Lte(options.CreatedAtLte()),
		)
	} else if options.HasCreatedAtGte() {
		q = q.Where(goqu.C(COLUMN_CREATED_AT).Gte(options.CreatedAtGte()))
	} else if options.HasCreatedAtLte() {
		q = q.Where(goqu.C(COLUMN_CREATED_AT).Lte(options.CreatedAtLte()))
	}

	if options.HasHandle() {
		q = q.Where(goqu.C(COLUMN_HANDLE).Eq(options.Handle()))
	}

	if options.HasID() {
		q = q.Where(goqu.C(COLUMN_ID).Eq(options.ID()))
	}

	if options.HasIDIn() {
		q = q.Where(goqu.C(COLUMN_ID).In(options.IDIn()))
	}

	if options.HasNameLike() {
		q = q.Where(goqu.C(COLUMN_NAME).Like(options.NameLike()))
	}

	if options.HasSiteID() {
		q = q.Where(goqu.C(COLUMN_SITE_ID).Eq(options.SiteID()))
	}

	if options.HasStatus() {
		q = q.Where(goqu.C(COLUMN_STATUS).Eq(options.Status()))
	}

	if options.HasStatusIn() {
		q = q.Where(goqu.C(COLUMN_STATUS).In(options.StatusIn()))
	}

	if !options.IsCountOnly() {
		if options.HasLimit() {
			q = q.Limit(uint(options.Limit()))
		}

		if options.HasOffset() {
			q = q.Offset(uint(options.Offset()))
		}
	}

	sortOrder := sb.DESC
	if options.HasSortOrder() {
		sortOrder = options.SortOrder()
	}

	if options.HasOrderBy() {
		if strings.EqualFold(sortOrder, sb.ASC) {
			q = q.Order(goqu.I(options.OrderBy()).Asc())
		} else {
			q = q.Order(goqu.I(options.OrderBy()).Desc())
		}
	}

	columns = []any{}

	for _, column := range options.Columns() {
		columns = append(columns, column)
	}

	if options.SoftDeletedIncluded() {
		return q, columns, nil // soft deleted templates requested specifically
	}

	softDeleted := goqu.C(COLUMN_SOFT_DELETED_AT).
		Gt(carbon.Now(carbon.UTC).ToDateTimeString())

	return q.Where(softDeleted), columns, nil
}
