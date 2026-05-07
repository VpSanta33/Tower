package logic

import (
	"context"
	"encoding/json"
	"time"

	"tower/api/internal/svc"

	"github.com/redis/go-redis/v9"
	"github.com/zeromicro/go-zero/core/logx"
	"go.mongodb.org/mongo-driver/bson"
)

// RecoveredTaskInfo 恢复的任务信息
type RecoveredTaskInfo struct {
	TaskId      string `json:"taskId"`
	MainTaskId  string `json:"mainTaskId"`
	WorkspaceId string `json:"workspaceId"`
	Status      string `json:"status"`
	StartTime   string `json:"startTime"`
}

// RecoverOrphanedTasks 查找并恢复卡住的任务
func RecoverOrphanedTasks(ctx context.Context, svcCtx *svc.ServiceContext, timeout time.Duration) ([]RecoveredTaskInfo, error) {
	workspaces, err := svcCtx.WorkspaceModel.FindAll(ctx)
	if err != nil {
		logx.Errorf("[OrphanedTaskRecovery] Failed to get workspaces: %v", err)
		return nil, err
	}

	cutoffTime := time.Now().Add(-timeout)
	var recoveredTasks []RecoveredTaskInfo

	for _, ws := range workspaces {
		taskModel := svcCtx.GetMainTaskModel(ws.Name)

		filter := bson.M{
			"status": "STARTED",
			"update_time": bson.M{
				"$lt": cutoffTime,
			},
		}

		tasks, err := taskModel.Find(ctx, filter, 0, 50)
		if err != nil {
			logx.Errorf("[OrphanedTaskRecovery] Failed to find tasks for workspace %s: %v", ws.Name, err)
			continue
		}

		for _, task := range tasks {
			update := bson.M{
				"status":      "PENDING",
				"update_time": time.Now(),
			}

			if err := taskModel.UpdateByTaskId(ctx, task.TaskId, update); err != nil {
				logx.Errorf("[OrphanedTaskRecovery] Failed to update task %s: %v", task.TaskId, err)
				continue
			}

			taskInfo := map[string]interface{}{
				"taskId":      task.TaskId,
				"mainTaskId":  task.TaskId,
				"workspaceId": ws.Name,
				"taskName":    task.Name,
				"config":      task.Config,
				"priority":    5,
				"createTime":  time.Now().Format("2006-01-02 15:04:05"),
			}

			taskData, _ := json.Marshal(taskInfo)
			score := float64(time.Now().Unix()) - 5000

			publicQueueKey := "tower:task:queue"
			if err := svcCtx.RedisClient.ZAdd(ctx, publicQueueKey, redis.Z{
				Score:  score,
				Member: taskData,
			}).Err(); err != nil {
				logx.Errorf("[OrphanedTaskRecovery] Failed to requeue task %s: %v", task.TaskId, err)
				continue
			}

			startTimeStr := ""
			if task.StartTime != nil {
				startTimeStr = task.StartTime.Format("2006-01-02 15:04:05")
			}

			recoveredTasks = append(recoveredTasks, RecoveredTaskInfo{
				TaskId:      task.TaskId,
				MainTaskId:  task.TaskId,
				WorkspaceId: ws.Name,
				Status:      task.Status,
				StartTime:   startTimeStr,
			})

			logx.Infof("[OrphanedTaskRecovery] Recovered task %s for workspace %s", task.TaskId, ws.Name)
		}
	}

	return recoveredTasks, nil
}

// CleanupStaleProcessingTasks 清理过期的处理中任务记录
func CleanupStaleProcessingTasks(ctx context.Context, svcCtx *svc.ServiceContext, workerName string) {
	processingKey := "tower:task:processing"
	taskIds, err := svcCtx.RedisClient.SMembers(ctx, processingKey).Result()
	if err != nil {
		return
	}

	cleaned := 0
	for _, taskId := range taskIds {
		statusKey := "tower:task:status:" + taskId
		statusData, err := svcCtx.RedisClient.Get(ctx, statusKey).Result()
		if err != nil {
			// 状态不存在，直接清理
			svcCtx.RedisClient.SRem(ctx, processingKey, taskId)
			cleaned++
			continue
		}

		if workerName != "" {
			var status map[string]interface{}
			if err := json.Unmarshal([]byte(statusData), &status); err == nil {
				if worker, ok := status["worker"].(string); ok && worker == workerName {
					svcCtx.RedisClient.SRem(ctx, processingKey, taskId)
					svcCtx.RedisClient.Del(ctx, statusKey)
					cleaned++
				}
			}
		}
	}

	if cleaned > 0 {
		logx.Infof("[OrphanedTaskRecovery] Cleaned up %d stale processing records", cleaned)
	}
}
