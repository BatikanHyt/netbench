package collector

import (
	"fmt"
	"sync"
	"time"
)

type SmtpStatCollector struct {
	GlobalStat     GlobalStatistic
	StatChannel    chan *SmtpEntry
	ResponseStatus map[string]int
}

func CreateSmtpStatCollector() *SmtpStatCollector {
	statistic := &SmtpStatCollector{
		StatChannel:    make(chan *SmtpEntry),
		ResponseStatus: make(map[string]int),
	}
	return statistic
}

func (s *SmtpStatCollector) GetGlobalStats() *GlobalStatistic {
	return &s.GlobalStat
}

var slock = sync.RWMutex{}

func (s *SmtpStatCollector) PrintProgressStats() {

	slock.RLock()
	defer slock.RUnlock()
	fmt.Printf("SMTP Codes:\n 2xx:%d, 3xx:%d, 4xx:%d, 5xx:%d, other:%d\nCurrent Total Request: %d, Current Total Time: %s, Avg Duration %s\n",
		s.ResponseStatus["2xx"], s.ResponseStatus["3xx"], s.ResponseStatus["4xx"], s.ResponseStatus["5xx"], s.ResponseStatus["other"],
		s.GlobalStat.TotalRequest, s.GlobalStat.TotalDuration, s.GlobalStat.AverageDuration)
}

func (s *SmtpStatCollector) Consume(wg *sync.WaitGroup) {
	defer wg.Done()
	start := time.Now()
	var avg_time time.Duration
	var count int64
loop:
	for {
		select {
		case smtpEntry, ok := <-s.StatChannel:
			if !ok {
				break loop
			}
			count++
			s.GlobalStat.TotalRequest++
			slock.Lock()
			if smtpEntry.ResponseCode < 300 {
				s.ResponseStatus["2xx"]++
				s.GlobalStat.SuccessfulReq++
			} else if smtpEntry.ResponseCode < 400 {
				s.ResponseStatus["3xx"]++
				s.GlobalStat.SuccessfulReq++
			} else if smtpEntry.ResponseCode < 500 {
				s.ResponseStatus["4xx"]++
				s.GlobalStat.FailedReq++
			} else if smtpEntry.ResponseCode < 600 {
				s.ResponseStatus["5xx"]++
				s.GlobalStat.FailedReq++
			} else {
				s.ResponseStatus["other"]++
				s.GlobalStat.FailedReq++
			}
			slock.Unlock()
			end := time.Since(start)
			avg_time += smtpEntry.Duration
			s.GlobalStat.TotalDuration = end
			s.GlobalStat.AverageDuration = time.Duration(int64(avg_time) / count)
			s.GlobalStat.TotalSize = smtpEntry.ReadSize + smtpEntry.WriteSize
		}
	}
	size_in_mb := float64(s.GlobalStat.TotalSize) / (1 << 20) //For MB
	s.GlobalStat.Throughput = size_in_mb / s.GlobalStat.TotalDuration.Seconds()
	avg_time = time.Duration(int64(avg_time) / count)
	s.GlobalStat.AverageDuration = avg_time
}

func (s *SmtpStatCollector) Finished() {
	close(s.StatChannel)
}
