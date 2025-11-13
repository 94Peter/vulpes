package ezgrpc

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/94peter/vulpes/ezgrpc/client"
)

var grpcClt client.Client

func init() {
	grpcClt = client.NewClient()
}

func Invoke[T any, R any](ctx context.Context, addr, service, method string, req T) (R, error) {
	var zeroR R

	if isNil(req) {
		return zeroR, fmt.Errorf("request is nil")
	}
	// reflect check req is nil
	jsonbody, err := json.Marshal(req)
	if err != nil {
		return zeroR, err
	}
	respByte, err := grpcClt.Invoke(ctx, addr, service, method, jsonbody)
	if err != nil {
		return zeroR, err
	}
	err = json.Unmarshal(respByte, &zeroR)
	if err != nil {
		return zeroR, err
	}
	return zeroR, nil
}

func Close() error {
	return grpcClt.Close()
}

func isNil[T any](v T) bool {
	if any(v) == nil {
		return true
	}
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Ptr, reflect.Slice:
		return rv.IsNil()
	}
	return false
}
