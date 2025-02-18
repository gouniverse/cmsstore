package cmsstore

import (
	"context"
	"database/sql"
	"errors"

	"github.com/gouniverse/base/database"
	"github.com/gouniverse/versionstore"
)

// == TYPE ====================================================================

type store struct {
	blockTableName     string
	pageTableName      string
	siteTableName      string
	templateTableName  string
	db                 *sql.DB
	dbDriverName       string
	automigrateEnabled bool
	debugEnabled       bool

	// Menus
	menusEnabled      bool
	menuTableName     string
	menuItemTableName string

	// Translations
	translationsEnabled        bool
	translationTableName       string
	translationLanguages       map[string]string
	translationLanguageDefault string

	versioningEnabled   bool
	versioningTableName string
	versioningStore     versionstore.StoreInterface

	// Shortcodes
	shortcodes  []ShortcodeInterface
	middlewares []MiddlewareInterface
}

// == INTERFACE ===============================================================

var _ StoreInterface = (*store)(nil) // verify it extends the interface

// PUBLIC METHODS ============================================================

// AutoMigrate auto migrate
func (store *store) AutoMigrate(context context.Context, opts ...Option) error {
	if store.db == nil {
		return errors.New("cms store: database is nil")
	}

	options := &Options{}
	for _, opt := range opts {
		opt(options)
	}

	transaction, hasTransaction := options.params["tx"].(*sql.Tx)
	isDryRun, hasDryRun := options.params["dryRun"].(bool)

	blockSql := store.blockTableCreateSql()
	menuSql := store.menuTableCreateSql()
	menuItemSql := store.menuItemTableCreateSql()
	pageSql := store.pageTableCreateSql()
	tableSql := store.siteTableCreateSql()
	templateSql := store.templateTableCreateSql()
	translationSql := store.translationTableCreateSql()

	if blockSql == "" {
		return errors.New("block table create sql is empty")
	}

	if pageSql == "" {
		return errors.New("page table create sql is empty")
	}

	if tableSql == "" {
		return errors.New("site table create sql is empty")
	}

	if templateSql == "" {
		return errors.New("template table create sql is empty")
	}

	if store.menusEnabled && store.menuTableName == "" {
		return errors.New("menu table name is empty")
	}

	if store.menusEnabled && store.menuItemTableName == "" {
		return errors.New("menu item table name is empty")
	}

	if store.translationsEnabled && translationSql == "" {
		return errors.New("translation table create sql is empty")
	}

	if store.versioningEnabled && store.versioningTableName == "" {
		return errors.New("versioning table name is empty")
	}

	sqlList := []string{
		blockSql,
		pageSql,
		tableSql,
		templateSql,
	}

	if store.menusEnabled {
		sqlList = append(sqlList, menuSql)
		sqlList = append(sqlList, menuItemSql)
	}

	if store.translationsEnabled {
		sqlList = append(sqlList, translationSql)
	}

	for _, sql := range sqlList {
		if hasDryRun && isDryRun {
			continue
		}

		if hasTransaction {
			_, err := transaction.ExecContext(context, sql)

			if err != nil {
				return err
			}

			continue
		} else {
			_, err := store.db.ExecContext(context, sql)

			if err != nil {
				return err
			}
		}
	}

	if store.versioningEnabled {
		err := store.versioningStore.AutoMigrate()

		if err != nil {
			return err
		}
	}

	return nil
}

// EnableDebug - enables the debug option
func (st *store) EnableDebug(debug bool) {
	st.debugEnabled = debug
}

func (store *store) MenusEnabled() bool {
	return store.menusEnabled
}

func (store *store) TranslationsEnabled() bool {
	return store.translationsEnabled
}

func (store *store) VersioningEnabled() bool {
	return store.versioningEnabled
}

func (store *store) VersioningCreate(ctx context.Context, version VersioningInterface) error {
	return store.versioningStore.VersionCreate(store.toQuerableContext(ctx), version)
}

func (store *store) VersioningDelete(ctx context.Context, version VersioningInterface) error {
	return store.versioningStore.VersionDelete(store.toQuerableContext(ctx), version)
}

func (store *store) VersioningDeleteByID(ctx context.Context, id string) error {
	return store.versioningStore.VersionDeleteByID(store.toQuerableContext(ctx), id)
}

func (store *store) VersioningFindByID(ctx context.Context, versioningID string) (VersioningInterface, error) {
	return store.versioningStore.VersionFindByID(store.toQuerableContext(ctx), versioningID)
}

func (store *store) VersioningList(ctx context.Context, query VersioningQueryInterface) ([]VersioningInterface, error) {
	list, err := store.versioningStore.VersionList(store.toQuerableContext(ctx), query)

	if err != nil {
		return nil, err
	}

	newlist := make([]VersioningInterface, len(list))

	for i, v := range list {
		newlist[i] = v
	}

	return newlist, nil
}

func (store *store) VersioningSoftDelete(ctx context.Context, versioning VersioningInterface) error {
	return store.versioningStore.VersionSoftDelete(store.toQuerableContext(ctx), versioning)
}

func (store *store) VersioningSoftDeleteByID(ctx context.Context, id string) error {
	return store.versioningStore.VersionSoftDeleteByID(store.toQuerableContext(ctx), id)
}

func (store *store) VersioningUpdate(ctx context.Context, version VersioningInterface) error {
	return store.versioningStore.VersionUpdate(store.toQuerableContext(ctx), version)
}

func (store *store) Shortcodes() []ShortcodeInterface {
	return store.shortcodes
}

func (store *store) AddShortcode(shortcode ShortcodeInterface) {
	store.shortcodes = append(store.shortcodes, shortcode)
}

func (store *store) AddShortcodes(shortcodes []ShortcodeInterface) {
	store.shortcodes = append(store.shortcodes, shortcodes...)
}

func (store *store) SetShortcodes(shortcodes []ShortcodeInterface) {
	store.shortcodes = shortcodes
}

func (store *store) Middlewares() []MiddlewareInterface {
	return store.middlewares
}

func (store *store) AddMiddleware(middleware MiddlewareInterface) {
	store.middlewares = append(store.middlewares, middleware)
}

func (store *store) AddMiddlewares(middlewares []MiddlewareInterface) {
	store.middlewares = append(store.middlewares, middlewares...)
}

func (store *store) SetMiddlewares(middlewares []MiddlewareInterface) {
	store.middlewares = middlewares
}

func (store *store) toQuerableContext(context context.Context) database.QueryableContext {
	if database.IsQueryableContext(context) {
		return context.(database.QueryableContext)
	}

	return database.Context(context, store.db)
}
