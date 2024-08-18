package worker

import (
	"os"
	"testing"

	"github.com/hibiken/asynq"
	"github.com/jaingounchained/todo/util"
	"github.com/rs/zerolog/log"
)

var testPeriodicTaskScheduler PeriodicTaskScheduler
var testTaskDistributor TaskDistributor
var testTaskProcessor TaskProcessor

func TestMain(m *testing.M) {
	config, err := util.LoadConfig("..")
	if err != nil {
		log.Fatal().Err(err).Msg("cannot load config")
	}

	redisOpt := asynq.RedisClientOpt{
		Addr: config.RedisAddress,
	}

	testPeriodicTaskScheduler, err = NewRedisPeriodicTaskScheduler(redisOpt)
	if err != nil {
		log.Fatal().Err(err).Msg("cannot create redis periodic task scheduler")
	}

	code := m.Run()
	m.Run()
	os.Exit(code)
}
