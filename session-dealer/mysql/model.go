package mysql

import (
	log "github.com/sirupsen/logrus"
)

type handshakeResponse41 struct {
	Capability uint32
	Collation  uint8
	User       string
	DBName     string
	Auth       []byte
}

// jigsaw record tcp package begin and end seq id
type jigsaw struct {
	b int64
	e int64
}

type packageWindowCounter struct {
	sizeCount map[int]int64
	visitCount int64
	suggestSize int
}

func newPackageWindowCounter() *packageWindowCounter {
	return &packageWindowCounter{
		sizeCount: make(map[int]int64, 4),
		suggestSize: 512,
	}
}

func (pwc *packageWindowCounter) refresh (readSize int, isLastPackage bool)  {
	if pwc.visitCount > 10000 {
		return
	}

	log.Debugf("WindowCounter: %#v", pwc.sizeCount)
	pwc.visitCount += 1
	miniMatchSize := maxIPPackageSize
	for size := range pwc.sizeCount  {
		if readSize % size == 0 && miniMatchSize > size {
			miniMatchSize = size
		}
	}
	if miniMatchSize < maxIPPackageSize {
		pwc.sizeCount[miniMatchSize] = pwc.sizeCount[miniMatchSize] + 1
	} else if !isLastPackage {
		pwc.sizeCount[readSize] = 1
	}

	mostFrequentSize := pwc.suggestSize
	mostFrequentCount := int64(0)
	for size, count := range pwc.sizeCount  {
		if count > mostFrequentCount {
			mostFrequentSize = size
			mostFrequentCount = count
		}
	}

	pwc.suggestSize = mostFrequentSize
}