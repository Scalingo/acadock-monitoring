package filters

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/Scalingo/go-utils/logger"
)

var ErrNotEnoughMetrics = fmt.Errorf("no enough metrics yet")

type MetricsReader interface {
	Read(ctx context.Context) (float64, error)
}

type ExponentialSmoothingOpts = func(ExponentialSmoothing) ExponentialSmoothing

type ExponentialSmoothing struct {
	reader          MetricsReader
	averageLength   int
	averageInterval time.Duration
	queueLength     int
	stopped         bool
	stopMutex       *sync.Mutex
	queueMutex      *sync.Mutex
	queue           []float64
	lastSampleTime  time.Time
	lastAverageTime time.Time
}

func WithAverageConfig(sampleCount int, averageInterval time.Duration) ExponentialSmoothingOpts {
	return func(res ExponentialSmoothing) ExponentialSmoothing {
		res.averageLength = sampleCount
		res.averageInterval = averageInterval
		return res
	}
}

func WithQueueLength(length int) ExponentialSmoothingOpts {
	return func(res ExponentialSmoothing) ExponentialSmoothing {
		res.queueLength = length
		return res
	}
}

func NewExponentialSmoothing(from MetricsReader, opts ...ExponentialSmoothingOpts) (*ExponentialSmoothing, error) {
	result := ExponentialSmoothing{
		reader:          from,
		averageLength:   5,
		averageInterval: 10 * time.Second,
		queueLength:     6,
		stopped:         false,
		stopMutex:       &sync.Mutex{},
		queueMutex:      &sync.Mutex{},
		queue:           make([]float64, 0),
		lastSampleTime:  time.Now(),
		lastAverageTime: time.Now(),
	}

	for _, opt := range opts {
		result = opt(result)
	}
	if result.queueLength <= 0 {
		return nil, fmt.Errorf("QueueLength should be >0, current value: %v", result.queueLength)
	}

	if result.averageLength <= 0 {
		return nil, fmt.Errorf("averageLength should be >0, current value: %v", result.averageLength)
	}

	if result.averageInterval <= 1*time.Millisecond {
		return nil, fmt.Errorf("averageInterval should be > 1ms, current values: %s", result.averageInterval.String())

	}

	return &result, nil
}

func (e *ExponentialSmoothing) Start(ctx context.Context) {
	log := logger.Get(ctx)
	values := make([]float64, 0, e.averageLength)
	for {
		// Wait for next slot
		e.waitForNextSample()
		if e.isStopped() {
			return
		}

		// Read value
		value, err := e.reader.Read(ctx)
		if err != nil {
			log.WithError(err).Error("fail to fetch metrics")
			continue
		}

		values = append(values, value)

		// Compute the next average if needed
		if e.lastAverageTime.Add(e.averageInterval).Before(time.Now()) {
			var result float64
			for _, v := range values {
				result += v
			}
			e.appendToQueue(result / float64(len(values)))
			values = make([]float64, 0, e.averageLength)
			e.lastAverageTime = time.Now()
		}
	}
}

func (e *ExponentialSmoothing) Read(ctx context.Context) (float64, error) {
	e.queueMutex.Lock()
	values := make([]float64, len(e.queue))
	copy(values, e.queue)
	e.queueMutex.Unlock()
	if len(values) < e.queueLength {
		return 0.0, ErrNotEnoughMetrics
	}
	fmt.Printf("%+v", values)

	alpha := math.Exp(float64(-1 * len(values)))

	return exponentialSmoothing(values, len(values)-1, alpha), nil
}

func exponentialSmoothing(values []float64, ptr int, alpha float64) float64 {
	if ptr <= 0 {
		return values[0]
	}

	return alpha*values[ptr] + ((1.0 - alpha) * exponentialSmoothing(values, ptr-1, alpha))
}

func (e *ExponentialSmoothing) waitForNextSample() {
	timeBetweenSamples := e.averageInterval / time.Duration(e.averageLength)
	timeToNextSample := e.lastSampleTime.Add(timeBetweenSamples)
	timeToWait := timeToNextSample.Sub(time.Now())

	if timeToWait > 0 {
		time.Sleep(timeToWait)
	}

	e.lastSampleTime = time.Now()
}

func (e *ExponentialSmoothing) appendToQueue(value float64) {
	e.queueMutex.Lock()
	defer e.queueMutex.Unlock()

	e.queue = append(e.queue, value)
	if len(e.queue) > e.queueLength {
		e.queue = e.queue[len(e.queue)-e.queueLength:]
	}
}

func (e ExponentialSmoothing) isStopped() bool {
	e.stopMutex.Lock()
	defer e.stopMutex.Unlock()
	return e.stopped
}

func (e *ExponentialSmoothing) Stop() {
	e.stopMutex.Lock()
	defer e.stopMutex.Unlock()
	e.stopped = true
}
