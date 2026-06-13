package repository

import (
	"context"
	"database/sql"
	"fmt"
	"hash/crc32"
	"time"
	"user-core-service/internal/domain"

	_ "github.com/lib/pq"
)

type ShardManager struct {
	shards map[int]*sql.DB
}

func NewShardManager() (*ShardManager, error) {
	dsnConfig := map[int]string{
		1: "postgres://postgres:postgres@localhost:5431/user_core_shard_1?sslmode=disable",
		2: "postgres://postgres:postgres@localhost:5432/user_core_shard_2?sslmode=disable",
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
	return int(checkSum%2)+1
}


func (sm *ShardManager) Save(ctx context.Context, user *domain.User) error {
	shardID := sm.getShardID(user.ID)
	fmt.Printf("[Postgres Shard] Saving user %s to Shard %d\n", user.ID, shardID)
	return nil
}

func (sm *ShardManager) GetByID(ctx context.Context, id string) (*domain.User, error) {
	shardID := sm.getShardID(id)
	fmt.Printf("[Postgres Shard] Fetching user by ID %s from Shard %d\n", id, shardID)
	return &domain.User{}, nil
}

func (sm *ShardManager) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	fmt.Printf("[Postgres Shard] GetByEmail call for %s (Requires lookup index)\n", email)
	return &domain.User{}, nil 
}

