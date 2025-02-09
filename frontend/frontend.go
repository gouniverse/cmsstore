package frontend

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"regexp"
	"sort"
	"strings"

	// "github.com/gouniverse/cms/types"
	"github.com/gouniverse/cmsstore"
	"github.com/gouniverse/hb"
	"github.com/gouniverse/shortcode"
	"github.com/gouniverse/ui"
	"github.com/gouniverse/utils"
	"github.com/jellydator/ttlcache/v3"
	"github.com/mingrammer/cfmt"
	"github.com/samber/lo"
)

type frontend struct {
	blockEditorRenderer func(blocks []ui.BlockInterface) string
	logger              *slog.Logger
	shortcodes          []cmsstore.ShortcodeInterface
	store               cmsstore.StoreInterface
	cacheEnabled        bool
	cacheExpireSeconds  int
	cache               *ttlcache.Cache[string, any]
}

var _ FrontendInterface = (*frontend)(nil)

// Handler is the main handler for the CMS frontend.
//
// It handles the routing of the request to the appropriate page.
//
// If the URI ends with ".ico", it will return a blank response, as the browsers
// (at least Chrome and Firefox) will always request the favicon even if
// it's not present in the HTML.
func (frontend *frontend) Handler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte(frontend.StringHandler(w, r)))
}

// FrontendHandlerRenderAsString is the same as FrontendHandler but returns a string
// instead of writing to the http.ResponseWriter.
//
// It handles the routing of the request to the appropriate page.
//
// If the URI ends with ".ico", it will return a blank response, as the browsers
// (at least Chrome and Firefox) will always request the favicon even if
// it's not present in the HTML.
//
// If the translations are enabled, it will use the language from the request context.
// If the language is not valid, it will use the default language for the translations.
func (frontend *frontend) StringHandler(w http.ResponseWriter, r *http.Request) string {
	domain := r.Host
	path := r.URL.Path

	uri := r.RequestURI

	if strings.HasSuffix(uri, ".ico") {
		return ""
	}

	languageAny := r.Context().Value(LanguageKey{})
	language := utils.ToString(languageAny)

	// if fr.translationsEnabled {
	// 	isValidLanguage := lo.Contains(lo.Keys(cms.translationLanguages), language)

	// 	if !isValidLanguage {
	// 		language = cms.translationLanguageDefault
	// 	}
	// }

	site, siteEnpoint, err := frontend.findSiteAndEndpointByDomainAndPath(r.Context(), domain, path)

	if err != nil {
		frontend.logger.Error(`At StringHandler`, "error", err.Error())
		return `Domain not supported: ` + domain
	}

	if site == nil {
		return `Domain not supported: ` + domain
	}

	calculatedPath := strings.TrimPrefix(domain+path, siteEnpoint)

	return frontend.PageRenderHtmlBySiteAndAlias(w, r, site.ID(), calculatedPath, language)
}

// fetchBlockContent returns the content of the block specified by the ID
//
// Business Logic:
// - if the block find returns an error error is returned
// - if the block is not active an empty string is returned
// - the block content is returned
//
// Parameters:
// - blockID: the ID of the block
//
// Returns:
// - content: the content of the block
func (frontend *frontend) fetchBlockContent(ctx context.Context, blockID string) (string, error) {
	if blockID == "" {
		return "", nil
	}

	key := "block_content_" + blockID

	if frontend.CacheHas(key) {
		blockContent := frontend.CacheGet(key)

		if blockContent == nil {
			return "", nil
		}

		return blockContent.(string), nil
	}

	block, err := frontend.store.BlockFindByID(ctx, blockID)

	if err != nil {
		frontend.CacheSet(key, "", 10) // 10 seconds only, error
		return "", err
	}

	if block == nil {
		frontend.CacheSet(key, "", frontend.cacheExpireSeconds)
		return "", nil
	}

	content := ""

	if block.IsActive() {
		content = block.Content()
	}

	frontend.CacheSet(key, content, frontend.cacheExpireSeconds)

	return content, nil
}

func (frontend *frontend) fetchPageAliasMapBySite(ctx context.Context, siteID string) (map[string]string, error) {
	cacheKey := "page_alias_map_site:" + siteID

	if frontend.CacheHas(cacheKey) {
		pageAliasMap := frontend.CacheGet(cacheKey)

		if pageAliasMap == nil {
			return map[string]string{}, nil // cache value is nil
		}

		return pageAliasMap.(map[string]string), nil
	}

	pages, err := frontend.store.PageList(ctx, cmsstore.PageQuery().
		SetSiteID(siteID).
		SetColumns([]string{cmsstore.COLUMN_ID, cmsstore.COLUMN_ALIAS}))

	if err != nil {
		return nil, err
	}

	pageAliasMap := make(map[string]string, len(pages))

	for _, page := range pages {
		pageAliasMap[page.ID()] = page.Alias()
	}

	frontend.CacheSet(cacheKey, pageAliasMap, frontend.cacheExpireSeconds)

	return pageAliasMap, nil
}

