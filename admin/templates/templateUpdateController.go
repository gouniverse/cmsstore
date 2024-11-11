package admin

import (
	"net/http"

	"github.com/gouniverse/api"
	"github.com/gouniverse/bs"
	"github.com/gouniverse/cdn"
	"github.com/gouniverse/cmsstore"
	"github.com/gouniverse/cmsstore/admin/shared"
	"github.com/gouniverse/form"
	"github.com/gouniverse/hb"
	"github.com/gouniverse/router"
	"github.com/gouniverse/utils"
)

const VIEW_SETTINGS = "settings"
const VIEW_CONTENT = "content"

const codemirrorCss = "//cdnjs.cloudflare.com/ajax/libs/codemirror/3.20.0/codemirror.min.css"
const codemirrorJs = "//cdnjs.cloudflare.com/ajax/libs/codemirror/3.20.0/codemirror.min.js"
const codemirrorXmlJs = "//cdnjs.cloudflare.com/ajax/libs/codemirror/3.20.0/mode/xml/xml.min.js"
const codemirrorHtmlmixedJs = "//cdnjs.cloudflare.com/ajax/libs/codemirror/3.20.0/mode/htmlmixed/htmlmixed.min.js"
const codemirrorJavascriptJs = "//cdnjs.cloudflare.com/ajax/libs/codemirror/3.20.0/mode/javascript/javascript.js"
const codemirrorCssJs = "//cdnjs.cloudflare.com/ajax/libs/codemirror/3.20.0/mode/css/css.js"
const codemirrorClikeJs = "//cdnjs.cloudflare.com/ajax/libs/codemirror/3.20.0/mode/clike/clike.min.js"
const codemirrorPhpJs = "//cdnjs.cloudflare.com/ajax/libs/codemirror/3.20.0/mode/php/php.min.js"
const codemirrorFormattingJs = "//cdnjs.cloudflare.com/ajax/libs/codemirror/2.36.0/formatting.min.js"
const codemirrorMatchBracketsJs = "//cdnjs.cloudflare.com/ajax/libs/codemirror/3.22.0/addon/edit/matchbrackets.min.js"

// == CONTROLLER ==============================================================

type templateUpdateController struct {
	ui UiInterface
}

var _ router.HTMLControllerInterface = (*templateUpdateController)(nil)

// == CONSTRUCTOR =============================================================

func NewTemplateUpdateController(ui UiInterface) *templateUpdateController {
	return &templateUpdateController{
		ui: ui,
	}
}

func (controller *templateUpdateController) Handler(w http.ResponseWriter, r *http.Request) string {
	data, errorMessage := controller.prepareDataAndValidate(r)

	if errorMessage != "" {
		return api.Error(errorMessage).ToString()
	}

	if r.Method == http.MethodPost {
		return controller.form(data).ToHTML()
	}

	html := controller.page(data)

	options := struct {
		Styles     []string
		StyleURLs  []string
		Scripts    []string
		ScriptURLs []string
	}{
		Styles: []string{
			`.CodeMirror {
				border: 1px solid #eee;
				height: auto;
			}
			`,
		},
		StyleURLs: []string{
			codemirrorCss,
		},
		Scripts: []string{},
		ScriptURLs: []string{
			cdn.Sweetalert2_10(),
			cdn.Htmx_2_0_0(),
			cdn.Jquery_3_7_1(),
			codemirrorJs,
			codemirrorXmlJs,
			codemirrorHtmlmixedJs,
			codemirrorJavascriptJs,
			codemirrorCssJs,
			codemirrorClikeJs,
			codemirrorPhpJs,
			codemirrorFormattingJs,
			codemirrorMatchBracketsJs,
		},
	}

	return controller.ui.Layout(w, r, "Edit Template | CMS", html.ToHTML(), options)
}

