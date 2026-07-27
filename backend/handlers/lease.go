package handlers

import (
	"log"
	"time"

	"github.com/jiabaoliu101/aiswitch/backend/database"
)

// RunLeaseExpirySweep marks accounts whose subscriber no longer has an active
// subscription. It intentionally does not make an account reusable: a worker
// or an administrator must revoke and verify the upstream credential before
// changing expiring -> revoking -> verified_revoked -> ready.
func RunLeaseExpirySweep() {
	result, err := database.DB.Exec(`
		UPDATE account_pool ap
		SET status = 'expiring', updated_at = NOW()
		WHERE ap.status = 'assigned'
		  AND EXISTS (
			SELECT 1 FROM account_assignments aa
			WHERE aa.account_id = ap.id AND aa.status = 'active'
		  )
		  AND NOT EXISTS (
			SELECT 1
			FROM account_assignments aa
			JOIN user_subscriptions us ON us.user_id = aa.user_id
			WHERE aa.account_id = ap.id
			  AND aa.status = 'active'
			  AND us.status = 'active'
			  AND us.expires_at > NOW()
		  )`,
	)
	if err != nil {
		log.Printf("lease expiry sweep failed: %v", err)
		return
	}
	if count, err := result.RowsAffected(); err == nil && count > 0 {
		log.Printf("lease expiry sweep marked %d account(s) as expiring", count)
	}
}

// StartLeaseExpirySweep runs once at startup and then periodically. The sweep
// only transitions state; it never revokes an upstream token by itself.
func StartLeaseExpirySweep() {
	RunLeaseExpirySweep()
	ticker := time.NewTicker(15 * time.Minute)
	go func() {
		for range ticker.C {
			RunLeaseExpirySweep()
		}
	}()
}
