package protocols

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/BatikanHyt/netbench/pkg/collector"
)

var protocolMap = map[string]func() BaseProtocol{
	"http": func() BaseProtocol { return NewHttpClient() },
	"smtp": func() BaseProtocol { return NewSmtpClient() },
}

var statMap = map[string]func() collector.StatBase{
	"http": func() collector.StatBase { return collector.CreateHttpStatCollector() },
	"smtp": func() collector.StatBase { return collector.CreateHttpStatCollector() },
}

type BaseProtocol interface {
	StartBenchmark()
	Initialize(collectorBase *collector.StatBase)
}

type Runner struct {
	Concurency    int    `json:"concurency"`
	TotalRequest  int    `json:"totalRequest"`
	Duration      string `json:"duration"`
	OutputFormat  string `json:"output"`
	Protocol      BaseProtocol
	StatCollector collector.StatBase
}

const (
	progressTimeThreshold     = time.Minute * 10
	progressRequestThreshhold = 100
)

func (r *Runner) Run() {
	if r.Protocol == nil {
		return
	}
	if r.StatCollector == nil {
		return
	}
	var wg sync.WaitGroup
	var cwg sync.WaitGroup
	r.Protocol.Initialize(&r.StatCollector)
	pool := make(chan struct{}, r.Concurency)
	cwg.Add(1)
	go r.StatCollector.Consume(&cwg)
	if r.Duration != "0s" {
		duration, _ := time.ParseDuration(r.Duration)
		timeout := time.After(duration)
		dur2 := progressTimeThreshold
		var progress_stats <-chan time.Time
		if duration > dur2 {
			progress_stats = time.After(duration / 10)
		}
	loop:
		for {
			select {
			case <-progress_stats:
				r.StatCollector.PrintProgressStats()
			case <-timeout:
				// timeout has been hit, break out of the loop
				break loop
			case pool <- struct{}{}:
				// acquire a token from the pool
				wg.Add(1)
				go func() {
					defer func() {
						// release the token
						<-pool
						wg.Done()
					}()
					r.Protocol.StartBenchmark()
				}()
			}
		}
	} else {
		wg.Add(r.TotalRequest)
		current_progress := 0
		printProgress := false
		if r.TotalRequest > progressRequestThreshhold {
			printProgress = true
		}
		for i := 0; i < r.TotalRequest; i++ {
			// acquire a token from the pool
			pool <- struct{}{}
			go func() {
				defer func() {
					if printProgress && current_progress >= r.TotalRequest/10 {
						current_progress = 0
						r.StatCollector.PrintProgressStats()
					}
					current_progress++
					// release the token
					<-pool
					wg.Done()
				}()
				r.Protocol.StartBenchmark()
			}()
		}
	}
	wg.Wait()
	r.StatCollector.Finished()
	cwg.Wait()
	r.printFinalResult()
}

func (r *Runner) printFinalResult() {
	globalStats := r.StatCollector.GetGlobalStats()
	req_per_sec := float64(globalStats.TotalRequest) / globalStats.TotalDuration.Seconds()
	fmt.Printf("\nTotal Request: %d, Total Duration: %s, Total recv/send bytes: %d\n"+
		"Succesfull requests: %d, Failed Requests %d Avg Response time:%s \nReq/sec:%f\nThroughput: %f MB/s\n",
		globalStats.TotalRequest, globalStats.TotalDuration, globalStats.TotalSize,
		globalStats.SuccessfulReq, globalStats.FailedReq, globalStats.AverageDuration, req_per_sec, globalStats.Throughput)

}

func (r *Runner) UnmarshalJSON(data []byte) error {
	var v map[string]interface{}
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}

	for key, value := range v {
		if key == "concurency" {
			r.Concurency = int(value.(float64))
		} else if key == "totalRequest" {
			r.TotalRequest = int(value.(float64))
		} else if key == "duration" {
			r.Duration = value.(string)
		} else if key == "output" {
			r.OutputFormat = value.(string)
		} else {
			createFunc, ok := protocolMap[key]
			if !ok {
				return fmt.Errorf("unsupported protocol: %s", key)
			}
			protoVal, ok := value.(map[string]interface{})
			if !ok {
				return fmt.Errorf("unsupported protocol: %s", key)
			}
			protoJson, _ := json.Marshal(protoVal)
			protocolValue := createFunc()
			if err := json.Unmarshal(protoJson, protocolValue); err != nil {
				return err
			}
			r.Protocol = protocolValue
			stat_funct := statMap[key]
			r.StatCollector = stat_funct()
		}
	}
	return nil
}
