package cmsstore

type PageQueryInterface interface {
	Validate() error

	Columns() []string
	SetColumns(columns []string) PageQueryInterface

	HasAlias() bool
	Alias() string
	SetAlias(alias string) PageQueryInterface

	HasAliasLike() bool
	AliasLike() string
	SetAliasLike(nameLike string) PageQueryInterface

	HasCreatedAtGte() bool
	CreatedAtGte() string
	SetCreatedAtGte(createdAtGte string) PageQueryInterface

	HasCreatedAtLte() bool
	CreatedAtLte() string
	SetCreatedAtLte(createdAtLte string) PageQueryInterface

	HasCountOnly() bool
	IsCountOnly() bool
	SetCountOnly(countOnly bool) PageQueryInterface

	HasHandle() bool
	Handle() string
	SetHandle(handle string) PageQueryInterface

	HasID() bool
	ID() string
	SetID(id string) PageQueryInterface

	HasIDIn() bool
	IDIn() []string
	SetIDIn(idIn []string) PageQueryInterface

	HasLimit() bool
	Limit() int
	SetLimit(limit int) PageQueryInterface

	HasNameLike() bool
	NameLike() string
	SetNameLike(nameLike string) PageQueryInterface

	HasOffset() bool
	Offset() int
	SetOffset(offset int) PageQueryInterface

	HasOrderBy() bool
	OrderBy() string
	SetOrderBy(orderBy string) PageQueryInterface

	HasSiteID() bool
	SiteID() string
	SetSiteID(siteID string) PageQueryInterface

	HasSortOrder() bool
	SortOrder() string
	SetSortOrder(sortOrder string) PageQueryInterface

	HasSoftDeletedIncluded() bool
	SoftDeletedIncluded() bool
	SetSoftDeletedIncluded(softDeleteIncluded bool) PageQueryInterface

	HasStatus() bool
	Status() string
	SetStatus(status string) PageQueryInterface

	HasStatusIn() bool
	StatusIn() []string
	SetStatusIn(statusIn []string) PageQueryInterface

	HasTemplateID() bool
	TemplateID() string
	SetTemplateID(templateID string) PageQueryInterface
}