func (controller templateUpdateController) page(data templateUpdateControllerData) hb.TagInterface {
	breadcrumbs := shared.Breadcrumbs([]shared.Breadcrumb{
		{
			Name: "Home",
			URL:  controller.ui.URL(controller.ui.Endpoint(), "", nil),
		},
		{
			Name: "CMS",
			URL:  controller.ui.URL(controller.ui.Endpoint(), "", nil),
		},
		{
			Name: "Template Manager",
			URL:  controller.ui.URL(controller.ui.Endpoint(), controller.ui.PathTemplateManager(), nil),
		},
		{
			Name: "Edit Template",
			URL:  controller.ui.URL(controller.ui.Endpoint(), controller.ui.PathTemplateUpdate(), map[string]string{"template_id": data.templateID}),
		},
	})

	buttonSave := hb.Button().
		Class("btn btn-primary ms-2 float-end").
		Child(hb.I().Class("bi bi-save").Style("margin-top:-4px;margin-right:8px;font-size:16px;")).
		HTML("Save").
		HxInclude("#FormTemplateUpdate").
		HxPost(controller.ui.URL(controller.ui.Endpoint(), controller.ui.PathTemplateUpdate(), map[string]string{"template_id": data.templateID})).
		HxTarget("#FormTemplateUpdate")

	buttonCancel := hb.Hyperlink().
		Class("btn btn-secondary ms-2 float-end").
		Child(hb.I().Class("bi bi-chevron-left").Style("margin-top:-4px;margin-right:8px;font-size:16px;")).
		HTML("Back").
		Href(controller.ui.URL(controller.ui.Endpoint(), controller.ui.PathTemplateManager(), nil))

	badgeStatus := hb.Div().
		Class("badge fs-6 ms-3").
		ClassIf(data.template.Status() == cmsstore.TEMPLATE_STATUS_ACTIVE, "bg-success").
		ClassIf(data.template.Status() == cmsstore.TEMPLATE_STATUS_INACTIVE, "bg-secondary").
		ClassIf(data.template.Status() == cmsstore.TEMPLATE_STATUS_DRAFT, "bg-warning").
		Text(data.template.Status())

	heading := hb.Heading1().
		Text("CMS. Edit Template:").
		Text(" ").
		Text(data.template.Name()).
		Child(hb.Sup().Child(badgeStatus)).
		Child(buttonSave).
		Child(buttonCancel)

	card := hb.Div().
		Class("card").
		Child(
			hb.Div().
				Class("card-header").
				Style(`display:flex;justify-content:space-between;align-items:center;`).
				Child(hb.Heading4().
					HTMLIf(data.view == VIEW_CONTENT, "Template Content").
					HTMLIf(data.view == VIEW_SETTINGS, "Template Settings").
					Style("margin-bottom:0;display:inline-block;")).
				Child(buttonSave),
		).
		Child(
			hb.Div().
				Class("card-body").
				Child(controller.form(data)))

	tabs := bs.NavTabs().
		Class("mb-3").
		Child(bs.NavItem().
			Child(bs.NavLink().
				ClassIf(data.view == VIEW_CONTENT, "active").
				Href(controller.ui.URL(controller.ui.Endpoint(), controller.ui.PathTemplateUpdate(), map[string]string{
					"template_id": data.templateID,
					"view":        VIEW_CONTENT,
				})).
				HTML("Content"))).
		Child(bs.NavItem().
			Child(bs.NavLink().
				ClassIf(data.view == VIEW_SETTINGS, "active").
				Href(controller.ui.URL(controller.ui.Endpoint(), controller.ui.PathTemplateUpdate(), map[string]string{
					"template_id": data.templateID,
					"view":        VIEW_SETTINGS,
				})).
				HTML("Settings")))

	return hb.Div().
		Class("container").
		Child(breadcrumbs).
		Child(hb.HR()).
		// HTML(header).
		Child(heading).
		// HTML(breadcrumbs).
		// Child(pageTitle).
		Child(tabs).
		Child(card)
}

