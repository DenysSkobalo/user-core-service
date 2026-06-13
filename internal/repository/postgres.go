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
	db, ok := sm.shards[shardID]
	if !ok {
		return fmt.Errorf("[Shard Manager] shard %d not found for user %s", shardID, user.ID)
	}

	query := "INSERT INTO users (id, email, password_hash) VALUES ($1, $2, $3)"
	
	_, err := db.ExecContext(ctx, query, user.ID, user.Email, user.PasswordHash)
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

	query := "SELECT id, email, password_hash FROM users WHERE id = $1"
	
	var u domain.User

	err := db.QueryRowContext(ctx, query, id).Scan(&u.ID, &u.Email, &u.PasswordHash)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil 
		}
		return nil, fmt.Errorf("[Shard Manager] failed to scan user from shard %d: %w", shardID, err)
	}

	return &u, nil
}

func (sm *ShardManager) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	fmt.Printf("[Postgres Shard] GetByEmail call for %s (Requires lookup index)\n", email)
	return &domain.User{}, nil 
}

