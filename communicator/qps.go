package communicator

import (
	"fmt"
	"math"
	"sync"
	"time"
)

const (
	CACHE_SIZE              = 1024
	STATISTIC_SECONDS int64 = 5
)

var (
	execTimeChan  chan int64
	execTimeCache []int64
	qpsLock       sync.Mutex
)

func init() {
	execTimeChan = make(chan int64, 256)
	execTimeCache = make([]int64, 0, CACHE_SIZE)
	go updateCachedExecTime()
}

type qpsConfig struct{}

func (qc *qpsConfig) setVal(val interface{}) (err error) {
	err = fmt.Errorf("cannot set QPS on sniffer")
	return
}

func (qc *qpsConfig) getVal() (val interface{}) {
	return computeQPS()
}

func ReceiveExecTime(execTime int64) {
	select {
	case execTimeChan <- execTime:
	default:
		return
	}
}

func updateCachedExecTime() {
	for et := range execTimeChan {
		qpsLock.Lock()
		execTimeCache = append(execTimeCache, et)
		if len(execTimeCache) > CACHE_SIZE {
			execTimeCache = execTimeCache[1:]
		}
		qpsLock.Unlock()
	}
}

func computeQPS() (qps int64) {
	if catpurePacketRate.mysqlCPR <= 0 {
		return 0
	}

	qpsLock.Lock()
	defer qpsLock.Unlock()

	// only deal execute time last 10 second
	nowNano := time.Now().UnixNano()
	lastTimeNano := nowNano - time.Second.Nanoseconds()*STATISTIC_SECONDS
	minExecTimeNano := nowNano
	var recentRecordNum int64
	for _, et := range execTimeCache {
		// ignore execute time before 10 second
		if et < lastTimeNano {
			continue
		}

		recentRecordNum += 1
		if et < minExecTimeNano {
			minExecTimeNano = et
		}
	}

	if recentRecordNum < 1 || nowNano == minExecTimeNano {
		return 0
	}

	qpsVal := float64(time.Second.Nanoseconds() /
		((nowNano - minExecTimeNano) / recentRecordNum)) /
		catpurePacketRate.mysqlCPR
	return int64(math.Floor(qpsVal))
}