func (frontend *frontend) fetchPageBySiteAndAlias(ctx context.Context, siteID string, alias string) (cmsstore.PageInterface, error) {
	cacheKey := "page_site:" + siteID + ":alias:" + alias

	if frontend.CacheHas(cacheKey) {
		page := frontend.CacheGet(cacheKey)

		if page == nil {
			return nil, nil // cache value is nil
		}

		return page.(cmsstore.PageInterface), nil
	}

	pages, err := frontend.store.PageList(ctx, cmsstore.PageQuery().
		SetSiteID(siteID).
		SetAlias(alias).
		SetLimit(1))

	if err != nil {
		return nil, err
	}

	var page cmsstore.PageInterface = nil

	if len(pages) > 0 {
		page = pages[0]
	}

	frontend.CacheSet(cacheKey, page, frontend.cacheExpireSeconds)

	return page, nil
}

// fetchActiveSites fetches the active sites from the database and stores them
// in the cache to avoid an extra database query every time this method is called
func (frontend *frontend) fetchActiveSites(ctx context.Context) ([]cmsstore.SiteInterface, error) {
	cacheKey := "sites_active"

	if frontend.CacheHas(cacheKey) {
		sites := frontend.CacheGet(cacheKey)

		if sites == nil {
			return []cmsstore.SiteInterface{}, nil
		}

		return sites.([]cmsstore.SiteInterface), nil
	}

	sites, err := frontend.store.SiteList(ctx, cmsstore.SiteQuery().
		SetStatus(cmsstore.SITE_STATUS_ACTIVE).
		SetColumns([]string{cmsstore.COLUMN_ID, cmsstore.COLUMN_DOMAIN_NAMES}))

	if err != nil {
		frontend.CacheSet(cacheKey, []cmsstore.SiteInterface{}, 10) // 10 seconds only, error
		return nil, err
	}

	frontend.CacheSet(cacheKey, sites, frontend.cacheExpireSeconds)

	return sites, nil
}

// findSiteAndEndpointByDomainAndPath returns the site and site endpoint
// for the given domain and path
//
// Note! a site endpoint can be a domain, subdomain or subdirectory
//
// Business Logic:
// - fetches active sites
// - maps the site endpoints to sites
// - sorts site endpoints by length (longest first)
// - matches the site endpoint as a prefix in the full page path (domain + path)
// - returns the site and site endpoint
// - results are cached in memory, to not fetch the same data multiple times
func (frontend *frontend) findSiteAndEndpointByDomainAndPath(ctx context.Context, domain string, path string) (site cmsstore.SiteInterface, endpoint string, err error) {
	key1 := "find_site_and_endpoint_site" + domain + path
	key2 := "find_site_and_endpoint_endpoint" + domain + path

	if frontend.CacheHas(key1) && frontend.CacheHas(key2) {
		site := frontend.CacheGet(key1)

		if site == nil {
			return nil, "", nil
		}

		endpoint := frontend.CacheGet(key2)

		if endpoint == nil {
			return nil, "", nil
		}

		return site.(cmsstore.SiteInterface), endpoint.(string), nil
	}

	sites, err := frontend.fetchActiveSites(ctx)

	if err != nil {
		return nil, "", err
	}

	domainNamesSiteMap := map[string]cmsstore.SiteInterface{}

	for _, site := range sites {
		domainNames, err := site.DomainNames()

		if err != nil {
			return nil, "", err
		}

		for _, domainName := range domainNames {
			domainNamesSiteMap[domainName] = site
		}
	}

	pagePath := domain + path

	keys := lo.Keys(domainNamesSiteMap)

	// sort keys by length desc
	sort.Slice(keys, func(i, j int) bool {
		return len(keys[i]) > len(keys[j])
	})

	// find the website, starting with the longest key
	for _, siteEndpoint := range keys {
		if strings.HasPrefix(pagePath, siteEndpoint) {

			frontend.CacheSet(key1, domainNamesSiteMap[siteEndpoint], frontend.cacheExpireSeconds)
			frontend.CacheSet(key2, siteEndpoint, frontend.cacheExpireSeconds)

			return domainNamesSiteMap[siteEndpoint], siteEndpoint, nil
		}
	}

	frontend.CacheSet(key1, nil, 10) // 10 seconds only, not found
	frontend.CacheSet(key2, "", 10)  // 10 seconds only, not found

	return nil, "", nil
}