func (controller templateUpdateController) form(data templateUpdateControllerData) hb.TagInterface {

	fieldsContent := controller.fieldsContent(data)
	fieldsSettings := controller.fieldsSettings(data)

	formpageUpdate := form.NewForm(form.FormOptions{
		ID: "FormTemplateUpdate",
	})

	if data.view == VIEW_SETTINGS {
		formpageUpdate.SetFields(fieldsSettings)
	}

	if data.view == VIEW_CONTENT {
		formpageUpdate.SetFields(fieldsContent)
	}

	if data.formErrorMessage != "" {
		formpageUpdate.AddField(&form.Field{
			Type:  form.FORM_FIELD_TYPE_RAW,
			Value: hb.Swal(hb.SwalOptions{Icon: "error", Text: data.formErrorMessage}).ToHTML(),
		})
	}

	if data.formSuccessMessage != "" {
		formpageUpdate.AddField(&form.Field{
			Type:  form.FORM_FIELD_TYPE_RAW,
			Value: hb.Swal(hb.SwalOptions{Icon: "success", Text: data.formSuccessMessage}).ToHTML(),
		})
	}

	return formpageUpdate.Build()
}

func (templateUpdateController) fieldsContent(data templateUpdateControllerData) []form.FieldInterface {
	fieldsContent := []form.FieldInterface{
		form.NewField(form.FieldOptions{
			Type: form.FORM_FIELD_TYPE_RAW,
			Value: hb.Div().
				Class(`alert alert-info`).
				Child(hb.Text("Available variables: [[PageContent]], [[PageCanonicalUrl]], [[PageMetaDescription]], [[PageMetaKeywords]], [[PageMetaRobots]], [[PageTitle]]")).
				ToHTML(),
		}),
		form.NewField(form.FieldOptions{
			Label: "Content (HTML)",
			Name:  "template_content",
			Type:  form.FORM_FIELD_TYPE_TEXTAREA,
			Value: data.formContent,
		}),
		form.NewField(form.FieldOptions{
			Label:    "Template ID",
			Name:     "template_id",
			Type:     form.FORM_FIELD_TYPE_HIDDEN,
			Value:    data.templateID,
			Readonly: true,
		}),
		form.NewField(form.FieldOptions{
			Label:    "View",
			Name:     "view",
			Type:     form.FORM_FIELD_TYPE_HIDDEN,
			Value:    VIEW_CONTENT,
			Readonly: true,
		}),
	}

	contentScript := hb.Script(`
function codeMirrorSelector() {
	return 'textarea[name="template_content"]';
}
function getCodeMirrorEditor() {
	return document.querySelector(codeMirrorSelector());
}
setTimeout(function () {
    console.log(getCodeMirrorEditor());
	if (getCodeMirrorEditor()) {
		var editor = CodeMirror.fromTextArea(getCodeMirrorEditor(), {
			lineNumbers: true,
			matchBrackets: true,
			mode: "application/x-httpd-php",
			indentUnit: 4,
			indentWithTabs: true,
			enterMode: "keep", tabMode: "shift"
		});
		$(document).on('mouseup', codeMirrorSelector(), function() {
			getCodeMirrorEditor().value = editor.getValue();
		});
		$(document).on('change', codeMirrorSelector(), function() {
			getCodeMirrorEditor().value = editor.getValue();
		});
		setInterval(()=>{
			getCodeMirrorEditor().value = editor.getValue();
		}, 1000)
	}
}, 500);
		`).ToHTML()

	fieldsContent = append(fieldsContent, &form.Field{
		Type:  form.FORM_FIELD_TYPE_RAW,
		Value: contentScript,
	})

	return fieldsContent
}

