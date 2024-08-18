package worker

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/hibiken/asynq"
	db "github.com/jaingounchained/todo/db/sqlc"
	"github.com/jaingounchained/todo/util"
	"github.com/rs/zerolog/log"
)

type PeriodicTaskScheduler interface {
	Start() error
	ScheduleTodoNotification(todo db.Todo) error
	RemoveTodoNotification(todo db.Todo) error
	// TODO: Implement update todo notification
}

type TodoConfigMap map[PayloadSendTodoNotification]time.Duration

type RedisPeriodicTaskScheduler struct {
	periodicTaskManager *asynq.PeriodicTaskManager
	todoConfigMap       TodoConfigMap
	mu                  sync.Mutex
}

func NewRedisPeriodicTaskScheduler(redisOpt asynq.RedisClientOpt) (PeriodicTaskScheduler, error) {
	todoConfigMap := TodoConfigMap(make(map[PayloadSendTodoNotification]time.Duration))

	periodicTaskManager, err := asynq.NewPeriodicTaskManager(
		asynq.PeriodicTaskManagerOpts{
			RedisConnOpt:               redisOpt,
			PeriodicTaskConfigProvider: todoConfigMap,
			SyncInterval:               5 * time.Second,
			SchedulerOpts: &asynq.SchedulerOpts{
				Logger: NewLogger(),
			},
		})
	if err != nil {
		return nil, fmt.Errorf("failed to start redis periodic task manager: %w", err)
	}

	return &RedisPeriodicTaskScheduler{
		periodicTaskManager: periodicTaskManager,
	}, nil
}

func (scheduler *RedisPeriodicTaskScheduler) Start() error {
	if err := scheduler.periodicTaskManager.Run(); err != nil {
		log.Fatal().Err(err).Msg("cannot start periodic task scheduler")
	}
	return nil
}

func (scheduler *RedisPeriodicTaskScheduler) RemoveTodoNotification(todo db.Todo) error {
	payload := convertTodoToPayload(todo)

	scheduler.mu.Lock()
	defer scheduler.mu.Unlock()

	delete(scheduler.todoConfigMap, payload)

	return nil
}

func (scheduler *RedisPeriodicTaskScheduler) ScheduleTodoNotification(todo db.Todo) error {
	payload := convertTodoToPayload(todo)

	scheduler.mu.Lock()
	defer scheduler.mu.Unlock()

	scheduler.todoConfigMap[payload] = time.Duration(*todo.PeriodicReminderTimeSeconds)

	return nil
}

func (m TodoConfigMap) GetConfigs() ([]*asynq.PeriodicTaskConfig, error) {
	config := make([]*asynq.PeriodicTaskConfig, 0)

	for payload, duration := range m {
		cronSpec, err := util.ConvertDurationToCron(duration)
		if err != nil {
			return nil, fmt.Errorf("Failed to create cronspec: %w", err)
		}

		jsonPayload, err := json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("Failed to marshal payload: %w", err)
		}

		config = append(config, &asynq.PeriodicTaskConfig{
			Cronspec: cronSpec,
			Task: asynq.NewTask(
				TaskSendTodoNotification,
				jsonPayload,
			),
		})
	}

	return config, nil
}

func convertTodoToPayload(todo db.Todo) PayloadSendTodoNotification {
	return PayloadSendTodoNotification{
		Username: todo.Owner,
		TodoID:   todo.ID,
	}
}
