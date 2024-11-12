package admin

import (
	"net/http"
	"strings"

	"github.com/gouniverse/api"
	"github.com/gouniverse/bs"
	"github.com/gouniverse/cdn"
	"github.com/gouniverse/cmsstore"
	"github.com/gouniverse/cmsstore/admin/shared"
	"github.com/gouniverse/form"
	"github.com/gouniverse/hb"
	"github.com/gouniverse/router"
	"github.com/gouniverse/sb"
	"github.com/gouniverse/utils"
	"github.com/samber/lo"
	"github.com/spf13/cast"
)

const ActionModalPageFilterShow = "modal_translation_filter_show"

// == CONTROLLER ==============================================================

type translationManagerController struct {
	ui UiInterface
}

var _ router.HTMLControllerInterface = (*translationManagerController)(nil)

// == CONSTRUCTOR =============================================================

func NewTranslationManagerController(ui UiInterface) *translationManagerController {
	return &translationManagerController{
		ui: ui,
	}
}

func (controller *translationManagerController) Handler(w http.ResponseWriter, r *http.Request) string {
	data, errorMessage := controller.prepareData(r)

	if errorMessage != "" {
		return api.Error(errorMessage).ToString()
	}

	if data.action == ActionModalPageFilterShow {
		return controller.onModalRecordFilterShow(data).ToHTML()
	}

	options := struct {
		Styles     []string
		StyleURLs  []string
		Scripts    []string
		ScriptURLs []string
	}{
		ScriptURLs: []string{
			cdn.Htmx_2_0_0(),
			cdn.Sweetalert2_11(),
		},
	}
	return controller.ui.Layout(w, r, "Translation Manager | CMS", controller.page(data).ToHTML(), options)
}

func (controller *translationManagerController) onModalRecordFilterShow(data translationManagerControllerData) *hb.Tag {
	modalCloseScript := `document.getElementById('ModalMessage').remove();document.getElementById('ModalBackdrop').remove();`

	title := hb.Heading5().
		Text("Filters").
		Style(`margin:0px;padding:0px;`)

	buttonModalClose := hb.Button().Type("button").
		Class("btn-close").
		Data("bs-dismiss", "modal").
		OnClick(modalCloseScript)

	buttonCancel := hb.Button().
		Child(hb.I().Class("bi bi-chevron-left me-2")).
		HTML("Cancel").
		Class("btn btn-secondary float-start").
		OnClick(modalCloseScript)

	buttonOk := hb.Button().
		Child(hb.I().Class("bi bi-check me-2")).
		HTML("Apply").
		Class("btn btn-primary float-end").
		OnClick(`FormFilters.submit();` + modalCloseScript)

	filterForm := form.NewForm(form.FormOptions{
		ID:        "FormFilters",
		Method:    http.MethodGet,
		ActionURL: shared.URL(controller.ui.Endpoint(), shared.PathTranslationsTranslationManager, nil),
		Fields: []form.FieldInterface{
			form.NewField(form.FieldOptions{
				Label: "Status",
				Name:  "status",
				Type:  form.FORM_FIELD_TYPE_SELECT,
				Help:  `The status of the translation.`,
				Value: data.formStatus,
				Options: []form.FieldOption{
					{
						Value: "",
						Key:   "",
					},
					{
						Value: "Active",
						Key:   cmsstore.SITE_STATUS_ACTIVE,
					},
					{
						Value: "Inactive",
						Key:   cmsstore.SITE_STATUS_INACTIVE,
					},
					{
						Value: "Draft",
						Key:   cmsstore.SITE_STATUS_DRAFT,
					},
				},
			}),
			form.NewField(form.FieldOptions{
				Label: "Name",
				Name:  "name",
				Type:  form.FORM_FIELD_TYPE_STRING,
				Value: data.formName,
				Help:  `Filter by name.`,
			}),
			form.NewField(form.FieldOptions{
				Label: "Created From",
				Name:  "created_from",
				Type:  form.FORM_FIELD_TYPE_DATE,
				Value: data.formCreatedFrom,
				Help:  `Filter by creation date.`,
			}),
			form.NewField(form.FieldOptions{
				Label: "Created To",
				Name:  "created_to",
				Type:  form.FORM_FIELD_TYPE_DATE,
				Value: data.formCreatedTo,
				Help:  `Filter by creation date.`,
			}),
			form.NewField(form.FieldOptions{
				Label: "Translation ID",
				Name:  "translation_id",
				Type:  form.FORM_FIELD_TYPE_STRING,
				Value: data.formTranslationID,
				Help:  `Find translation by reference number (ID).`,
			}),
			form.NewField(form.FieldOptions{
				Label: "Path",
				Name:  "path",
				Type:  form.FORM_FIELD_TYPE_STRING,
				Value: shared.PathTranslationsTranslationManager,
				Help:  `Path to this page.`,
			}),
		},
	}).Build()

	modal := bs.Modal().
		ID("ModalMessage").
		Class("fade show").
		Style(`display:block;position:fixed;top:50%;left:50%;transform:translate(-50%,-50%);z-index:1051;`).
		Children([]hb.TagInterface{
			bs.ModalDialog().Children([]hb.TagInterface{
				bs.ModalContent().Children([]hb.TagInterface{
					bs.ModalHeader().Children([]hb.TagInterface{
						title,
						buttonModalClose,
					}),

					bs.ModalBody().
						Child(filterForm),

					bs.ModalFooter().
						Style(`display:flex;justify-content:space-between;`).
						Child(buttonCancel).
						Child(buttonOk),
				}),
			}),
		})

	backdrop := hb.Div().
		ID("ModalBackdrop").
		Class("modal-backdrop fade show").
		Style("display:block;")

	return hb.Wrap().Children([]hb.TagInterface{
		modal,
		backdrop,
	})

}