func (controller templateUpdateController) fieldsSettings(data templateUpdateControllerData) []form.FieldInterface {
	fieldsSettings := []form.FieldInterface{
		form.NewField(form.FieldOptions{
			Label: "Status",
			Name:  "template_status",
			Type:  form.FORM_FIELD_TYPE_SELECT,
			Value: data.formStatus,
			Help:  "The status of this webpage. Published pages will be displayed on the webtemplate.",
			Options: []form.FieldOption{
				{
					Value: "- not selected -",
					Key:   "",
				},
				{
					Value: "Draft",
					Key:   cmsstore.TEMPLATE_STATUS_DRAFT,
				},
				{
					Value: "Published",
					Key:   cmsstore.TEMPLATE_STATUS_ACTIVE,
				},
				{
					Value: "Unpublished",
					Key:   cmsstore.TEMPLATE_STATUS_INACTIVE,
				},
			},
		}),
		form.NewField(form.FieldOptions{
			Label: "Template Name (Internal)",
			Name:  "template_name",
			Type:  form.FORM_FIELD_TYPE_STRING,
			Value: data.formName,
			Help:  "The name of the template as displayed in the admin panel. This is not vsible to the template vistors",
		}),
		form.NewField(form.FieldOptions{
			Label: "Admin Notes (Internal)",
			Name:  "template_memo",
			Type:  form.FORM_FIELD_TYPE_TEXTAREA,
			Value: data.formMemo,
			Help:  "Admin notes for this template. These notes will not be visible to the public.",
		}),
		form.NewField(form.FieldOptions{
			Label:    "Webtemplate ID",
			Name:     "template_id",
			Type:     form.FORM_FIELD_TYPE_STRING,
			Value:    data.templateID,
			Readonly: true,
			Help:     "The reference number (ID) of the webtemplate. This is used to identify the webtemplate in the system and should not be changed.",
		}),
		form.NewField(form.FieldOptions{
			Label:    "View",
			Name:     "view",
			Type:     form.FORM_FIELD_TYPE_HIDDEN,
			Value:    data.view,
			Readonly: true,
		}),
	}

	return fieldsSettings
}

func (controller templateUpdateController) saveTemplate(r *http.Request, data templateUpdateControllerData) (d templateUpdateControllerData, errorMessage string) {
	data.formContent = utils.Req(r, "template_content", "")
	data.formMemo = utils.Req(r, "template_memo", "")
	data.formName = utils.Req(r, "template_name", "")
	data.formStatus = utils.Req(r, "template_status", "")
	data.formTitle = utils.Req(r, "template_title", "")

	if data.view == VIEW_SETTINGS {
		if data.formStatus == "" {
			data.formErrorMessage = "Status is required"
			return data, ""
		}
	}

	if data.view == VIEW_SETTINGS {
		data.template.SetMemo(data.formMemo)
		data.template.SetName(data.formName)
		data.template.SetStatus(data.formStatus)
	}

	if data.view == VIEW_CONTENT {
		data.template.SetContent(data.formContent)
	}

	err := controller.ui.Store().TemplateUpdate(data.template)

	if err != nil {
		//config.LogStore.ErrorWithContext("At templateUpdateController > prepareDataAndValidate", err.Error())
		data.formErrorMessage = "System error. Saving template failed. " + err.Error()
		return data, ""
	}

	data.formSuccessMessage = "template saved successfully"

	return data, ""
}

func (controller templateUpdateController) prepareDataAndValidate(r *http.Request) (data templateUpdateControllerData, errorMessage string) {
	data.action = utils.Req(r, "action", "")
	data.templateID = utils.Req(r, "template_id", "")
	data.view = utils.Req(r, "view", "")

	if data.view == "" {
		data.view = VIEW_CONTENT
	}

	if data.templateID == "" {
		return data, "template id is required"
	}

	var err error
	data.template, err = controller.ui.Store().TemplateFindByID(data.templateID)

	if err != nil {
		controller.ui.Logger().Error("At templateUpdateController > prepareDataAndValidate", "error", err.Error())
		return data, err.Error()
	}

	if data.template == nil {
		return data, "template not found"
	}

	data.formContent = data.template.Content()
	data.formName = data.template.Name()
	data.formMemo = data.template.Memo()
	data.formStatus = data.template.Status()

	if r.Method != http.MethodPost {
		return data, ""
	}

	return controller.saveTemplate(r, data)
}

type templateUpdateControllerData struct {
	action     string
	templateID string
	template   cmsstore.TemplateInterface
	view       string

	formErrorMessage   string
	formSuccessMessage string
	formContent        string
	formName           string
	formMemo           string
	formStatus         string
	formTitle          string
}