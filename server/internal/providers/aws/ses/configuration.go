package ses

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sesv2"
)

const (
	TransactionalConfigurationSet = "dugble-transactional"
	MarketingConfigurationSet     = "dugble-marketing"
)

var sharedConfigurationSets = []string{
	TransactionalConfigurationSet,
	MarketingConfigurationSet,
}

// EnsureSharedConfigurationSets converges the shared Dugble SES configuration
// sets in one supported region. Existing sets are left in place.
func (c *Client) EnsureSharedConfigurationSets(ctx context.Context, region string) error {
	client, err := c.configurationClient(region)
	if err != nil {
		return err
	}
	for _, name := range sharedConfigurationSets {
		_, err := client.CreateConfigurationSet(ctx, &sesv2.CreateConfigurationSetInput{
			ConfigurationSetName: aws.String(name),
		})
		if err != nil && !isAlreadyExists(err) {
			return fmt.Errorf("create SES configuration set %q: %w", name, err)
		}
	}
	return nil
}