func (controller *translationManagerController) page(data translationManagerControllerData) hb.TagInterface {
	adminHeader := controller.ui.AdminHeader()

	adminHomeBreadcrumb := lo.If(controller.ui.AdminHomeURL() != "", shared.Breadcrumb{
		Name: "Home",
		URL:  controller.ui.AdminHomeURL(),
	}).Else(shared.Breadcrumb{})

	breadcrumbs := shared.Breadcrumbs([]shared.Breadcrumb{
		adminHomeBreadcrumb,
		{
			Name: "CMS",
			URL:  shared.URL(controller.ui.Endpoint(), "", nil),
		},
		{
			Name: "Translation Manager",
			URL:  shared.URL(controller.ui.Endpoint(), shared.PathTranslationsTranslationManager, nil),
		},
	})

	buttonPageNew := hb.Button().
		Class("btn btn-primary float-end").
		Child(hb.I().Class("bi bi-plus-circle").Style("margin-top:-4px;margin-right:8px;font-size:16px;")).
		HTML("New Translation").
		HxGet(shared.URL(controller.ui.Endpoint(), shared.PathTranslationsTranslationCreate, nil)).
		HxTarget("body").
		HxSwap("beforeend")

	title := hb.Heading1().
		HTML("CMS. Translation Manager").
		Child(buttonPageNew)

	return hb.Div().
		Class("container").
		Child(breadcrumbs).
		Child(hb.HR()).
		Child(adminHeader).
		Child(hb.HR()).
		Child(title).
		Child(controller.tableRecords(data))
}

func (controller *translationManagerController) tableRecords(data translationManagerControllerData) hb.TagInterface {
	table := hb.Table().
		Class("table table-striped table-hover table-bordered").
		Children([]hb.TagInterface{
			hb.Thead().Children([]hb.TagInterface{
				hb.TR().Children([]hb.TagInterface{
					hb.TH().
						Child(controller.sortableColumnLabel(data, "Name", cmsstore.COLUMN_NAME)).
						Text(", ").
						Child(controller.sortableColumnLabel(data, "Domains", cmsstore.COLUMN_DOMAIN_NAMES)).
						Text(", ").
						Child(controller.sortableColumnLabel(data, "Reference", cmsstore.COLUMN_ID)).
						Style(`cursor: pointer;`),
					hb.TH().
						Child(controller.sortableColumnLabel(data, "Status", cmsstore.COLUMN_STATUS)).
						Style("width: 200px;cursor: pointer;"),
					hb.TH().
						Child(controller.sortableColumnLabel(data, "Created", cmsstore.COLUMN_CREATED_AT)).
						Style("width: 1px;cursor: pointer;"),
					hb.TH().
						Child(controller.sortableColumnLabel(data, "Modified", cmsstore.COLUMN_UPDATED_AT)).
						Style("width: 1px;cursor: pointer;"),
					hb.TH().
						HTML("Actions"),
				}),
			}),
			hb.Tbody().Children(lo.Map(data.recordList, func(translation cmsstore.TranslationInterface, _ int) hb.TagInterface {

				translationName := translation.Name()

				translationLink := hb.Hyperlink().
					Text(translationName).
					Href(shared.URL(controller.ui.Endpoint(), shared.PathTranslationsTranslationUpdate, map[string]string{
						"translation_id": translation.ID(),
					}))

				status := hb.Span().
					Style(`font-weight: bold;`).
					StyleIf(translation.IsActive(), `color:green;`).
					StyleIf(translation.IsSoftDeleted(), `color:silver;`).
					StyleIf(translation.IsInactive(), `color:red;`).
					HTML(translation.Status())

				buttonEdit := hb.Hyperlink().
					Class("btn btn-primary me-2").
					Child(hb.I().Class("bi bi-pencil-square")).
					Title("Edit").
					Href(shared.URL(controller.ui.Endpoint(), shared.PathTranslationsTranslationUpdate, map[string]string{
						"translation_id": translation.ID(),
					}))

				buttonDelete := hb.Hyperlink().
					Class("btn btn-danger").
					Child(hb.I().Class("bi bi-trash")).
					Title("Delete").
					HxGet(shared.URL(controller.ui.Endpoint(), shared.PathTranslationsTranslationDelete, map[string]string{
						"translation_id": translation.ID(),
					})).
					HxTarget("body").
					HxSwap("beforeend")

				return hb.TR().Children([]hb.TagInterface{
					hb.TD().
						Child(hb.Div().Child(translationLink)).
						Child(hb.Div().
							Style("font-size: 11px;").
							HTML("Ref: ").
							HTML(translation.ID())),
					hb.TD().
						Child(status),
					hb.TD().
						Child(hb.Div().
							Style("font-size: 13px;white-space: nowrap;").
							HTML(translation.CreatedAtCarbon().Format("d M Y"))),
					hb.TD().
						Child(hb.Div().
							Style("font-size: 13px;white-space: nowrap;").
							HTML(translation.UpdatedAtCarbon().Format("d M Y"))),
					hb.TD().
						Child(buttonEdit).
						// Child(buttonImpersonate).
						Child(buttonDelete),
				})
			})),
		})

	// cfmt.Successln("Table: ", table)

	return hb.Wrap().Children([]hb.TagInterface{
		controller.tableFilter(data),
		table,
		controller.tablePagination(data, int(data.recordCount), data.pageInt, data.perPage),
	})
}

