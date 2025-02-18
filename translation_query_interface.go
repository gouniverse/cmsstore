package cmsstore

type TranslationQueryInterface interface {
	Validate() error

	Columns() []string
	SetColumns(columns []string) TranslationQueryInterface

	HasCountOnly() bool
	IsCountOnly() bool
	SetCountOnly(countOnly bool) TranslationQueryInterface

	HasCreatedAtGte() bool
	CreatedAtGte() string
	SetCreatedAtGte(createdAtGte string) TranslationQueryInterface

	HasCreatedAtLte() bool
	CreatedAtLte() string
	SetCreatedAtLte(createdAtLte string) TranslationQueryInterface

	HasHandle() bool
	Handle() string
	SetHandle(handle string) TranslationQueryInterface

	HasHandleOrID() bool
	HandleOrID() string
	SetHandleOrID(handleOrID string) TranslationQueryInterface

	HasID() bool
	ID() string
	SetID(id string) TranslationQueryInterface

	HasIDIn() bool
	IDIn() []string
	SetIDIn(idIn []string) TranslationQueryInterface

	HasNameLike() bool
	NameLike() string
	SetNameLike(nameLike string) TranslationQueryInterface

	HasOffset() bool
	Offset() int
	SetOffset(offset int) TranslationQueryInterface

	HasLimit() bool
	Limit() int
	SetLimit(limit int) TranslationQueryInterface

	HasSortOrder() bool
	SortOrder() string
	SetSortOrder(sortOrder string) TranslationQueryInterface

	HasOrderBy() bool
	OrderBy() string
	SetOrderBy(orderBy string) TranslationQueryInterface

	HasSiteID() bool
	SiteID() string
	SetSiteID(siteID string) TranslationQueryInterface

	HasSoftDeletedIncluded() bool
	SoftDeletedIncluded() bool
	SetSoftDeletedIncluded(includeSoftDeleted bool) TranslationQueryInterface

	HasStatus() bool
	Status() string
	SetStatus(status string) TranslationQueryInterface

	HasStatusIn() bool
	StatusIn() []string
	SetStatusIn(statusIn []string) TranslationQueryInterface
}
