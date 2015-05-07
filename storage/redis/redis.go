package redis

import (
	"encoding/json"
	redis "github.com/garyburd/redigo/redis"
	info "github.com/google/cadvisor/info/v1"
	storage "github.com/google/cadvisor/storage"
	"sync"
	"time"
)

type redisStorage struct {
	conn           redis.Conn
	machineName    string
	redisKey       string
	bufferDuration time.Duration
	lastWrite      time.Time
	lock           sync.Mutex
	readyToFlush   func() bool
}

func (self *redisStorage) OverrideReadyToFlush(readyToFlush func() bool) {
	self.readyToFlush = readyToFlush
}

func (self *redisStorage) defaultReadyToFlush() bool {
	return time.Since(self.lastWrite) >= self.bufferDuration
}

func (self *redisStorage) AddStats(ref info.ContainerReference, stats *info.ContainerStats) error {
	if stats == nil {
		return nil
	}
	var seriesToFlush []byte
	func() {
		// AddStats will be invoked simultaneously from multiple threads and only one of them will perform a write.
		self.lock.Lock()
		defer self.lock.Unlock()
		b, _ := json.Marshal(stats)
		if self.readyToFlush() {
			seriesToFlush = b
			b = nil
			self.lastWrite = time.Now()
		}
	}()
	if len(seriesToFlush) > 0 {
		//use redis's "LPUSH" to push the data to the redis
		self.conn.Send("LPUSH", self.redisKey, seriesToFlush)
	}
	return nil
}

// just need to push the data to the redis, do not need to pull form the redis,so not override RecentStats()
func (self *redisStorage) RecentStats(containerName string, numStats int) ([]*info.ContainerStats, error) {
	return nil, nil
}

func (self *redisStorage) Close() error {
	return self.conn.Close()
}

// Create a new redis storage driver.
// machineName: A unique identifier to identify the host that current cAdvisor
// instance is running on.
// redisHost: The host which runs redis.
// redisKey: The key for the Data that stored in the redis
func New(machineName,
	redisKey,
	redisHost string,
	bufferDuration time.Duration,
) (storage.StorageDriver, error) {
	conn, err := redis.Dial("tcp", redisHost)
	if err != nil {
		return nil, err
	}
	ret := &redisStorage{
		conn:           conn,
		machineName:    machineName,
		redisKey:       redisKey,
		bufferDuration: bufferDuration,
		lastWrite:      time.Now(),
	}
	ret.readyToFlush = ret.defaultReadyToFlush
	return ret, nil
}
