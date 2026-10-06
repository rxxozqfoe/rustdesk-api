package cache

import (
	"fmt"
	"testing"
)

func TestFileCacheSet(t *testing.T) {
	fc := New("file")
	err := fc.Set("123", "ddd", 0)
	if err != nil {
		fmt.Println(err.Error())
		t.Fatalf("写入失败")
	}
}

func TestFileCacheGet(t *testing.T) {
	fc := New("file")
	err := fc.Set("123", "45156", 300)
	if err != nil {
		t.Fatalf("写入失败")
	}
	res := ""
	err = fc.Get("123", &res)
	if err != nil {
		t.Fatalf("读取失败")
	}
	fmt.Println("res", res)
}

func TestRedisCacheSet(t *testing.T) {
	rc := NewRedis(redisTestOptions(t))
	err := rc.Set("123", "ddd", 0)
	if err != nil {
		fmt.Println(err.Error())
		t.Fatalf("写入失败")
	}
}

func TestRedisCacheGet(t *testing.T) {
	rc := NewRedis(redisTestOptions(t))
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
