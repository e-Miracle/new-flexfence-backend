package legal

const (
	CompanyName       = "Emiracle Cybernetics LTD"
	AppName           = "FlexFence"
	ContactEmail      = "flexfence@emiracle.me"
	ProductsURL       = "https://emiracle.me/products"
	DocumentVersion   = "1.0"
	DocumentEffective = "2026-05-30"
)

// Meta is returned with every legal document for client display and auto-population.
type Meta struct {
	CompanyName    string `json:"company_name"`
	AppName        string `json:"app_name"`
	ContactEmail   string `json:"contact_email"`
	ProductsURL    string `json:"products_url"`
	EffectiveDate  string `json:"effective_date"`
	Version        string `json:"version"`
}

func CurrentMeta() Meta {
	return Meta{
		CompanyName:   CompanyName,
		AppName:       AppName,
		ContactEmail:  ContactEmail,
		ProductsURL:   ProductsURL,
		EffectiveDate: DocumentEffective,
		Version:       DocumentVersion,
	}
}
