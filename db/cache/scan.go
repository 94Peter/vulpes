package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/94peter/vulpes/log"
)

const (
	keyTypeString = "string"
	keyTypeHash   = "hash"
)

// ScanExecute iterates through keys in the cache matching a given pattern and executes a function for each key
// that can be successfully deserialized into the generic type T.
// It supports keys stored as JSON strings or Hashes.
// If the pattern is an empty string, it defaults to "*" to scan all keys.
func ScanExecute[T any](ctx context.Context, pattern string, f func(key string, value T) error) error {
	if conn == nil {
		return ErrCacheNotConnected
	}

	scanPattern := pattern
	if scanPattern == "" {
		scanPattern = "*" // Default to scanning all keys if no pattern is provided.
	}

	// Warning: SCAN with a broad match pattern like "*" can be slow and resource-intensive on large databases.
	// It's recommended to use a more specific pattern whenever possible to limit the scope of the scan.
	iter := conn.Scan(ctx, 0, scanPattern, 0).Iterator()
	for iter.Next(ctx) {
		key := iter.Val()
		keyType, err := conn.Type(ctx, key).Result()
		if err != nil {
			log.Warn(fmt.Sprintf("Error getting type for key %s: %v", key, err))
			continue
		}

		var value T
		var success bool

		switch keyType {
		case keyTypeString:
			// For strings, assume the value is a JSON-encoded object.
			valStr, err := conn.Get(ctx, key).Result()
			if err != nil {
				log.Warn(fmt.Sprintf("Error getting string value for key %s: %v", key, err))
				continue
			}
			if err := json.Unmarshal([]byte(valStr), &value); err == nil {
				success = true
			}
			// If unmarshal fails, we assume it's not the target type and just continue.

		case keyTypeHash:
			// For hashes, scan the fields directly into the struct.
			if err := conn.HGetAll(ctx, key).Scan(&value); err == nil {
				success = true
			}
			// If scan fails, we assume the hash doesn't match the struct and continue.

		default:
			// Ignore other Redis types (list, set, zset, etc.)
			continue
		}

		if success {
			if err := f(key, value); err != nil {
				log.Warn(fmt.Sprintf("Error executing callback for key %s: %v", key, err))
				// Continue processing other keys even if one callback fails.
				continue
			}
		}
	}

	if err := iter.Err(); err != nil {
		log.Error("Error during cache scan iteration", log.Err(err))
		return err
	}

	return nil
}

// ScanExecuteInt iterates through keys in the cache matching a given pattern and executes a function for each key
// whose value can be parsed as an integer.
// It only considers keys of type 'string'.
// If the pattern is an empty string, it defaults to "*" to scan all keys.
func DeleteAfterScanExecuteInt(ctx context.Context, pattern string, f func(key string, value int) error) error {
	if conn == nil {
		return ErrCacheNotConnected
	}

	scanPattern := pattern
	if scanPattern == "" {
		scanPattern = "*" // Default to scanning all keys if no pattern is provided.
	}

	iter := conn.Scan(ctx, 0, scanPattern, 0).Iterator()
	for iter.Next(ctx) {
		key := iter.Val()
		keyType, err := conn.Type(ctx, key).Result()
		if err != nil {
			log.Warn(fmt.Sprintf("Error getting type for key %s: %v", key, err))
			continue
		}
		if keyType != keyTypeString {
			continue
		}

		valStr, err := conn.Get(ctx, key).Result()
		if err != nil {
			log.Warn(fmt.Sprintf("Error getting string value for key %s: %v", key, err))
			continue
		}

		valInt, err := strconv.Atoi(valStr)
		if err != nil {
			// Not an integer value, just skip.
			continue
		}

		if err := f(key, valInt); err != nil {
			log.Warn(fmt.Sprintf("Error executing callback for key %s: %v", key, err))
			// Continue processing other keys even if one callback fails.
			continue
		}
		err = conn.Del(ctx, key).Err()
		if err != nil {
			log.Warn(fmt.Sprintf("Error deleting key %s: %v", key, err))
			continue
		}
	}

	if err := iter.Err(); err != nil {
		log.Error("Error during cache scan iteration", log.Err(err))
		return err
	}

	return nil
}