// PageRenderHtmlBySiteAndAlias generates and returns the HTML content of a page identified by its alias and site ID.
//
// It follows these steps:
// 1. Fetch the page by site ID and alias.
// 2. If the page is not found, log an error and return a "not found" message.
// 3. Retrieve page attributes such as content, metadata, and editor type.
// 4. If the page uses the block editor, convert its JSON content to HTML.
// 5. Retrieve applicable middlewares from the page metadata.
// 6. If a template is associated with the page, fetch and apply it.
// 7. Render the final HTML using the collected page data.
// 8. Apply middlewares to the rendered HTML and return the final output.
//
// Errors encountered during page retrieval, template fetching, or HTML rendering are logged appropriately.
//
// Parameters:
// - w (http.ResponseWriter): The HTTP response writer.
// - r (*http.Request): The HTTP request.
// - siteID (string): The ID of the site where the page is located.
// - alias (string): The unique alias used to identify the page within the site.
// - language (string): The language code for rendering language-specific content.
//
// Returns:
// - string: The fully rendered HTML of the page, including templates and middleware transformations.
func (frontend *frontend) PageRenderHtmlBySiteAndAlias(w http.ResponseWriter, r *http.Request, siteID, alias, language string) string {
	// Attempt to find the page by site ID and alias.
	page, err := frontend.pageFindBySiteAndAlias(r.Context(), siteID, alias)
	if err != nil {
		frontend.logger.Error("PageRenderHtmlBySiteAndAlias: Error finding page", "alias", alias, "error", err)
		return hb.NewDiv().Text("Error loading page").ToHTML()
	}
	if page == nil {
		frontend.logger.Warn("PageRenderHtmlBySiteAndAlias: Page not found", "alias", alias)
		return hb.NewDiv().Text("Page with alias '").Text(alias).Text("' not found").ToHTML()
	}

	// Retrieve page content and determine if block editor transformation is needed.
	pageContent := page.Content()
	if page.Editor() == cmsstore.PAGE_EDITOR_BLOCKEDITOR {
		pageContent = frontend.convertBlockJsonToHtml(pageContent)
	}

	// Determine if a template is associated with the page and apply it.
	finalContent := lo.If(page.TemplateID() == "", pageContent).ElseF(func() string {
		template, err := frontend.store.TemplateFindByID(r.Context(), page.TemplateID())
		if err != nil {
			frontend.logger.Error("PageRenderHtmlBySiteAndAlias: Template load error", "templateID", page.TemplateID(), "error", err)
			return pageContent
		}

		if template == nil {
			return pageContent
		}

		return template.Content()
	})

	// Collect metadata to be passed into the template.
	pageData := struct {
		PageContent         string
		PageCanonicalURL    string
		PageMetaDescription string
		PageMetaKeywords    string
		PageMetaRobots      string
		PageTitle           string
		Language            string
	}{
		PageContent:         pageContent,
		PageCanonicalURL:    page.CanonicalUrl(),
		PageMetaDescription: page.MetaDescription(),
		PageMetaKeywords:    page.MetaKeywords(),
		PageMetaRobots:      page.MetaRobots(),
		PageTitle:           page.Title(),
		Language:            language,
	}

	// Render the final HTML output based on the collected page data.
	html, err := frontend.renderContentToHtml(r, finalContent, pageData)
	if err != nil {
		frontend.logger.Error("PageRenderHtmlBySiteAndAlias: Rendering error", "error", err)
		return hb.NewDiv().Text("Error occurred").ToHTML()
	}

	pageMiddlewaresBefore := page.MiddlewaresBefore()
	pageMiddlewaresAfter := page.MiddlewaresAfter()

	// Apply middleware transformations to the rendered HTML before returning the final result.
	return frontend.applyMiddlewares(w, r, html, pageMiddlewaresBefore, pageMiddlewaresAfter)
}

func (frontend *frontend) pageMiddlewaresFromMeta(page cmsstore.PageInterface) []string {
	meta := page.Meta("middlewares")

	if meta == "" {
		return []string{}
	}

	m, err := utils.FromJSON(page.Meta("middlewares"), []string{})

	if err != nil {
		cfmt.Error(err)
		return []string{}
	}

	return lo.Map(m.([]interface{}), func(v interface{}, _ int) string {
		return v.(string)
	})
}

