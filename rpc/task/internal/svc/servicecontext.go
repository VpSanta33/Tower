package svc

import (
	"context"
	"fmt"
	"sync"
	"time"

	"tower/model"
	"tower/rpc/task/internal/config"
	"tower/scheduler"

	"github.com/redis/go-redis/v9"
	"github.com/zeromicro/go-zero/core/logx"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type ServiceContext struct {
	Config                  config.Config
	MongoClient             *mongo.Client
	MongoDB                 *mongo.Database
	RedisClient             *redis.Client
	NucleiTemplateModel     *model.NucleiTemplateModel
	FingerprintModel        *model.FingerprintModel
	CustomPocModel          *model.CustomPocModel
	HttpServiceMappingModel *model.HttpServiceMappingModel
	WorkspaceModel          *model.WorkspaceModel
	SubfinderProviderModel  *model.SubfinderProviderModel
	NotifyConfigModel       *model.NotifyConfigModel
	TaskRecoveryManager     *scheduler.TaskRecoveryManager // 任务恢复管理器

	// workspaceCache: workspaceId -> 是否存在；TTL 60s（修复 R1）
	workspaceCache   map[string]workspaceCacheEntry
	workspaceCacheMu sync.RWMutex
}

type workspaceCacheEntry struct {
	exists bool
	expire time.Time
}

func NewServiceContext(c config.Config) (*ServiceContext, error) {
	logx.Infof("Connecting to MongoDB: %s", c.Mongo.Uri)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	mongoClient, err := mongo.Connect(ctx, options.Client().ApplyURI(c.Mongo.Uri))
	if err != nil {
		return nil, fmt.Errorf("connect MongoDB: %w", err)
	}

	// 测试 MongoDB 连接
	if err := mongoClient.Ping(ctx, nil); err != nil {
		return nil, fmt.Errorf("ping MongoDB: %w", err)
	}
	logx.Info("MongoDB connected successfully")

	mongoDB := mongoClient.Database(c.Mongo.DbName)

	// 使用go-zero Redis配置
	logx.Infof("Connecting to Redis: %s", c.RedisConf.Host)
	rdb := redis.NewClient(&redis.Options{
		Addr:     c.RedisConf.Host,
		Password: c.RedisConf.Pass,
		DB:       0,
	})

	// 测试 Redis 连接
	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("ping Redis: %w", err)
	}
	logx.Info("Redis connected successfully")

	// 创建任务恢复管理器
	recoveryManager := scheduler.NewTaskRecoveryManager(rdb, context.Background())
	recoveryManager.Start()
	logx.Info("Task recovery manager started")

	return &ServiceContext{
		Config:                  c,
		MongoClient:             mongoClient,
		MongoDB:                 mongoDB,
		RedisClient:             rdb,
		NucleiTemplateModel:     model.NewNucleiTemplateModel(mongoDB),
		FingerprintModel:        model.NewFingerprintModel(mongoDB),
		CustomPocModel:          model.NewCustomPocModel(mongoDB),
		HttpServiceMappingModel: model.NewHttpServiceMappingModel(mongoDB),
		WorkspaceModel:          model.NewWorkspaceModel(mongoDB),
		SubfinderProviderModel:  model.NewSubfinderProviderModel(mongoDB),
		NotifyConfigModel:       model.NewNotifyConfigModel(mongoDB),
		TaskRecoveryManager:     recoveryManager,
		workspaceCache:          make(map[string]workspaceCacheEntry),
	}, nil
}

// IsValidWorkspace 校验 workspaceId 是否合法（修复 R1）
// 规则：空串或 "default" 视为合法（多租户默认空间）；其他 ID 必须在 workspace 集合中存在
// 带 60s 缓存，避免高频写入打爆 MongoDB
func (s *ServiceContext) IsValidWorkspace(ctx context.Context, id string) bool {
	if id == "" || id == "default" {
		return true
	}

	s.workspaceCacheMu.RLock()
	if e, ok := s.workspaceCache[id]; ok && time.Now().Before(e.expire) {
		s.workspaceCacheMu.RUnlock()
		return e.exists
	}
	s.workspaceCacheMu.RUnlock()

	exists := false
	if ws, err := s.WorkspaceModel.FindById(ctx, id); err == nil && ws != nil && !ws.Id.IsZero() {
		exists = true
	}

	s.workspaceCacheMu.Lock()
	s.workspaceCache[id] = workspaceCacheEntry{
		exists: exists,
		expire: time.Now().Add(60 * time.Second),
	}
	s.workspaceCacheMu.Unlock()
	return exists
}

func (s *ServiceContext) GetAssetModel(workspaceId string) *model.AssetModel {
	if workspaceId == "" {
		workspaceId = "default"
	}
	return model.NewAssetModel(s.MongoDB, workspaceId)
}

func (s *ServiceContext) GetMainTaskModel(workspaceId string) *model.MainTaskModel {
	if workspaceId == "" {
		workspaceId = "default"
	}
	return model.NewMainTaskModel(s.MongoDB, workspaceId)
}

func (s *ServiceContext) GetVulModel(workspaceId string) *model.VulModel {
	if workspaceId == "" {
		workspaceId = "default"
	}
	return model.NewVulModel(s.MongoDB, workspaceId)
}

func (s *ServiceContext) GetExecutorTaskModel(workspaceId string) *model.ExecutorTaskModel {
	if workspaceId == "" {
		workspaceId = "default"
	}
	return model.NewExecutorTaskModel(s.MongoDB, workspaceId)
}

func (s *ServiceContext) GetAssetHistoryModel(workspaceId string) *model.AssetHistoryModel {
	if workspaceId == "" {
		workspaceId = "default"
	}
	return model.NewAssetHistoryModel(s.MongoDB, workspaceId)
}
