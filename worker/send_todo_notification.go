package worker

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hibiken/asynq"
	"github.com/rs/zerolog/log"
)

const TaskSendTodoNotification = "task:send_todo_notification"

type PayloadSendTodoNotification struct {
	Username string `json:"username"`
	TodoID   int64  `json:"todoId"`
}

func (processor *RedisTaskProcessor) ProcessTaskSendTodoNotification(ctx context.Context, task *asynq.Task) error {
	var payload PayloadSendTodoNotification
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		return fmt.Errorf("failed to unmarshal payload %w", asynq.SkipRetry)
	}

	// send notification
	if err := processor.notifier.SendNotification(); err != nil {
		return fmt.Errorf("failed to send notification: %w", err)
	}

	log.Info().
		Str("type", task.Type()).
		Bytes("payload", task.Payload()).
		Str("username", payload.Username).
		Int64("todoId", payload.TodoID).
		Msg("processed task")

	return nil
}
