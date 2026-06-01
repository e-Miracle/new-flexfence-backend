package legal

import "strings"

// Document is the API payload for a legal page.
type Document struct {
	Slug    string `json:"slug"`
	Title   string `json:"title"`
	Content string `json:"content"` // Markdown with placeholders already expanded.
	Meta    Meta   `json:"meta"`
}

func expand(template string, m Meta) string {
	r := strings.NewReplacer(
		"{{COMPANY_NAME}}", m.CompanyName,
		"{{APP_NAME}}", m.AppName,
		"{{CONTACT_EMAIL}}", m.ContactEmail,
		"{{PRODUCTS_URL}}", m.ProductsURL,
		"{{EFFECTIVE_DATE}}", m.EffectiveDate,
	)
	return r.Replace(template)
}

func PrivacyPolicy() Document {
	m := CurrentMeta()
	return Document{
		Slug:    "privacy",
		Title:   "Privacy Policy",
		Content: expand(privacyPolicyTemplate, m),
		Meta:    m,
	}
}

func TermsOfService() Document {
	m := CurrentMeta()
	return Document{
		Slug:    "terms",
		Title:   "Terms of Service",
		Content: expand(termsOfServiceTemplate, m),
		Meta:    m,
	}
}
