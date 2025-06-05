package constant

import "time"

const (
	UserCachePrefix    = "user:%s"
	UserCacheTTL       = time.Hour * 24
	VerificationTimeout = time.Minute * 5
	VerificationCoolDown = time.Minute * 2
)
