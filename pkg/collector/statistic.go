package collector

import (
	"sync"
	"time"
)

type StatEntry interface{}

type HttpEntry struct {
	ResponseCode int
	WriteSize    int64
	ReadSize     int64
	Duration     time.Duration
}

type SmtpEntry struct {
	ResponseCode int
	WriteSize    int64
	ReadSize     int64
	Duration     time.Duration
}

type GlobalStatistic struct {
	TotalRequest    int
	TotalDuration   time.Duration
	SuccessfulReq   int
	FailedReq       int
	AverageDuration time.Duration
	Throughput      float64
	TotalSize       int64
}

type StatBase interface {
	Consume(wg *sync.WaitGroup)
	Finished()
	GetGlobalStats() *GlobalStatistic
	PrintProgressStats()
}
