package ratelimitq

import (
	"io/ioutil"
	"net/url"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/boltdb/bolt"
	"github.com/fanyang01/crawler/queue"
	"github.com/fanyang01/crawler/queue/ratelimitq/diskheap"
	"github.com/stretchr/testify/assert"
)

func mustParseURL(ur string) *url.URL {
	u, err := url.Parse(ur)
	if err != nil {
		panic(err)
	}
	return u
}

func mustParseInt(s string) int {
	i, err := strconv.ParseInt(s, 0, 32)
	if err != nil {
		panic(err)
	}
	return int(i)
}

func testPriority(t *testing.T, secondary Secondary) {
	pq := New(&Option{MaxHosts: 100, Secondary: secondary})
	now := time.Now()
	pq.Push(&queue.Item{
		Score: 300,
		URL:   mustParseURL("/300"),
		Next:  now.Add(50 * time.Millisecond),
	})
	pq.Push(&queue.Item{
		Score: 100,
		URL:   mustParseURL("/100"),
		Next:  now.Add(50 * time.Millisecond),
	})
	pq.Push(&queue.Item{
		Score: 200,
		URL:   mustParseURL("/200"),
		Next:  now.Add(50 * time.Millisecond),
	})
	item, _ := pq.Pop()
	assert.Equal(t, "/300", item.URL.Path)
	item, _ = pq.Pop()
	assert.Equal(t, "/200", item.URL.Path)
	item, _ = pq.Pop()
	assert.Equal(t, "/100", item.URL.Path)
}

func testTime(t *testing.T, secondary Secondary) {
	wq := New(&Option{MaxHosts: 100, Secondary: secondary})
	now := time.Now()
	items := []*queue.Item{
		{
			Next: now.Add(50 * time.Millisecond),
			URL:  mustParseURL("http://a.example.com/50"),
		}, {
			Next: now.Add(75 * time.Millisecond),
			URL:  mustParseURL("http://b.example.com/75"),
		}, {
			Next: now.Add(25 * time.Millisecond),
			URL:  mustParseURL("http://a.example.com/25"),
		}, {
			Next: now.Add(100 * time.Millisecond),
			URL:  mustParseURL("http://b.example.com/100"),
		},
	}
	exp := []string{
		"/25",
		"/50",
		"/75",
		"/100",
	}
	for _, item := range items {
		wq.Push(item)
	}
	for i := 0; i < len(items); i++ {
		item, _ := wq.Pop()
		assert.Equal(t, exp[i], item.URL.Path)
	}
}

func testRateLimit(t *testing.T, secondary Secondary) {
	f := func(host string) time.Duration {
		switch host {
		case "a.example.com":
			return 50 * time.Millisecond
		case "b.example.com":
			return 25 * time.Millisecond
		default:
			return 0
		}
	}
	wq := New(&Option{MaxHosts: 100, Limit: f, Secondary: secondary})
	now := time.Now()
	items := []*queue.Item{
		{
			Next: now.Add(25 * time.Millisecond),
			URL:  mustParseURL("http://a.example.com/25"),
		}, {
			Next: now.Add(50 * time.Millisecond),
			URL:  mustParseURL("http://a.example.com/50"),
		}, {
			Next: now.Add(60 * time.Millisecond),
			URL:  mustParseURL("http://b.example.com/60"),
		}, {
			Next: now.Add(100 * time.Millisecond),
			URL:  mustParseURL("http://b.example.com/100"),
		},
	}
	exp := []string{
		"/25",
		"/60",
		"/50",
		"/100",
	}
	for _, item := range items {
		wq.Push(item)
	}
	for i := 0; i < len(items); i++ {
		item, _ := wq.Pop()
		assert.Equal(t, exp[i], item.URL.Path)
	}
}

func tmpfile() string {
	f, _ := ioutil.TempFile("", "ratelimitq")
	name := f.Name()
	f.Close()
	return name
}

func newDiskHeap(t *testing.T, name string, bufsize int) *diskheap.DiskHeap {
	db, err := bolt.Open(name, 0644, nil)
	if err != nil {
		t.Fatal(err)
	}
	return diskheap.New(db, nil, bufsize)
}

func TestTime(t *testing.T) {
	testTime(t, nil)
	name := tmpfile()
	defer os.Remove(name)
	testTime(t, newDiskHeap(t, name, 0))

	name = tmpfile()
	defer os.Remove(name)
	testTime(t, newDiskHeap(t, name, 2))
}

func TestPriority(t *testing.T) {
	testPriority(t, nil)
	name := tmpfile()
	defer os.Remove(name)
	testPriority(t, newDiskHeap(t, name, 1))
}

func TestRateLimit(t *testing.T) {
	testRateLimit(t, nil)
	name := tmpfile()
	defer os.Remove(name)
	testRateLimit(t, newDiskHeap(t, name, 0))

	name = tmpfile()
	defer os.Remove(name)
	testRateLimit(t, newDiskHeap(t, name, 3))
}
