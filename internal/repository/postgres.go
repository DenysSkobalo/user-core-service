package repository

import (
	"context"
	"database/sql"
	"fmt"
	"hash/crc32"
	"time"
	"gostream-hub/internal/config"
	"gostream-hub/internal/domain"

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

func (sm *ShardManager) Delete(ctx context.Context, id string) error {
	shardID := sm.getShardID(id)
	db, ok := sm.shards[shardID]
	if !ok {
		return fmt.Errorf("[Shard Manager] shard %d not found for user %s", shardID, id)
	}

	dbCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	query := "DELETE FROM users WHERE id = $1"
	_, err := db.ExecContext(dbCtx, query, id)
	if err != nil {
		return fmt.Errorf("[Shard Manager] failed to execute delete on shard %d: %w", shardID, err)
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
	dbCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	resultCh := make(chan *domain.User, len(sm.shards))
	errCh := make(chan error, len(sm.shards))

	for shardID, db := range sm.shards {
		go func(db *sql.DB, sID int) {
			query := "SELECT id, email, password_hash FROM users WHERE email = $1"
			var u domain.User

		
			err := db.QueryRowContext(dbCtx, query, email).Scan(&u.ID, &u.Email, &u.PasswordHash)
			if err == nil {
				resultCh <- &u
				return 
			}

			errCh <- err
		}(db, shardID)
	}

	errCount := 0
	for {
		select {
		case user := <-resultCh:
			return user, nil
		case <-errCh:
			errCount++
			if errCount == len(sm.shards) {
				return nil, nil
			}

		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
}

func (sm *ShardManager) HealthCheckShards(healthCtx context.Context) error {
	for shardID, db := range sm.shards {
		if err := db.PingContext(healthCtx); err != nil {
			return fmt.Errorf("ShardID %d connection failed: %w", shardID, err)
		}
	}
	return nil
}
