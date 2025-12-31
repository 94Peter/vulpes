package constant

import "time"

const DefaultTimeout = 5 * time.Second
const DefaultKeepAlive = 3 * time.Minute
const DefaultIdleTimeout = time.Minute
const DefaultReadHeaderTimeout = 5 * time.Second
const DefaultReadTimeout = 15 * time.Second
const DefaultWriteTimeout = 15 * time.Second

const DefaultBackoffBaseDelay = 500 * time.Millisecond
const DefaultBackoffMultiplier = 1.1
const DefaultBackoffJitter = 0.1
const DefaultBackoffMaxDelay = 3 * time.Second
