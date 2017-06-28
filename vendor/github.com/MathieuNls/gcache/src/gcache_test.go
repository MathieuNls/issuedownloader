package gcache

import (
	"strings"
	"testing"
)

func TestNew(t *testing.T) {

	cacheStore := GetCacheInstance()
	cacheStore.Put("store", "here", "there")

	value := cacheStore.Fetch("store", "here")

	if value != "there" {
		t.Error("expected there got", value)
	}

	value = cacheStore.FetchWithCB("store", "here", nil)

	if value != "there" {
		t.Error("expected there got", value)
	}

	value = cacheStore.FetchWithCB("store", "here", func(result interface{}) interface{} {
		return "bla"
	})

	if value != "bla" {
		t.Error("expected bla got", value)
	}

	value = cacheStore.FetchWithCB("store", "adw", func(result interface{}) interface{} {
		return "bla"
	})

	if value != nil {
		t.Error("expected nil got", value)
	}

	value = cacheStore.FetchWithCB("store", "awdawd", nil)

	if value != nil {
		t.Error("expected nil got", value)
	}

	cacheStore.Put("store", "herea", "there")

	value = cacheStore.FetchWithCB("store", "herea", nil)

	if value != "there" {
		t.Error("expected there got", value)
	}

	value = cacheStore.Fetch("storea", "here")

	if value != nil {
		t.Error("expected nil got", value)
	}

	if cacheStore.Stats("storea").Misses != 1 {
		t.Error("expected 1 got", cacheStore.Stats("storea").Misses)
	}

	if cacheStore.Stats("store").Misses != 0 || cacheStore.Stats("store").Hits != 6 {
		t.Error("expected 1 & 6 got", cacheStore.Stats("store"))
	}

	if strings.Contains(cacheStore.Stats("store").String(), "6") == false {
		t.Error("expected 6 got", cacheStore.Stats("store"))
	}

}
