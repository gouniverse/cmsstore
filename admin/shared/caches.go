package shared

import (
	"net/http"
	"strings"

	"github.com/gouniverse/cmsstore"
)

// CachedSitesActive returns a list of active sites, caching the result for 2 minutes
func CachedSitesActive(store cmsstore.StoreInterface) ([]cmsstore.SiteInterface, error) {
	const cacheExpireSeconds = 2 * 60 // 2 minutes

	key := "sites_active"
	if InMemCache.Has(key) {
		sites, err := InMemCache.Get(key)

		if err != nil {
			return nil, err
		}

		return sites.([]cmsstore.SiteInterface), err
	}

	sites, err := store.SiteList(cmsstore.SiteQuery().
		SetStatus(cmsstore.SITE_STATUS_ACTIVE).
		SetColumns([]string{cmsstore.COLUMN_ID, cmsstore.COLUMN_DOMAIN_NAMES}))

	if err != nil {
		InMemCache.Set(key, []cmsstore.SiteInterface{}, cacheExpireSeconds)
		return nil, err
	}

	InMemCache.Set(key, sites, cacheExpireSeconds)

	return sites, nil
}

// CachedSiteList returns a list of all sites, caching the result for 2 minutes
func CachedSiteList(store cmsstore.StoreInterface) ([]cmsstore.SiteInterface, error) {
	const cacheExpireSeconds = 2 * 60 // 2 minutes

	key := "site_list"

	if InMemCache.Has(key) {
		sites, err := InMemCache.Get(key)

		if err != nil {
			return nil, err
		}

		return sites.([]cmsstore.SiteInterface), err
	}

	sites, err := store.SiteList(cmsstore.SiteQuery().
		SetColumns([]string{
			cmsstore.COLUMN_ID,
			cmsstore.COLUMN_DOMAIN_NAMES,
			cmsstore.COLUMN_NAME,
		}))

	if err != nil {
		InMemCache.Set(key, []cmsstore.SiteInterface{}, cacheExpireSeconds)
		return nil, err
	}

	InMemCache.Set(key, sites, cacheExpireSeconds)

	return sites, nil
}

// CachedSiteFindByID returns a site by ID, caching the result for 2 minutes
func CachedSiteByID(store cmsstore.StoreInterface, siteID string) (cmsstore.SiteInterface, error) {
	list, err := CachedSiteList(store)

	if err != nil {
		return nil, err
	}

	for _, site := range list {
		if site.ID() == siteID {
			return site, nil
		}
	}

	return nil, nil
}

// CachedSiteURL returns a site URL, caching the result for 2 minutes
func CachedSiteURL(r *http.Request, store cmsstore.StoreInterface, siteID string) (string, error) {
	const cacheExpireSeconds = 2 * 60 // 2 minutes
	site, err := CachedSiteByID(store, siteID)

	if err != nil {
		return "", err
	}

	key := "site_url:" + siteID

	domains, err := site.DomainNames()

	if err != nil {
		InMemCache.Set(key, "", cacheExpireSeconds)

		return "", err
	}

	if len(domains) > 0 {
		url := "https://" + domains[0]

		if r.TLS == nil {
			url = "http://" + domains[0]
		}

		InMemCache.Set(key, url, cacheExpireSeconds)

		return url, nil
	}

	InMemCache.Set(key, "", cacheExpireSeconds)

	return "", nil
}

func PageURL(r *http.Request, store cmsstore.StoreInterface, storeID string, pageAlias string) (string, error) {
	siteURL, err := CachedSiteURL(r, store, storeID)

	if err != nil {
		return "", err
	}

	return siteURL + "/" + strings.TrimPrefix(pageAlias, "/"), nil
}