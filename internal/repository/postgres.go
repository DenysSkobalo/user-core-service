package repository

import (
	"context"
	"database/sql"
	"fmt"
	"hash/crc32"
	"time"
	"user-core-service/internal/config"
	"user-core-service/internal/domain"

	_ "github.com/lib/pq"
)

type ShardManager struct {
	shards map[int]*sql.DB
}

func NewShardManager(c *config.Config) (*ShardManager, error) {
	dsnConfig := map[int]string{
		1: c.Shard1DSN,
		2: c.Shard2DSN,
	}

	shardsPool := make(map[int]*sql.DB)

	for shardID, dsn := range dsnConfig {
		db, err := sql.Open("postgres", dsn)
		if err != nil {
			return nil, fmt.Errorf("failed to open shard %d: %w", shardID, err)
		}

		db.SetMaxOpenConns(100)
		db.SetMaxIdleConns(25)
		db.SetConnMaxIdleTime(5 * time.Minute)

		if err := db.Ping(); err != nil {
			return nil, fmt.Errorf("failed to ping shard %d: %w", shardID, err)
		}

		shardsPool[shardID] = db
	}

	return &ShardManager{
		shards: shardsPool,
	}, nil
}

func (sm *ShardManager) Close() error {
	for sharID, db := range sm.shards {
		if err := db.Close(); err != nil {
			return fmt.Errorf("failed to close shard %d: %w", sharID, err)
		}
	}
	return nil
}

func (sm *ShardManager) getShardID(id string) int {
	checkSum := crc32.ChecksumIEEE([]byte(id))
	return int(checkSum%2) + 1
}

func (sm *ShardManager) Save(ctx context.Context, user *domain.User) error {
	shardID := sm.getShardID(user.ID)
	db, ok := sm.shards[shardID]
	if !ok {
		return fmt.Errorf("[Shard Manager] shard %d not found for user %s", shardID, user.ID)
	}

	dbCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	query := "INSERT INTO users (id, email, password_hash) VALUES ($1, $2, $3)"

	_, err := db.ExecContext(dbCtx, query, user.ID, user.Email, user.PasswordHash)
	if err != nil {
		return fmt.Errorf("[Shard Manager] failed to execute insert on shard %d: %w", shardID, err)
	}

	return nil
}

func (sm *ShardManager) GetByID(ctx context.Context, id string) (*domain.User, error) {
	shardID := sm.getShardID(id)
	db, ok := sm.shards[shardID]
	if !ok {
		return nil, fmt.Errorf("[Shard Manager] shard %d not found for id %s", shardID, id)
	}

	dbCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	query := "SELECT id, email, password_hash FROM users WHERE id = $1"

	var u domain.User

	err := db.QueryRowContext(dbCtx, query, id).Scan(&u.ID, &u.Email, &u.PasswordHash)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("[Shard Manager] failed to scan user from shard %d: %w", shardID, err)
	}

	return &u, nil
}

func (sm *ShardManager) GetByEmailFallback(ctx context.Context, email string) (*domain.User, error) {
	for shardID, db := range sm.shards {
		query := "SELECT id, email, password_hash FROM users WHERE email = $1"
		var u domain.User

		dbCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
		defer cancel()

		err := db.QueryRowContext(dbCtx, query, email).Scan(&u.ID, &u.Email, &u.PasswordHash)
		if err == nil {
			return &u, nil
		}

		if err == sql.ErrNoRows {
			continue
		}

		return nil, fmt.Errorf("fallback scan failed on shard %d: %w", shardID, err)
	}

	return nil, nil
}

func (sm *ShardManager) HealthCheckShards(healthCtx context.Context) error {
	for shardID, db := range sm.shards {
		if err := db.PingContext(healthCtx); err != nil {
			return fmt.Errorf("ShardID %d connection failed: %w", shardID, err)
		}
	}
	return nil
}
