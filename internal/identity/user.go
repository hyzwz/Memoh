package identity

import (
	"fmt"

	ctr "github.com/memohai/memoh/internal/containerd"
)

// ValidateUserID enforces a conservative ID charset for isolation.
func ValidateUserID(userID string) error {
	if userID == "" {
		return fmt.Errorf("%w: user id required", ctr.ErrInvalidArgument)
	}
	for _, r := range userID {
		if !(r == '-' || r == '_' || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')) {
			return fmt.Errorf("%w: invalid user id", ctr.ErrInvalidArgument)
		}
	}
	return nil
}