func (frontend *frontend) convertBlockJsonToHtml(blocksJson string) string {
	if frontend.blockEditorRenderer == nil {
		return "Block editor not configured"
	}

	if !utils.IsJSON(blocksJson) {
		return "Malformed block content"
	}

	blocks, err := ui.UnmarshalJsonToBlocks(blocksJson)

	if err != nil {
		return "Error parsing block content"
	}

	return frontend.blockEditorRenderer(blocks)
}

// renderContentToHtml renders the content to HTML
//
// This is done in the following steps (sequence is important):
// 1. replaces placeholders with values
// 2. renders the blocks
// 3. renders the shortcodes
// 3. renders the translations
// 4. returns the HTML
//
// Parameters:
// - r: the HTTP request
// - content: the content to render
// - options: the options for the rendering
//
// Returns:
// - html: the rendered HTML
// - err: the error, if any, or nil otherwise
func (frontend *frontend) renderContentToHtml(r *http.Request, content string, options struct {
	PageContent         string
	PageCanonicalURL    string
	PageMetaDescription string
	PageMetaKeywords    string
	PageMetaRobots      string
	PageTitle           string
	Language            string
}) (html string, err error) {
	replacements := map[string]string{
		"PageContent":         options.PageContent,
		"PageCanonicalUrl":    options.PageCanonicalURL,
		"PageMetaDescription": options.PageMetaDescription,
		"PageMetaKeywords":    options.PageMetaKeywords,
		"PageRobots":          options.PageMetaRobots,
		"PageTitle":           options.PageTitle,
	}

	for key, value := range replacements {
		content = strings.ReplaceAll(content, "[["+key+"]]", value)
		content = strings.ReplaceAll(content, "[[ "+key+" ]]", value)
	}

	content, err = frontend.contentRenderBlocks(r.Context(), content)

	if err != nil {
		return "", err
	}

	content, err = frontend.ContentRenderShortcodes(r, content)

	if err != nil {
		return "", err
	}

	language := lo.If(options.Language == "", "en").Else(options.Language)

	content, err = frontend.contentRenderTranslations(content, language)

	if err != nil {
		return "", err
	}

	return content, nil
}

// pageFindBySiteAndAlias helper method to find a page by site and alias
//
// =====================================================================
//  1. It will attempt to find the page by the provided site and alias exactly
//     as provided
//  2. It will attempt to find the page with the site and the alias prefixed with "/"
//     in case of error
//
// =====================================================================
func (frontend *frontend) pageFindBySiteAndAlias(ctx context.Context, siteID string, alias string) (cmsstore.PageInterface, error) {
	// Try to find by "alias"
	page, err := frontend.fetchPageBySiteAndAlias(ctx, siteID, alias)

	if err != nil {
		return nil, err
	}

	if page != nil {
		return page, nil
	}

	// Try to find by "/alias"
	page, err = frontend.fetchPageBySiteAndAlias(ctx, siteID, "/"+alias)

	if err != nil {
		return nil, err
	}

	if page != nil {
		return page, nil
	}

	page, err = frontend.pageFindBySiteAndAliasWithPatterns(ctx, siteID, alias)

	if err != nil {
		return nil, err
	}

	if page != nil {
		return page, nil
	}

	return nil, nil
}

// PageFindByAliasWithPatterns helper method to find a page by matching patterns
//
// =====================================================================
//
//	The following patterns are supported:
//	:any
//	:num
//	:all
//	:string
//	:number
//	:numeric
//	:alpha
//
// =====================================================================
func (frontend *frontend) pageFindBySiteAndAliasWithPatterns(ctx context.Context, siteID string, alias string) (cmsstore.PageInterface, error) {
	patterns := map[string]string{
		":any":     "([^/]+)",
		":num":     "([0-9]+)",
		":all":     "(.*)",
		":string":  "([a-zA-Z]+)",
		":number":  "([0-9]+)",
		":numeric": "([0-9-.]+)",
		":alpha":   "([a-zA-Z0-9-_]+)",
	}

	pageAliasMap, err := frontend.fetchPageAliasMapBySite(ctx, siteID)

	if err != nil {
		return nil, err
	}

	for pageID, pageAlias := range pageAliasMap {
		if !strings.Contains(pageAlias, ":") {
			continue
		}

		for pattern, replacement := range patterns {
			pageAlias = strings.ReplaceAll(pageAlias, pattern, replacement)
		}

		matcher := regexp.MustCompile("^" + pageAlias + "$")
		if matcher.MatchString(alias) {
			return frontend.store.PageFindByID(ctx, pageID)
		}
	}

	return nil, nil
}

