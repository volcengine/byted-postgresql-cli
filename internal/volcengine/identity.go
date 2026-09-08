package volcengine

import (
	"context"
	"fmt"
	"strconv"

	"github.com/volcengine/volcengine-go-sdk/service/sts"
)

// CallerAccountID resolves the account that owns the current AK/SK or
// Console Login STS credentials through the Volcengine STS identity API.
func (c *Client) CallerAccountID(ctx context.Context) (string, error) {
	if c.sts == nil {
		return "", fmt.Errorf("STS identity client is not initialized")
	}
	result, err := c.sts.GetCallerIdentityWithContext(ctx, &sts.GetCallerIdentityInput{})
	if err != nil {
		return "", fmt.Errorf("failed to get caller identity: %w", err)
	}
	if result == nil || result.AccountId == nil || *result.AccountId <= 0 {
		return "", fmt.Errorf("STS caller identity returned no account ID")
	}
	return strconv.FormatInt(*result.AccountId, 10), nil
}
