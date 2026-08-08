package domain

import (
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
)

const pgForeignKeyViolationCode = "23503"

// ErrProductCategoryNotFound é retornado quando a categoria informada não existe (ou não pertence à empresa).
var ErrProductCategoryNotFound = errors.New("categoria informada não existe")

// TranslateProductDBError converte erros de constraint do Postgres em erros de domínio conhecidos,
// para que a camada HTTP possa responder com um status apropriado (400) em vez de 500.
func TranslateProductDBError(err error) error {
	if err == nil {
		return nil
	}

	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return err
	}

	if pgErr.Code == pgForeignKeyViolationCode && pgErr.ConstraintName == "fk_products_category" {
		return ErrProductCategoryNotFound
	}

	return err
}
