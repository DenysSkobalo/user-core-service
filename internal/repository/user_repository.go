package repository

import (
	"context"
	"fmt"
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
		return fmt.Errorf("[Repository] database write succeeded, but redis index creation failed: %w", err)
	}

	return nil
}

func (r *UserRepositoryImpl) GetByID(ctx context.Context, id string) (*domain.User, error) {
	return r.shardMgr.GetByID(ctx, id)
}

func (r *UserRepositoryImpl) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	id, err := r.cacheMgr.GetIDByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("[Repository] redis email lookup failed: %w", err)
	}

	if id == "" {
		return nil, nil 
	}

	return r.shardMgr.GetByID(ctx, id)
}
