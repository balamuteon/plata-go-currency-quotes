package quote

import (
	"time"

	"github.com/balamuteon/plata-go-currency-quotes/internal/domain"
)

type requestUpdateDTO struct {
	Pair string `json:"pair"`
}

type updateResponseDTO struct {
	UpdateID  string              `json:"update_id"`
	Pair      string              `json:"pair"`
	Status    domain.UpdateStatus `json:"status"`
	Price     *float64            `json:"price,omitempty"`
	CreatedAt time.Time           `json:"created_at"`
	UpdatedAt time.Time           `json:"updated_at"`
	Error     string              `json:"error,omitempty"`
}

func updateFromDomain(update domain.QuoteUpdate) updateResponseDTO {
	result := updateResponseDTO{
		UpdateID:  update.ID,
		Pair:      update.Pair.String(),
		Status:    update.Status,
		Price:     update.Price,
		CreatedAt: update.CreatedAt,
		UpdatedAt: update.UpdatedAt,
	}
	if update.Status == domain.StatusFailed {
		result.Error = "quote update failed after retries"
	}
	return result
}

type latestResponseDTO struct {
	Pair      string    `json:"pair"`
	Price     float64   `json:"price"`
	UpdatedAt time.Time `json:"updated_at"`
	UpdateID  string    `json:"update_id"`
}

func latestFromDomain(update domain.QuoteUpdate) latestResponseDTO {
	return latestResponseDTO{
		Pair:      update.Pair.String(),
		Price:     *update.Price,
		UpdatedAt: update.UpdatedAt,
		UpdateID:  update.ID,
	}
}