// RenderBlocks renders the blocks in a string
func (frontend *frontend) contentRenderBlocks(ctx context.Context, content string) (string, error) {
	blockIDs := contentFindIdsByPatternPrefix(content, "BLOCK")

	if len(blockIDs) == 0 {
		return content, nil
	}

	var err error

	for _, blockID := range blockIDs {
		content, err = frontend.contentRenderBlockByID(ctx, content, blockID)

		if err != nil {
			return content, err
		}
	}

	return content, nil
}

// contentRenderTranslations renders the translations in a string
func (frontend *frontend) contentRenderTranslations(content string, language string) (string, error) {
	translationIDs := contentFindIdsByPatternPrefix(content, "TRANSLATION")

	if len(translationIDs) == 0 {
		return content, nil
	}

	var err error
	for _, translationID := range translationIDs {
		content, err = frontend.ContentRenderTranslationByIdOrHandle(content, translationID, language)

		if err != nil {
			return content, err
		}
	}

	return content, nil
}

// returns the IDs in the content who have the following format [[prefix_id]]
func contentFindIdsByPatternPrefix(content, prefix string) []string {
	ids := []string{}

	re := regexp.MustCompilePOSIX("|\\[\\[" + prefix + "_(.*)\\]\\]|U")

	matches := re.FindAllStringSubmatch(content, -1)

	for _, match := range matches {
		if match[0] == "" {
			continue
		}
		if match[1] == "" {
			continue // no need to add empty IDs
		}
		ids = append(ids, match[1])
	}

	return ids
}

// ContentRenderBlockByID renders the block specified by the ID in the content
//
// Business Logic:
// - if the blockID is empty the initial content is returned
// - if the block content returns an error the initial content is returned
// - the block tag is replaced by the block content in the initial content
//
// Parameters:
// - content: the content to render
// - blockID: the ID of the block
//
// Returns:
// - content: the rendered content
func (frontend *frontend) contentRenderBlockByID(ctx context.Context, content string, blockID string) (string, error) {
	if blockID == "" {
		return content, nil
	}

	blockContent, err := frontend.fetchBlockContent(ctx, blockID)

	if err != nil {
		return content, err
	}

	content = strings.ReplaceAll(content, "[[BLOCK_"+blockID+"]]", blockContent)
	content = strings.ReplaceAll(content, "[[ BLOCK_"+blockID+" ]]", blockContent)

	return content, nil
}

// ContentRenderShortcodes renders the shortcodes in a string
func (frontend *frontend) ContentRenderShortcodes(req *http.Request, content string) (string, error) {
	sh, err := shortcode.NewShortcode(shortcode.WithBrackets("<", ">"))

	if err != nil {
		return "", err
	}

	for _, shortcode := range frontend.shortcodes {
		content = sh.RenderWithRequest(req, content, shortcode.Alias(), shortcode.Render)
	}

	return content, nil
}

// ContentRenderTranslationByIdOrHandle renders the translation specified by the ID in a content
// if the blockID is empty or not found the initial content is returned
func (frontend *frontend) ContentRenderTranslationByIdOrHandle(content string, translationID string, language string) (string, error) {
	return content, nil

	// Will be implemented once translations are transferred

	// if translationID == "" {
	// 	return content, nil
	// }

	// translation, err := frontend.store.TranslationFindByIdOrHandle(translationID, language)

	// if err != nil {
	// 	return "", err
	// }

	// content = strings.ReplaceAll(content, "[[TRANSLATION_"+translationID+"]]", translation)
	// content = strings.ReplaceAll(content, "[[ TRANSLATION_"+translationID+" ]]", translation)

	// return content, nil
}

// TemplateRenderHtmlByID builds the HTML of a template based on its ID
func (frontend *frontend) TemplateRenderHtmlByID(
	r *http.Request,
	templateID string,
	options TemplateRenderHtmlByIDOptions,
) (string, error) {
	if templateID == "" {
		return "", errors.New("template id is empty")
	}

	template, err := frontend.store.TemplateFindByID(r.Context(), templateID)

	if err != nil {
		return "", err
	}

	if template == nil {
		return "", errors.New("template not found")
	}

	if !template.IsActive() {
		return "", errors.New("template " + templateID + " is not active")
	}

	content := template.Content()

	html, err := frontend.renderContentToHtml(r, content, options)

	if err != nil {
		return "", err
	}

	return html, nil
}
