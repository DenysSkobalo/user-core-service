package repository

import (
	"context"
	"fmt"
	"log/slog"
	"time"
	"user-core-service/internal/domain"
)

type UserRepositoryImpl struct {
	shardMgr *ShardManager
	cacheMgr *CacheManager
}

func NewUserRepository(sm *ShardManager, cm *CacheManager) UserRepository {
	return &UserRepositoryImpl{
		shardMgr: sm,
		cacheMgr: cm,
	}
}

func (r *UserRepositoryImpl) Save(ctx context.Context, user *domain.User) error {
	err := r.shardMgr.Save(ctx, user)
	if err != nil {
		return fmt.Errorf("[Repository] failed to save user to shard: %w", err)
	}

	err = r.cacheMgr.SetEmailIndex(ctx, user.Email, user.ID)
	if err != nil {
		slog.ErrorContext(
			ctx, "dual-write critical mismatch: redis index creation failed after postgres save. Initiating rollback...",
			"user_id", user.ID,
			"email", user.Email,
			"error", err,
		)

		return fmt.Errorf("[Repository] database transaction aborted due to global index failure: %w", err)
	}

	return nil
}

func (r *UserRepositoryImpl) GetByID(ctx context.Context, id string) (*domain.User, error) {
	return r.shardMgr.GetByID(ctx, id)
}

func (r *UserRepositoryImpl) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	id, err := r.cacheMgr.GetIDByEmail(ctx, email)
	if err != nil {
		slog.WarnContext(ctx, "redis lookup failed, switching to db shards fallback scan", "email", email, "error", err)
		return r.shardMgr.GetByEmailFallback(ctx, email)
	}

	if id == "" {
		slog.InfoContext(ctx, "redis cache miss for email index, executing database fallback scan", "email", email)

		user, err := r.shardMgr.GetByEmailFallback(ctx, email)
		if err != nil {
			return nil, err
		}

		if user == nil {
			return nil, nil
		}

		go func(u *domain.User) {
			lazyCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			if cacheErr := r.cacheMgr.SetEmailIndex(lazyCtx, u.Email, u.ID); cacheErr != nil {
				slog.Error("failed to restore evicted redis index during lazy recovery", "email", u.Email, "error", cacheErr)
			} else {
				slog.Info("redis index successfully restored via lazy recovery", "email", u.Email)
			}
		}(user)

		return user, nil
	}

	return r.shardMgr.GetByID(ctx, id)
}

func (r *UserRepositoryImpl) PingDB(ctx context.Context) error {
	return r.shardMgr.HealthCheckShards(ctx)
}

func (r *UserRepositoryImpl) PingCache(ctx context.Context) error {
	return r.cacheMgr.HealthCheckCache(ctx)
}
