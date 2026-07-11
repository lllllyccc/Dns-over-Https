package resolver

import (
	"encoding/binary"
	"fmt"
	"sync"
	"time"

	"github.com/miekg/dns"
	bolt "go.etcd.io/bbolt"
)

type CacheEntry struct {
	Response  *dns.Msg
	ExpiresAt time.Time
}

type Cache struct {
	db      *bolt.DB
	mu      sync.RWMutex
	entries map[string]*CacheEntry
	maxSize int
	ttl     time.Duration
}

func NewCache(dbPath string, maxSize int, ttlSec int) (*Cache, error) {
	db, err := bolt.Open(dbPath, 0600, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		return nil, fmt.Errorf("open cache db: %w", err)
	}

	err = db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte("dns"))
		return err
	})
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("create cache bucket: %w", err)
	}

	return &Cache{
		db:      db,
		entries: make(map[string]*CacheEntry),
		maxSize: maxSize,
		ttl:     time.Duration(ttlSec) * time.Second,
	}, nil
}

func cacheKey(q *dns.Msg) string {
	if len(q.Question) == 0 {
		return ""
	}
	return fmt.Sprintf("%s:%d", q.Question[0].Name, q.Question[0].Qtype)
}

func (c *Cache) Get(q *dns.Msg) (*dns.Msg, bool) {
	key := cacheKey(q)
	if key == "" {
		return nil, false
	}

	c.mu.RLock()
	if entry, ok := c.entries[key]; ok {
		if time.Now().Before(entry.ExpiresAt) {
			resp := entry.Response.Copy()
			resp.Id = q.Id
			c.mu.RUnlock()
			return resp, true
		}
	}
	c.mu.RUnlock()

	var data []byte
	err := c.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("dns"))
		if b == nil {
			return nil
		}
		data = b.Get([]byte(key))
		return nil
	})
	if err != nil || data == nil {
		return nil, false
	}

	resp := new(dns.Msg)
	if err := resp.Unpack(data); err != nil {
		return nil, false
	}

	expiresAt := time.Unix(int64(binary.BigEndian.Uint64(data[len(data)-8:])), 0)
	if time.Now().After(expiresAt) {
		return nil, false
	}

	resp.Id = q.Id

	c.mu.Lock()
	c.entries[key] = &CacheEntry{Response: resp, ExpiresAt: expiresAt}
	if len(c.entries) > c.maxSize {
		c.evictOldest()
	}
	c.mu.Unlock()

	return resp, true
}

func (c *Cache) Set(q *dns.Msg, resp *dns.Msg) {
	key := cacheKey(q)
	if key == "" {
		return
	}

	ttl := c.ttl
	if len(resp.Answer) > 0 {
		if t := resp.Answer[0].Header().Ttl; t > 0 {
			d := time.Duration(t) * time.Second
			if d < ttl {
				ttl = d
			}
		}
	}

	expiresAt := time.Now().Add(ttl)
	entry := &CacheEntry{Response: resp.Copy(), ExpiresAt: expiresAt}

	data, err := resp.Pack()
	if err != nil {
		return
	}

	ts := make([]byte, 8)
	binary.BigEndian.PutUint64(ts, uint64(expiresAt.Unix()))
	data = append(data, ts...)

	c.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("dns"))
		return b.Put([]byte(key), data)
	})

	c.mu.Lock()
	c.entries[key] = entry
	if len(c.entries) > c.maxSize {
		c.evictOldest()
	}
	c.mu.Unlock()
}

func (c *Cache) evictOldest() {
	var oldestKey string
	var oldestTime time.Time
	for k, v := range c.entries {
		if oldestKey == "" || v.ExpiresAt.Before(oldestTime) {
			oldestKey = k
			oldestTime = v.ExpiresAt
		}
	}
	if oldestKey != "" {
		delete(c.entries, oldestKey)
	}
}

func (c *Cache) Purge() {
	c.mu.Lock()
	c.entries = make(map[string]*CacheEntry)
	c.mu.Unlock()

	c.db.Update(func(tx *bolt.Tx) error {
		return tx.DeleteBucket([]byte("dns"))
	})
	c.db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte("dns"))
		return err
	})
}

func (c *Cache) Stats() (total int, inMemory int) {
	c.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("dns"))
		if b != nil {
			total = b.Stats().KeyN
		}
		return nil
	})
	c.mu.RLock()
	inMemory = len(c.entries)
	c.mu.RUnlock()
	return
}

func (c *Cache) Close() error {
	return c.db.Close()
}
