package cmsstore

import (
	"database/sql"
	"os"

	"github.com/gouniverse/utils"
	_ "modernc.org/sqlite"
)

func initDB(filepath string) *sql.DB {
	if filepath != ":memory:" && utils.FileExists(filepath) {
		err := os.Remove(filepath) // remove database

		if err != nil {
			panic(err)
		}
	}

	dsn := filepath + "?parseTime=true"
	db, err := sql.Open("sqlite", dsn)

	if err != nil {
		panic(err)
	}

	return db
}

func initStore(filepath string) (StoreInterface, error) {
	db := initDB(filepath)

	store, err := NewStore(NewStoreOptions{
		DB:                 db,
		BlockTableName:     "block_table",
		PageTableName:      "page_table",
		SiteTableName:      "site_table",
		TemplateTableName:  "template_table",
		MenusEnabled:       true,
		MenuTableName:      "menu_table",
		MenuItemTableName:  "menu_item_table",
		AutomigrateEnabled: true,
	})

	if err != nil {
		return nil, err
	}

	return store, nil
}
