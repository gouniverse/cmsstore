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

func (store *store) SiteCount(ctx context.Context, options SiteQueryInterface) (int64, error) {
	options.SetCountOnly(true)

	q, _, err := store.siteSelectQuery(options)

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

func (store *store) SiteCreate(ctx context.Context, site SiteInterface) error {
	if site == nil {
		return errors.New("site is nil")
	}

	site.SetCreatedAt(carbon.Now(carbon.UTC).ToDateTimeString(carbon.UTC))
	site.SetUpdatedAt(carbon.Now(carbon.UTC).ToDateTimeString(carbon.UTC))

	data := site.Data()

	sqlStr, params, errSql := goqu.Dialect(store.dbDriverName).
		Insert(store.siteTableName).
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
		return errors.New("sitestore: database is nil")
	}

	_, err := database.Execute(store.toQuerableContext(ctx), sqlStr, params...)

	if err != nil {
		return err
	}

	site.MarkAsNotDirty()

	return nil
}

func (store *store) SiteDelete(ctx context.Context, site SiteInterface) error {
	if site == nil {
		return errors.New("site is nil")
	}

	return store.SiteDeleteByID(ctx, site.ID())
}

func (store *store) SiteDeleteByID(ctx context.Context, id string) error {
	if id == "" {
		return errors.New("site id is empty")
	}

	sqlStr, params, errSql := goqu.Dialect(store.dbDriverName).
		Delete(store.siteTableName).
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

func (store *store) SiteFindByDomainName(ctx context.Context, domainName string) (site SiteInterface, err error) {
	if domainName == "" {
		return nil, errors.New("site domain is empty")
	}

	list, err := store.SiteList(ctx, SiteQuery().
		SetDomainName(domainName).
		SetLimit(1))

	if err != nil {
		return nil, err
	}

	if len(list) > 0 {
		return list[0], nil
	}

	return nil, nil
}

func (store *store) SiteFindByHandle(ctx context.Context, handle string) (site SiteInterface, err error) {
	if handle == "" {
		return nil, errors.New("site handle is empty")
	}

	list, err := store.SiteList(ctx, SiteQuery().
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

func (store *store) SiteFindByID(ctx context.Context, id string) (site SiteInterface, err error) {
	if id == "" {
		return nil, errors.New("site id is empty")
	}

	list, err := store.SiteList(ctx, SiteQuery().SetID(id).SetLimit(1))

	if err != nil {
		return nil, err
	}

	if len(list) > 0 {
		return list[0], nil
	}

	return nil, nil
}

func (store *store) SiteList(ctx context.Context, query SiteQueryInterface) ([]SiteInterface, error) {
	q, columns, err := store.siteSelectQuery(query)

	if err != nil {
		return []SiteInterface{}, err
	}

	sqlStr, _, errSql := q.Select(columns...).ToSQL()

	if errSql != nil {
		return []SiteInterface{}, nil
	}

	if store.debugEnabled {
		log.Println(sqlStr)
	}

	if store.db == nil {
		return []SiteInterface{}, errors.New("sitestore: database is nil")
	}

	modelMaps, err := database.SelectToMapString(store.toQuerableContext(ctx), sqlStr)

	if err != nil {
		return []SiteInterface{}, err
	}

	list := []SiteInterface{}

	lo.ForEach(modelMaps, func(modelMap map[string]string, index int) {
		model := NewSiteFromExistingData(modelMap)
		list = append(list, model)
	})

	return list, nil
}

func (store *store) SiteSoftDelete(ctx context.Context, site SiteInterface) error {
	if site == nil {
		return errors.New("site is nil")
	}

	site.SetSoftDeletedAt(carbon.Now(carbon.UTC).ToDateTimeString(carbon.UTC))

	return store.SiteUpdate(ctx, site)
}

func (store *store) SiteSoftDeleteByID(ctx context.Context, id string) error {
	site, err := store.SiteFindByID(ctx, id)

	if err != nil {
		return err
	}

	return store.SiteSoftDelete(ctx, site)
}

func (store *store) SiteUpdate(ctx context.Context, site SiteInterface) error {
	if site == nil {
		return errors.New("site is nil")
	}

	site.SetUpdatedAt(carbon.Now(carbon.UTC).ToDateTimeString())

	dataChanged := site.DataChanged()

	delete(dataChanged, COLUMN_ID) // ID is not updateable

	if len(dataChanged) < 1 {
		return nil
	}

	sqlStr, params, errSql := goqu.Dialect(store.dbDriverName).
		Update(store.siteTableName).
		Prepared(true).
		Set(dataChanged).
		Where(goqu.C(COLUMN_ID).Eq(site.ID())).
		ToSQL()

	if errSql != nil {
		return errSql
	}

	if store.debugEnabled {
		log.Println(sqlStr)
	}

	if store.db == nil {
		return errors.New("sitestore: database is nil")
	}

	_, err := database.Execute(store.toQuerableContext(ctx), sqlStr, params...)

	site.MarkAsNotDirty()

	return err
}

func (store *store) siteSelectQuery(options SiteQueryInterface) (selectDataset *goqu.SelectDataset, columns []any, err error) {
	if options == nil {
		return nil, []any{}, errors.New("site options cannot be nil")
	}

	if err := options.Validate(); err != nil {
		return nil, []any{}, err
	}

	q := goqu.Dialect(store.dbDriverName).From(store.siteTableName)

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

	if options.HasDomainName() {
		q = q.Where(goqu.C(COLUMN_DOMAIN_NAMES).ILike(`%"` + options.DomainName() + `"%`))
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
		q = q.Where(goqu.C(COLUMN_NAME).ILike(`%` + options.NameLike() + `%`))
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
		return q, columns, nil // soft deleted sites requested specifically
	}

	softDeleted := goqu.C(COLUMN_SOFT_DELETED_AT).
		Gt(carbon.Now(carbon.UTC).ToDateTimeString())

	return q.Where(softDeleted), columns, nil
}
