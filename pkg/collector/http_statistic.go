package collector

import (
	"fmt"
	"sync"
	"time"
)

type HttpStatCollector struct {
	GlobalStat     GlobalStatistic
	ResponseStatus map[string]int // Status Codes and corresponded count
	StatChannel    chan *HttpEntry
}

func CreateHttpStatCollector() *HttpStatCollector {
	statistic := &HttpStatCollector{
		StatChannel:    make(chan *HttpEntry),
		ResponseStatus: make(map[string]int),
	}
	return statistic
}

func (h *HttpStatCollector) GetGlobalStats() *GlobalStatistic {
	return &h.GlobalStat
}

// prints stats on every +10% process
func (h *HttpStatCollector) PrintProgressStats() {
	fmt.Printf("HTTP Codes:\n 1xx:%d 2xx:%d, 3xx:%d, 4xx:%d, 5xx:%d, other:%d\nCurrent Total Request: %d, Current Total Time: %s, Avg Duration %s\n",
		h.ResponseStatus["1xx"], h.ResponseStatus["2xx"], h.ResponseStatus["3xx"], h.ResponseStatus["4xx"], h.ResponseStatus["5xx"], h.ResponseStatus["other"],
		h.GlobalStat.TotalRequest, h.GlobalStat.TotalDuration, h.GlobalStat.AverageDuration)
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
			if httpEntry.ResponseCode < 200 {
				h.ResponseStatus["1xx"]++
				h.GlobalStat.SuccessfulReq++
			} else if httpEntry.ResponseCode < 300 {
				h.ResponseStatus["2xx"]++
				h.GlobalStat.SuccessfulReq++
			} else if httpEntry.ResponseCode < 400 {
				h.ResponseStatus["3xx"]++
				h.GlobalStat.SuccessfulReq++
			} else if httpEntry.ResponseCode < 500 {
				h.ResponseStatus["4xx"]++
				h.GlobalStat.FailedReq++
			} else if httpEntry.ResponseCode < 600 {
				h.ResponseStatus["5xx"]++
				h.GlobalStat.FailedReq++
			} else {
				h.ResponseStatus["other"]++
				h.GlobalStat.FailedReq++
			}
			avg_time += httpEntry.Duration
			h.GlobalStat.TotalSize = httpEntry.ReadSize + httpEntry.WriteSize
		}
	}
	end := time.Since(start)
	h.GlobalStat.TotalDuration = end
	size_in_mb := float64(h.GlobalStat.TotalSize) / (1 << 20) //For MB
	h.GlobalStat.Throughput = size_in_mb / h.GlobalStat.TotalDuration.Seconds()
	avg_time = time.Duration(int64(avg_time) / count)
	h.GlobalStat.AverageDuration = avg_time

}

func (h *HttpStatCollector) Finished() {
	close(h.StatChannel)
}
