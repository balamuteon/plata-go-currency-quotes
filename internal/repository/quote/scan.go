package quote

import "github.com/balamuteon/plata-go-currency-quotes/internal/domain"

type rowScanner interface {
	Scan(dest ...any) error
}

const returningFields = `
	id::text,
	pair,
	status,
	price,
	attempts,
	created_at,
	updated_at,
	COALESCE(lease_token::text, ''),
	COALESCE(last_error, '')`

func scanQuoteUpdate(row rowScanner) (domain.QuoteUpdate, error) {
	var (
		update domain.QuoteUpdate
		pair   string
		status string
	)

	err := row.Scan(
		&update.ID,
		&pair,
		&status,
		&update.Price,
		&update.Attempts,
		&update.CreatedAt,
		&update.UpdatedAt,
		&update.LeaseToken,
		&update.ErrorMessage,
	)
	if err != nil {
		return domain.QuoteUpdate{}, err
	}

	update.Status, err = domain.ParseUpdateStatus(status)
	if err != nil {
		return domain.QuoteUpdate{}, err
	}

	update.Pair, err = domain.ParseCurrencyPair(pair)
	if err != nil {
		return domain.QuoteUpdate{}, err
	}

	return update, nil
}
