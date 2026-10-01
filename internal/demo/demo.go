// Package demo mantém a empresa de demonstração usada na landing page
// ("Ver demonstração"). Os IDs são fixos para que o reset diário recrie os
// dados sem invalidar as sessões abertas e para que o resto da API consiga
// identificar a empresa demo sem consultar o banco.
package demo

import "github.com/google/uuid"

var (
	CompanyID = uuid.MustParse("de300000-0000-4000-8000-000000000001")
	UserID    = uuid.MustParse("de300000-0000-4000-8000-000000000002")
)

const UserEmail = "demo@protrack.com.br"

// IsDemoCompany indica se o company_id pertence à empresa de demonstração.
func IsDemoCompany(companyID uuid.UUID) bool {
	return companyID == CompanyID
}