func (controller *translationManagerController) sortableColumnLabel(data translationManagerControllerData, tableLabel string, columnName string) hb.TagInterface {
	isSelected := strings.EqualFold(data.sortBy, columnName)

	direction := lo.If(data.sortOrder == sb.ASC, sb.DESC).Else(sb.ASC)

	if !isSelected {
		direction = sb.ASC
	}

	link := shared.URL(controller.ui.Endpoint(), shared.PathTranslationsTranslationManager, map[string]string{
		"page":           "0",
		"by":             columnName,
		"sort":           direction,
		"date_from":      data.formCreatedFrom,
		"date_to":        data.formCreatedTo,
		"status":         data.formStatus,
		"translation_id": data.formTranslationID,
	})
	return hb.Hyperlink().
		HTML(tableLabel).
		Child(controller.sortingIndicator(columnName, data.sortBy, direction)).
		Href(link)
}

func (controller *translationManagerController) sortingIndicator(columnName string, sortByColumnName string, sortOrder string) hb.TagInterface {
	isSelected := strings.EqualFold(sortByColumnName, columnName)

	direction := lo.If(isSelected && sortOrder == "asc", "up").
		ElseIf(isSelected && sortOrder == "desc", "down").
		Else("none")

	sortingIndicator := hb.Span().
		Class("sorting").
		HTMLIf(direction == "up", "&#8595;").
		HTMLIf(direction == "down", "&#8593;").
		HTMLIf(direction != "down" && direction != "up", "")

	return sortingIndicator
}

func (controller *translationManagerController) tableFilter(data translationManagerControllerData) hb.TagInterface {
	buttonFilter := hb.Button().
		Class("btn btn-sm btn-info me-2").
		Style("margin-bottom: 2px; margin-left:2px; margin-right:2px;").
		Child(hb.I().Class("bi bi-filter me-2")).
		Text("Filters").
		HxPost(shared.URL(controller.ui.Endpoint(), shared.PathTranslationsTranslationManager, map[string]string{
			"action":         ActionModalPageFilterShow,
			"name":           data.formName,
			"status":         data.formStatus,
			"translation_id": data.formTranslationID,
			"created_from":   data.formCreatedFrom,
			"created_to":     data.formCreatedTo,
		})).
		HxTarget("body").
		HxSwap("beforeend")

	description := []string{
		hb.Span().HTML("Showing translations").Text(" ").ToHTML(),
	}

	if data.formStatus != "" {
		description = append(description, hb.Span().Text("with status: "+data.formStatus).ToHTML())
	} else {
		description = append(description, hb.Span().Text("with status: any").ToHTML())
	}

	if data.formName != "" {
		description = append(description, hb.Span().Text("and name: "+data.formName).ToHTML())
	}

	if data.formTranslationID != "" {
		description = append(description, hb.Span().Text("and ID: "+data.formTranslationID).ToHTML())
	}

	if data.formCreatedFrom != "" && data.formCreatedTo != "" {
		description = append(description, hb.Span().Text("and created between: "+data.formCreatedFrom+" and "+data.formCreatedTo).ToHTML())
	} else if data.formCreatedFrom != "" {
		description = append(description, hb.Span().Text("and created after: "+data.formCreatedFrom).ToHTML())
	} else if data.formCreatedTo != "" {
		description = append(description, hb.Span().Text("and created before: "+data.formCreatedTo).ToHTML())
	}

	return hb.Div().
		Class("card bg-light mb-3").
		Style("").
		Children([]hb.TagInterface{
			hb.Div().Class("card-body").
				Child(buttonFilter).
				Child(hb.Span().
					HTML(strings.Join(description, " "))),
		})
}

