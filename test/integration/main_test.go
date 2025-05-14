package integration

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
	"github.com/redis/go-redis/v9"
)

var (
	db            *sql.DB
	redisClient   *redis.Client
	testUserID    uuid.UUID
	testChatID    uuid.UUID
	dbName        string
	pool          *dockertest.Pool
	pgResource    *dockertest.Resource
	redisResource *dockertest.Resource
)

func TestMain(m *testing.M) {
	// Setup
	var err error

	// Create a unique database name for this test run
	dbName = fmt.Sprintf("rtcs_test_%d", time.Now().UnixNano())

	// Connect to Docker
	pool, err = dockertest.NewPool("")
	if err != nil {
		log.Fatalf("Could not connect to Docker: %s", err)
	}

	// Set up PostgreSQL container
	pgResource, err = setupPostgres()
	if err != nil {
		log.Fatalf("Could not start PostgreSQL: %s", err)
	}

	// Set up Redis container
	redisResource, err = setupRedis()
	if err != nil {
		log.Fatalf("Could not start Redis: %s", err)
	}

	// Run tests
	code := m.Run()

	// Teardown
	if err = pool.Purge(pgResource); err != nil {
		log.Fatalf("Could not purge PostgreSQL: %s", err)
	}
	if err = pool.Purge(redisResource); err != nil {
		log.Fatalf("Could not purge Redis: %s", err)
	}

	os.Exit(code)
}

func setupPostgres() (*dockertest.Resource, error) {
	// Start PostgreSQL container
	pgResource, err := pool.RunWithOptions(&dockertest.RunOptions{
		Repository: "postgres",
		Tag:        "15",
		Env: []string{
			"POSTGRES_PASSWORD=postgres",
			"POSTGRES_USER=postgres",
			fmt.Sprintf("POSTGRES_DB=%s", dbName),
		},
	}, func(config *docker.HostConfig) {
		// Set AutoRemove to true so that stopped container goes away by itself
		config.AutoRemove = true
		config.RestartPolicy = docker.RestartPolicy{
			Name: "no",
		}
	})

	if err != nil {
		return nil, err
	}

	// Exponential backoff for container startup
	if err := pool.Retry(func() error {
		var err error
		db, err = sql.Open("postgres", fmt.Sprintf("postgres://postgres:postgres@localhost:%s/%s?sslmode=disable", pgResource.GetPort("5432/tcp"), dbName))
		if err != nil {
			return err
		}
		return db.Ping()
	}); err != nil {
		return nil, err
	}

	// Apply migrations
	if err := applyMigrations(); err != nil {
		return nil, err
	}

	// Create test data
	if err := createTestData(); err != nil {
		return nil, err
	}

	return pgResource, nil
}

func setupRedis() (*dockertest.Resource, error) {
	// Start Redis container
	redisResource, err := pool.RunWithOptions(&dockertest.RunOptions{
		Repository: "redis",
		Tag:        "7",
	}, func(config *docker.HostConfig) {
		// Set AutoRemove to true so that stopped container goes away by itself
		config.AutoRemove = true
		config.RestartPolicy = docker.RestartPolicy{
			Name: "no",
		}
	})

	if err != nil {
		return nil, err
	}

	// Exponential backoff for container startup
	if err := pool.Retry(func() error {
		var err error
		redisClient = redis.NewClient(&redis.Options{
			Addr: fmt.Sprintf("localhost:%s", redisResource.GetPort("6379/tcp")),
		})
		_, err = redisClient.Ping(context.Background()).Result()
		return err
	}); err != nil {
		return nil, err
	}

	return redisResource, nil
}

func applyMigrations() error {
	// Read migrations from files
	migrations := []string{
		// Schema migration (create tables)
		`
		CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
		
		CREATE TABLE IF NOT EXISTS users (
			id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
			created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
			deleted_at TIMESTAMP WITH TIME ZONE,
			username VARCHAR(255) NOT NULL,
			password VARCHAR(255) NOT NULL,
			display_name VARCHAR(255),
			avatar_url VARCHAR(512),
			about TEXT,
			email VARCHAR(255),
			name VARCHAR(255),
			picture VARCHAR(255),
			auth_type VARCHAR(50) DEFAULT 'local',
			CONSTRAINT uni_users_username UNIQUE (username)
		);
		
		CREATE TABLE IF NOT EXISTS chats (
			id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
			name VARCHAR(255) NOT NULL,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
			deleted_at TIMESTAMP WITH TIME ZONE
		);
		
		CREATE TABLE IF NOT EXISTS chat_users (
			chat_id UUID REFERENCES chats(id),
			user_id UUID REFERENCES users(id),
			joined_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (chat_id, user_id)
		);
		
		CREATE TABLE IF NOT EXISTS messages (
			id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
			chat_id UUID NOT NULL REFERENCES chats(id),
			sender_id UUID NOT NULL REFERENCES users(id),
			text TEXT NOT NULL,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
			deleted_at TIMESTAMP WITH TIME ZONE
		);
		
		CREATE INDEX IF NOT EXISTS idx_messages_chat_id ON messages(chat_id);
		CREATE INDEX IF NOT EXISTS idx_messages_sender_id ON messages(sender_id);
		CREATE INDEX IF NOT EXISTS idx_users_username ON users(username);
		CREATE INDEX IF NOT EXISTS idx_chat_users_user_id ON chat_users(user_id);
		CREATE INDEX IF NOT EXISTS idx_chat_users_chat_id ON chat_users(chat_id);
		`,
	}

	// Execute each migration
	for _, migration := range migrations {
		_, err := db.Exec(migration)
		if err != nil {
			return fmt.Errorf("error applying migration: %w", err)
		}
	}

	return nil
}

func createTestData() error {
	// Create test user
	testUserID = uuid.New()
	_, err := db.Exec(
		"INSERT INTO users (id, username, password, display_name) VALUES ($1, $2, $3, $4)",
		testUserID, "testuser", "$2a$10$Nv1jZGDOThHC8cECL4XOXelT8mtoYVpxUUO.zWQnbMOBY8TL5Zi.C", "Test User", // Password is "password"
	)
	if err != nil {
		return fmt.Errorf("error creating test user: %w", err)
	}

	// Create test chat
	testChatID = uuid.New()
	_, err = db.Exec(
		"INSERT INTO chats (id, name) VALUES ($1, $2)",
		testChatID, "Test Chat",
	)
	if err != nil {
		return fmt.Errorf("error creating test chat: %w", err)
	}

	// Add user to chat
	_, err = db.Exec(
		"INSERT INTO chat_users (chat_id, user_id) VALUES ($1, $2)",
		testChatID, testUserID,
	)
	if err != nil {
		return fmt.Errorf("error adding user to chat: %w", err)
	}

	return nil
}

// Helper function to cleanup test data
func cleanupTestData() error {
	_, err := db.Exec("DELETE FROM messages")
	if err != nil {
		return err
	}
	_, err = db.Exec("DELETE FROM chat_users")
	if err != nil {
		return err
	}
	_, err = db.Exec("DELETE FROM chats")
	if err != nil {
		return err
	}
	_, err = db.Exec("DELETE FROM users")
	if err != nil {
		return err
	}
	return nil
}
