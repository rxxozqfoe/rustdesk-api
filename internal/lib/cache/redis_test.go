package cache

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/go-redis/redis/v8"
)

// redisTestOptions returns the connection options for the Redis integration
// tests. The address defaults to a developer machine but can be overridden with
// REDIS_TEST_ADDR. These tests need a real Redis; when none is reachable they
// skip so unit CI (which has no Redis) stays green instead of timing out.
func redisTestOptions(tb testing.TB) *redis.Options {
	tb.Helper()
	addr := os.Getenv("REDIS_TEST_ADDR")
	if addr == "" {
		addr = "192.168.1.168:6379"
	}
	opt := &redis.Options{
		Addr:        addr,
		Password:    "", // no password set
		DB:          0,  // use default DB
		DialTimeout: 500 * time.Millisecond,
	}

	c := redis.NewClient(opt)
	defer func() { _ = c.Close() }()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := c.Ping(ctx).Err(); err != nil {
		tb.Skipf("redis not reachable at %s: %v", addr, err)
	}
	return opt
}

func TestRedisSet(t *testing.T) {
	//rc := New("redis")
	rc := RedisCacheInit(redisTestOptions(t))
	err := rc.Set("123", "ddd", 0)
	if err != nil {
		fmt.Println(err.Error())
		t.Fatalf("写入失败")
	}
}

func TestRedisGet(t *testing.T) {
	rc := RedisCacheInit(redisTestOptions(t))
	err := rc.Set("123", "451156", 300)
	if err != nil {
		t.Fatalf("写入失败")
	}
	res := ""
	err = rc.Get("123", &res)
	if err != nil {
		t.Fatalf("读取失败")
	}
	fmt.Println("res", res)
}

func TestRedisGetJson(t *testing.T) {
	rc := RedisCacheInit(redisTestOptions(t))
	type r struct {
		Aa string `json:"a"`
		B  string `json:"c"`
	}
	old := &r{
		Aa: "ab", B: "cdc",
	}
	err := rc.Set("1233", old, 300)
	if err != nil {
		t.Fatalf("写入失败")
	}

	res := &r{}
	err2 := rc.Get("1233", res)
	if err2 != nil {
		t.Fatalf("读取失败")
	}
	if !reflect.DeepEqual(res, old) {
		t.Fatalf("读取错误")
	}
	fmt.Println(res, res.Aa)
}

func BenchmarkRSet(b *testing.B) {
	rc := RedisCacheInit(redisTestOptions(b))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := rc.Set("123", "{dsv}", 1000); err != nil {
			b.Fatalf("写入失败%v", err)
		}
	}
}

func BenchmarkRGet(b *testing.B) {
	rc := RedisCacheInit(redisTestOptions(b))
	b.ResetTimer()
	v := ""
	for i := 0; i < b.N; i++ {
		if err := rc.Get("123", &v); err != nil {
			b.Fatalf("读取失败%v", err)
		}
	}
}