func (controller *translationManagerController) tablePagination(data translationManagerControllerData, count int, page int, perPage int) hb.TagInterface {
	url := shared.URL(controller.ui.Endpoint(), shared.PathTranslationsTranslationManager, map[string]string{
		"status":       data.formStatus,
		"name":         data.formName,
		"created_from": data.formCreatedFrom,
		"created_to":   data.formCreatedTo,
		"by":           data.sortBy,
		"order":        data.sortOrder,
	})

	url = lo.Ternary(strings.Contains(url, "?"), url+"&page=", url+"?page=") // page must be last

	pagination := bs.Pagination(bs.PaginationOptions{
		NumberItems:       count,
		CurrentPageNumber: page,
		PagesToShow:       5,
		PerPage:           perPage,
		URL:               url,
	})

	return hb.Div().
		Class(`d-flex justify-content-left mt-5 pagination-primary-soft rounded mb-0`).
		HTML(pagination)
}

func (controller *translationManagerController) prepareData(r *http.Request) (data translationManagerControllerData, errorMessage string) {
	var err error
	initialPerPage := 20
	data.request = r
	data.action = utils.Req(r, "action", "")
	data.page = utils.Req(r, "page", "0")
	data.pageInt = cast.ToInt(data.page)
	data.perPage = cast.ToInt(utils.Req(r, "per_page", cast.ToString(initialPerPage)))
	data.sortOrder = utils.Req(r, "sort", sb.DESC)
	data.sortBy = utils.Req(r, "by", cmsstore.COLUMN_CREATED_AT)
	data.formName = utils.Req(r, "name", "")
	data.formStatus = utils.Req(r, "status", "")
	data.formCreatedFrom = utils.Req(r, "created_from", "")
	data.formCreatedTo = utils.Req(r, "created_to", "")

	recordList, recordCount, err := controller.fetchRecordList(data)

	if err != nil {
		controller.ui.Logger().Error("At translationManagerController > prepareData", "error", err.Error())
		return data, "error retrieving web translations"
	}

	data.recordList = recordList
	data.recordCount = recordCount

	return data, ""
}

func (controller *translationManagerController) fetchRecordList(data translationManagerControllerData) (records []cmsstore.TranslationInterface, recordCount int64, err error) {
	translationIDs := []string{}

	if data.formTranslationID != "" {
		translationIDs = append(translationIDs, data.formTranslationID)
	}

	// if data.formCreatedFrom != "" {
	// 	query.CreatedAtGte = data.formCreatedFrom + " 00:00:00"
	// }

	// if data.formCreatedTo != "" {
	// 	query.CreatedAtLte = data.formCreatedTo + " 23:59:59"
	// }

	query := cmsstore.TranslationQuery().
		SetLimit(data.perPage).
		SetOffset(data.pageInt * data.perPage).
		SetOrderBy(data.sortBy).
		SetSortOrder(data.sortOrder)

	if len(translationIDs) > 0 {
		query.SetIDIn(translationIDs)
	}

	if data.formStatus != "" {
		query.SetStatus(data.formStatus)
	}

	if data.formName != "" {
		query.SetNameLike(data.formName)
	}

	recordList, err := controller.ui.Store().TranslationList(query)

	if err != nil {
		return []cmsstore.TranslationInterface{}, 0, err
	}

	recordCount, err = controller.ui.Store().TranslationCount(query)

	if err != nil {
		return []cmsstore.TranslationInterface{}, 0, err
	}

	return recordList, recordCount, nil
}

type translationManagerControllerData struct {
	request           *http.Request
	action            string
	page              string
	pageInt           int
	perPage           int
	sortOrder         string
	sortBy            string
	formStatus        string
	formName          string
	formCreatedFrom   string
	formCreatedTo     string
	formTranslationID string
	recordList        []cmsstore.TranslationInterface
	recordCount       int64
}