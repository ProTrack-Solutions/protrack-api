package enums

type CompanySettingsKey string

const (
	IsWhatsappActive            CompanySettingsKey = "is_whatsapp_active"
	IsExcessUsage               CompanySettingsKey = "is_excess_usage"
	LowStock                    CompanySettingsKey = "low_stock"
	MediumStock                 CompanySettingsKey = "mediun_stock"
	NormalStock                 CompanySettingsKey = "normal_stock"
	SaleOverdueTemplate         CompanySettingsKey = "sale_overdue_template"
	LanguageSaleOverdueTemplate CompanySettingsKey = "language_sale_overdue_template"
)
