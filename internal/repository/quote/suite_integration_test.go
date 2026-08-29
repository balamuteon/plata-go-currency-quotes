package quote_test

import (
	"context"
	"testing"
	"time"

	quoterepository "github.com/balamuteon/plata-go-currency-quotes/internal/repository/quote"
	"github.com/balamuteon/plata-go-currency-quotes/pkg/testdb"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/suite"
)

type RepositorySuite struct {
	suite.Suite
	dbPool     *pgxpool.Pool
	repository *quoterepository.Repository
	teardown   func()
}

func (s *RepositorySuite) SetupSuite() {
	ctx := context.Background()

	dsn, teardown, err := testdb.SetupTestDatabase(ctx)
	s.Require().NoError(err, "failed to setup test database container")
	s.teardown = teardown

	pool, err := pgxpool.New(ctx, dsn)
	s.Require().NoError(err, "failed to connect to DB")
	s.dbPool = pool

	s.repository = quoterepository.NewRepository(pool)
}

func (s *RepositorySuite) TearDownSuite() {
	if s.dbPool != nil {
		s.dbPool.Close()
	}
	if s.teardown != nil {
		s.teardown()
	}
}

func (s *RepositorySuite) SetupTest() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, err := s.dbPool.Exec(ctx, "TRUNCATE TABLE quote_updates RESTART IDENTITY CASCADE")
	s.Require().NoError(err, "failed to truncate tables")
}

func TestRepositorySuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(RepositorySuite))
}
