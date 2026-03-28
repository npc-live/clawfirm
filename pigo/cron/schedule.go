// Package cron provides a scheduled-job runner for pi-go agents.
package cron

import (
	"container/list"
	"fmt"
	"sync"
	"time"

	robfigcron "github.com/robfig/cron/v3"
)

// lruCacheSize is the max number of parsed cron schedules to keep cached.
const lruCacheSize = 512

// scheduleCache is a simple LRU cache for parsed cron schedules.
type scheduleCache struct {
	mu    sync.Mutex
	cap   int
	items map[string]*list.Element
	order *list.List // front = most recently used
}

type cacheEntry struct {
	key string
	val robfigcron.Schedule
}

func newScheduleCache(capacity int) *scheduleCache {
	return &scheduleCache{
		cap:   capacity,
		items: make(map[string]*list.Element, capacity),
		order: list.New(),
	}
}

// get returns a cached schedule or nil.
func (c *scheduleCache) get(key string) robfigcron.Schedule {
	c.mu.Lock()
	defer c.mu.Unlock()
	if el, ok := c.items[key]; ok {
		c.order.MoveToFront(el)
		return el.Value.(*cacheEntry).val
	}
	return nil
}

// put stores a parsed schedule, evicting the oldest if at capacity.
func (c *scheduleCache) put(key string, val robfigcron.Schedule) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if el, ok := c.items[key]; ok {
		c.order.MoveToFront(el)
		el.Value.(*cacheEntry).val = val
		return
	}
	if c.order.Len() >= c.cap {
		oldest := c.order.Back()
		if oldest != nil {
			c.order.Remove(oldest)
			delete(c.items, oldest.Value.(*cacheEntry).key)
		}
	}
	el := c.order.PushFront(&cacheEntry{key: key, val: val})
	c.items[key] = el
}

// parseCronExpr parses a cron expression with optional timezone, using the LRU cache.
func parseCronExpr(cache *scheduleCache, expr, tz string) (robfigcron.Schedule, error) {
	key := expr + "|" + tz
	if s := cache.get(key); s != nil {
		return s, nil
	}

	var loc *time.Location
	if tz != "" {
		var err error
		loc, err = time.LoadLocation(tz)
		if err != nil {
			return nil, fmt.Errorf("cron: invalid timezone %q: %w", tz, err)
		}
	}

	parser := robfigcron.NewParser(robfigcron.Minute | robfigcron.Hour | robfigcron.Dom | robfigcron.Month | robfigcron.Dow | robfigcron.Descriptor)
	sched, err := parser.Parse(expr)
	if err != nil {
		return nil, fmt.Errorf("cron: parse %q: %w", expr, err)
	}

	// Wrap with timezone if specified.
	if loc != nil {
		sched = &tzSchedule{inner: sched, loc: loc}
	}

	cache.put(key, sched)
	return sched, nil
}

// tzSchedule wraps a cron.Schedule to compute Next in a specific timezone.
type tzSchedule struct {
	inner robfigcron.Schedule
	loc   *time.Location
}

func (s *tzSchedule) Next(t time.Time) time.Time {
	return s.inner.Next(t.In(s.loc))
}

// nextFireAt returns the duration until the "at" time fires.
func nextFireAt(at string) (time.Duration, error) {
	t, err := time.Parse(time.RFC3339, at)
	if err != nil {
		return 0, fmt.Errorf("cron: parse at %q: %w", at, err)
	}
	d := time.Until(t)
	if d < 0 {
		return 0, fmt.Errorf("cron: at time %q is in the past", at)
	}
	return d, nil
}

// nextFireEvery returns the initial delay for an "every" schedule,
// optionally aligning to an anchor.
func nextFireEvery(everyMs, anchorMs int64) time.Duration {
	every := time.Duration(everyMs) * time.Millisecond
	if anchorMs <= 0 {
		return every
	}
	anchor := time.Unix(0, anchorMs*int64(time.Millisecond))
	now := time.Now()
	if anchor.After(now) {
		return time.Until(anchor)
	}
	elapsed := now.Sub(anchor)
	remainder := elapsed % every
	if remainder == 0 {
		return every
	}
	return every - remainder
}
