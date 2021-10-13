package workqueue

import (
	"time"

	"github.com/symcn/api"
	"golang.org/x/time/rate"
	"k8s.io/client-go/util/workqueue"
)

// ratelimit queue set
var (
	defaultQueueName             = "symcn-queue"
	defaultGotInterval           = time.Second * 1
	defaultRateLimitTimeInterval = time.Second * 1
	defaultRateLimitTimeMax      = time.Second * 60
	defaultRateLimit             = 10
	defaultRateBurst             = 100
	defaultThreadiness           = 1
)

type QueueConfig struct {
	Name                  string
	GotInterval           time.Duration
	RateLimitTimeInterval time.Duration
	RateLimitTimeMax      time.Duration
	RateLimit             int
	RateBurst             int
	Threadiness           int
	Do                    api.Reconciler
}

type compltedConfig struct {
	*QueueConfig
}

// CompletedConfig wrapper workqueue
type CompletedConfig struct {
	*compltedConfig
}

type queue struct {
	*CompletedConfig
	Workqueue workqueue.RateLimitingInterface
	Stats     *stats
}

func NewQueueConfig(reconcile api.Reconciler) *QueueConfig {
	qc := &QueueConfig{
		Name:                  defaultQueueName,
		GotInterval:           defaultGotInterval,
		RateLimitTimeInterval: defaultRateLimitTimeInterval,
		RateLimitTimeMax:      defaultRateLimitTimeMax,
		RateLimit:             defaultRateLimit,
		RateBurst:             defaultRateBurst,
		Threadiness:           defaultThreadiness,
		Do:                    reconcile,
	}

	return qc
}

func Complted(qc *QueueConfig) *CompletedConfig {
	cc := &CompletedConfig{&compltedConfig{qc}}

	if cc.GotInterval < defaultGotInterval {
		cc.GotInterval = defaultGotInterval
	}

	if cc.Threadiness < 1 {
		cc.Threadiness = defaultThreadiness
	}

	return cc
}

// NewQueue build queue
func (cc *CompletedConfig) NewQueue() (api.WorkQueue, error) {
	stats, err := buildStats(cc.Name)
	if err != nil {
		return nil, err
	}

	q := &queue{
		CompletedConfig: cc,
		Stats:           stats,
		Workqueue: workqueue.NewNamedRateLimitingQueue(
			workqueue.NewMaxOfRateLimiter(
				workqueue.NewItemExponentialFailureRateLimiter(cc.RateLimitTimeInterval, cc.RateLimitTimeMax),
				&workqueue.BucketRateLimiter{Limiter: rate.NewLimiter(rate.Limit(float64(cc.RateLimit)), cc.RateBurst)},
			),
			cc.Name,
		),
	}

	return q, nil
}
