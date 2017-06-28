[![Build Status](https://travis-ci.org/MathieuNls/gcache.png)](https://travis-ci.org/MathieuNls/gcache)
[![GoDoc](https://godoc.org/github.com/MathieuNls/gcache?status.png)](https://godoc.org/github.com/MathieuNls/gcache)
[![codecov](https://codecov.io/gh/MathieuNls/gcache/branch/master/graph/badge.svg)](https://codecov.io/gh/MathieuNls/gcache)

# gcache

```go

cacheStore := GetCacheInstance()
cacheStore.Put("store", "key", "value")
value := cacheStore.Fetch("store", "key") //value == "value"
```