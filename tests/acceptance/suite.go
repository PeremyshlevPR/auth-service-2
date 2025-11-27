package acceptance

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"
	"github.com/prperemyshlev/auth-service-2/pkg/database"
	"github.com/stretchr/testify/suite"
)

const (
	postgresDSN = "postgres://auth_service:auth_service_password@localhost:5432/auth_service_db?sslmode=disable"
	redisDSN    = "localhost:6379"
)

// Suite represents the test suite for acceptance tests
type Suite struct {
	suite.Suite
	Postgres *database.Postgres
	Redis    *database.Redis
	App      *TestApp
}

func (s *Suite) SetupSuite() {
	pg, err := database.NewPostgres(postgresDSN)
	if err != nil {
		s.T().Fatalf("Failed to connect to PostgreSQL: %v", err)
	}

	redis, err := database.NewRedis(redisDSN, "", 0)
	if err != nil {
		pg.Close()
		s.T().Fatalf("Failed to connect to Redis: %v", err)
	}

	if err := s.setupDatabase(pg.DB); err != nil {
		pg.Close()
		redis.Close()
		s.T().Fatalf("Failed to run migrations: %v", err)
	}

	app, err := NewTestApp(pg, redis)
	if err != nil {
		_ = pg.Close()
		_ = redis.Close()
		s.T().Fatalf("Failed to create test app: %v", err)
	}

	s.Postgres = pg
	s.Redis = redis
	s.App = app
}

func (s *Suite) TearDownSuite() {
	if s.App != nil {
		_ = s.App.Close()
	}
	if s.Postgres != nil {
		_ = s.Postgres.Close()
	}
	if s.Redis != nil {
		_ = s.Redis.Close()
	}
}

func (s *Suite) SetupTest() {
	err := s.cleanupDatabase()
	if err != nil {
		s.T().Fatalf("Failed to cleanup database: %v", err)
	}

	ctx := context.Background()
	if err := s.Redis.Client.FlushDB(ctx).Err(); err != nil {
		s.T().Fatalf("Failed to flush Redis: %v", err)
	}
}

func (s *Suite) cleanupDatabase() error {
	return s.executeSQLFile(s.Postgres.DB, filepath.Join("testdata", "cleanup.sql"))
}

func (s *Suite) setupDatabase(db *sql.DB) error {
	return s.executeSQLFile(db, filepath.Join("testdata", "setup.sql"))
}

func (s *Suite) executeSQLFile(db *sql.DB, filePath string) error {
	sqlBytes, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read %s: %w", filePath, err)
	}

	if _, err := db.Exec(string(sqlBytes)); err != nil {
		return fmt.Errorf("failed to execute %s: %w", filePath, err)
	}

	return nil
}
