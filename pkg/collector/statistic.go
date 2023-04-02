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
}

type HttpStatCollector struct {
	GlobalStat     GlobalStatistic
	ResponseStatus map[int]int // Status Codes and corresponded count
	StatChannel    chan *HttpEntry
}

func CreateHttpStatCollector() *HttpStatCollector {
	statistic := &HttpStatCollector{
		StatChannel:    make(chan *HttpEntry),
		ResponseStatus: make(map[int]int),
	}
	return statistic
}

func (h *HttpStatCollector) GetGlobalStats() *GlobalStatistic {
	return &h.GlobalStat
}

func (h *HttpStatCollector) Consume(wg *sync.WaitGroup) {
	defer wg.Done()
	start := time.Now()
	var avg_time time.Duration
	var count int64
loop:
	for {
		select {
		case httpEntry, ok := <-h.StatChannel:
			if !ok {
				break loop
			}
			count++
			h.GlobalStat.TotalRequest++
			if httpEntry.ResponseCode < 400 {
				h.GlobalStat.SuccessfulReq++
			} else {
				h.GlobalStat.FailedReq++
			}
			h.ResponseStatus[httpEntry.ResponseCode]++
			avg_time += httpEntry.Duration
			h.GlobalStat.TotalSize = httpEntry.ReadSize + httpEntry.WriteSize
		}
	}
	end := time.Since(start)
	h.GlobalStat.TotalDuration = end
	h.GlobalStat.Throughput = float64(h.GlobalStat.TotalSize) / (1 << 20) //For MB
	avg_time = time.Duration(int64(avg_time) / count)
	h.GlobalStat.AverageDuration = avg_time

}

func (h *HttpStatCollector) Finished() {
	close(h.StatChannel)
}
