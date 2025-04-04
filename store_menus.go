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

// MenuCount returns the count of menus that match the provided query options.
func (store *store) MenuCount(ctx context.Context, options MenuQueryInterface) (int64, error) {
	options.SetCountOnly(true)

	q, _, err := store.menuSelectQuery(options)
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

// MenuCreate creates a new menu in the database.
func (store *store) MenuCreate(ctx context.Context, menu MenuInterface) error {
	if !store.menusEnabled {
		return errors.New("menus are disabled")
	}

	if menu == nil {
		return errors.New("menu is nil")
	}
	if menu.CreatedAt() == "" {
		menu.SetCreatedAt(carbon.Now(carbon.UTC).ToDateTimeString(carbon.UTC))
	}
	if menu.UpdatedAt() == "" {
		menu.SetUpdatedAt(carbon.Now(carbon.UTC).ToDateTimeString(carbon.UTC))
	}

	data := menu.Data()

	sqlStr, params, errSql := goqu.Dialect(store.dbDriverName).
		Insert(store.menuTableName).
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
		return errors.New("menustore: database is nil")
	}

	_, err := database.Execute(store.toQuerableContext(ctx), sqlStr, params...)
	if err != nil {
		return err
	}

	menu.MarkAsNotDirty()

	return nil
}

// MenuDelete deletes a menu from the database by its ID.
func (store *store) MenuDelete(ctx context.Context, menu MenuInterface) error {
	if !store.menusEnabled {
		return errors.New("menus are disabled")
	}

	if menu == nil {
		return errors.New("menu is nil")
	}

	return store.MenuDeleteByID(ctx, menu.ID())
}

// MenuDeleteByID deletes a menu from the database by its ID.
func (store *store) MenuDeleteByID(ctx context.Context, id string) error {
	if !store.menusEnabled {
		return errors.New("menus are disabled")
	}

	if id == "" {
		return errors.New("menu id is empty")
	}

	sqlStr, params, errSql := goqu.Dialect(store.dbDriverName).
		Delete(store.menuTableName).
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

// MenuFindByHandle finds a menu by its handle.
func (store *store) MenuFindByHandle(ctx context.Context, handle string) (menu MenuInterface, err error) {
	if !store.menusEnabled {
		return nil, errors.New("menus are disabled")
	}

	if handle == "" {
		return nil, errors.New("menu handle is empty")
	}

	list, err := store.MenuList(ctx, MenuQuery().
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

// MenuFindByID finds a menu by its ID.
func (store *store) MenuFindByID(ctx context.Context, id string) (menu MenuInterface, err error) {
	if !store.menusEnabled {
		return nil, errors.New("menus are disabled")
	}

	if id == "" {
		return nil, errors.New("menu id is empty")
	}

	list, err := store.MenuList(ctx, MenuQuery().SetID(id).SetLimit(1))
	if err != nil {
		return nil, err
	}

	if len(list) > 0 {
		return list[0], nil
	}

	return nil, nil
}

// MenuList returns a list of menus that match the provided query options.
func (store *store) MenuList(ctx context.Context, query MenuQueryInterface) ([]MenuInterface, error) {
	if !store.menusEnabled {
		return []MenuInterface{}, errors.New("menus are disabled")
	}

	q, columns, err := store.menuSelectQuery(query)
	if err != nil {
		return []MenuInterface{}, err
	}

	sqlStr, _, errSql := q.Select(columns...).ToSQL()
	if errSql != nil {
		return []MenuInterface{}, nil
	}

	if store.debugEnabled {
		log.Println(sqlStr)
	}

	if store.db == nil {
		return []MenuInterface{}, errors.New("menustore: database is nil")
	}

	modelMaps, err := database.SelectToMapString(store.toQuerableContext(ctx), sqlStr)
	if err != nil {
		return []MenuInterface{}, err
	}

	list := []MenuInterface{}
	lo.ForEach(modelMaps, func(modelMap map[string]string, index int) {
		model := NewMenuFromExistingData(modelMap)
		list = append(list, model)
	})

	return list, nil
}

// MenuSoftDelete marks a menu as soft-deleted by setting the soft_deleted_at timestamp.
func (store *store) MenuSoftDelete(ctx context.Context, menu MenuInterface) error {
	if !store.menusEnabled {
		return errors.New("menus are disabled")
	}

	if menu == nil {
		return errors.New("menu is nil")
	}

	menu.SetSoftDeletedAt(carbon.Now(carbon.UTC).ToDateTimeString(carbon.UTC))

	return store.MenuUpdate(ctx, menu)
}

// MenuSoftDeleteByID marks a menu as soft-deleted by its ID.
func (store *store) MenuSoftDeleteByID(ctx context.Context, id string) error {
	if !store.menusEnabled {
		return errors.New("menus are disabled")
	}

	menu, err := store.MenuFindByID(ctx, id)
	if err != nil {
		return err
	}

	return store.MenuSoftDelete(ctx, menu)
}

// MenuUpdate updates an existing menu in the database.
func (store *store) MenuUpdate(ctx context.Context, menu MenuInterface) error {
	if !store.menusEnabled {
		return errors.New("menus are disabled")
	}

	if menu == nil {
		return errors.New("menu is nil")
	}

	menu.SetUpdatedAt(carbon.Now(carbon.UTC).ToDateTimeString())

	dataChanged := menu.DataChanged()
	delete(dataChanged, COLUMN_ID) // ID is not updateable

	if len(dataChanged) < 1 {
		return nil
	}

	sqlStr, params, errSql := goqu.Dialect(store.dbDriverName).
		Update(store.menuTableName).
		Prepared(true).
		Set(dataChanged).
		Where(goqu.C(COLUMN_ID).Eq(menu.ID())).
		ToSQL()
	if errSql != nil {
		return errSql
	}

	if store.debugEnabled {
		log.Println(sqlStr)
	}

	if store.db == nil {
		return errors.New("menustore: database is nil")
	}

	_, err := database.Execute(store.toQuerableContext(ctx), sqlStr, params...)
	if err != nil {
		return err
	}

	menu.MarkAsNotDirty()

	return nil
}

// menuSelectQuery constructs a SQL query for selecting menus based on the provided query options.
func (store *store) menuSelectQuery(options MenuQueryInterface) (selectDataset *goqu.SelectDataset, columns []any, err error) {
	if options == nil {
		return nil, nil, errors.New("menu query cannot be nil")
	}

	if err := options.Validate(); err != nil {
		return nil, nil, err
	}

	q := goqu.Dialect(store.dbDriverName).From(store.menuTableName)

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
		return q, columns, nil // soft deleted menus requested specifically
	}

	softDeleted := goqu.C(COLUMN_SOFT_DELETED_AT).
		Gt(carbon.Now(carbon.UTC).ToDateTimeString())

	return q.Where(softDeleted), columns, nil
}
