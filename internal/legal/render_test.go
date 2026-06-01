package legal

import (
	"strings"
	"testing"
)

func TestPrivacyPolicyPopulatesCompanyMeta(t *testing.T) {
	doc := PrivacyPolicy()
	if doc.Meta.CompanyName != CompanyName {
		t.Fatalf("company: got %q want %q", doc.Meta.CompanyName, CompanyName)
	}
	if doc.Meta.ContactEmail != ContactEmail {
		t.Fatalf("contact: got %q want %q", doc.Meta.ContactEmail, ContactEmail)
	}
	if strings.Contains(doc.Content, "{{") {
		t.Fatal("unexpanded placeholder in privacy content")
	}
	if !strings.Contains(doc.Content, ProductsURL) {
		t.Fatal("expected products URL in privacy content")
	}
}

func TestTermsOfServiceSlug(t *testing.T) {
	doc := TermsOfService()
	if doc.Slug != "terms" {
		t.Fatalf("slug: got %q", doc.Slug)
	}
}
