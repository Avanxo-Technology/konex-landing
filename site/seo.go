package site

import "strings"

// shareImageURL is the absolute URL of the page's share image (og:image and
// twitter:image), built from the Cockpit "webapp" singleton: its "Default
// Image" (defaultImage) and "Site URL" (siteUrl). Nothing is hard-coded, so
// changing the image in Cockpit changes the preview.
//
// Why this exists: core builds the SEO image from the asset path and, with
// local storage, emits it relative ("/storage/uploads/..."). WhatsApp, Facebook,
// LinkedIn and X ignore a relative og:image, so no preview shows. The blog addon
// already makes its image absolute with the site URL; the generic SEO tags do
// not. Delete this file once core does.
//
// storageAdapter and s3PublicURL are the STORAGE_ADAPTER and S3_PUBLIC_URL
// env vars: they mirror how core picks the asset base (S3 public URL when set,
// otherwise the local /storage/uploads mount).
//
// It returns "" when there is no image or no way to make it absolute, which
// leaves core's own value untouched.
func shareImageURL(webapp map[string]any, storageAdapter, s3PublicURL string) string {
	img, _ := webapp["defaultImage"].(map[string]any)
	path, _ := img["path"].(string)
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}

	// The CMS already gave a full URL.
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		return path
	}
	if strings.HasPrefix(path, "//") {
		return "https:" + path
	}

	base := "/storage/uploads"
	if storageAdapter == "s3" && strings.TrimSpace(s3PublicURL) != "" {
		base = strings.TrimRight(strings.TrimSpace(s3PublicURL), "/")
	}
	url := base + "/" + strings.TrimLeft(path, "/")
	if strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://") {
		return url
	}

	siteURL, _ := webapp["siteUrl"].(string)
	siteURL = strings.TrimRight(strings.TrimSpace(siteURL), "/")
	if siteURL == "" {
		return ""
	}
	return siteURL + url
}
