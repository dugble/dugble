package ses

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sesv2"
	sestypes "github.com/aws/aws-sdk-go-v2/service/sesv2/types"
)

const FeedbackEventDestinationName = "dugble-feedback"

var feedbackEventTypes = []sestypes.EventType{
	sestypes.EventTypeSend,
	sestypes.EventTypeReject,
	sestypes.EventTypeBounce,
	sestypes.EventTypeComplaint,
	sestypes.EventTypeDelivery,
	sestypes.EventTypeOpen,
	sestypes.EventTypeClick,
	sestypes.EventTypeRenderingFailure,
	sestypes.EventTypeDeliveryDelay,
	sestypes.EventTypeSubscription,
}

// EnsureSharedFeedbackDestinations converges one SNS feedback destination on
// each shared Dugble configuration set in region. The SNS topic is supplied by
// the caller; SES does not create or manage SNS topics here.
func (c *Client) EnsureSharedFeedbackDestinations(ctx context.Context, region, topicARN string) error {
	region, supported := NormalizeSESRegion(region)
	if !supported {
		return fmt.Errorf("SES configuration region %q is not supported", strings.TrimSpace(region))
	}
	if err := validateSNSTopicARN(topicARN, region); err != nil {
		return err
	}
	if err := c.EnsureSharedConfigurationSets(ctx, region); err != nil {
		return err
	}
	client, err := c.configurationClient(region)
	if err != nil {
		return err
	}
	for _, configurationSet := range sharedConfigurationSets {
		if err := ensureFeedbackDestination(ctx, client, configurationSet, strings.TrimSpace(topicARN)); err != nil {
			return err
		}
	}
	return nil
}

func ensureFeedbackDestination(ctx context.Context, client sesConfigurationAPI, configurationSet, topicARN string) error {
	definition := &sestypes.EventDestinationDefinition{
		Enabled:            true,
		MatchingEventTypes: append([]sestypes.EventType(nil), feedbackEventTypes...),
		SnsDestination:     &sestypes.SnsDestination{TopicArn: aws.String(topicARN)},
	}
	_, err := client.CreateConfigurationSetEventDestination(ctx, &sesv2.CreateConfigurationSetEventDestinationInput{
		ConfigurationSetName: aws.String(configurationSet),
		EventDestinationName: aws.String(FeedbackEventDestinationName),
		EventDestination:     definition,
	})
	if err == nil {
		return nil
	}
	if !isAlreadyExists(err) {
		return fmt.Errorf("create SES event destination for %q: %w", configurationSet, err)
	}
	_, err = client.UpdateConfigurationSetEventDestination(ctx, &sesv2.UpdateConfigurationSetEventDestinationInput{
		ConfigurationSetName: aws.String(configurationSet),
		EventDestinationName: aws.String(FeedbackEventDestinationName),
		EventDestination:     definition,
	})
	if err != nil {
		return fmt.Errorf("update SES event destination for %q: %w", configurationSet, err)
	}
	return nil
}

func validateSNSTopicARN(topicARN, region string) error {
	topicARN = strings.TrimSpace(topicARN)
	if topicARN == "" {
		return errors.New("SNS topic ARN is required for SES feedback")
	}
	parts := strings.SplitN(topicARN, ":", 6)
	if len(parts) != 6 || parts[0] != "arn" || strings.TrimSpace(parts[1]) == "" || parts[2] != "sns" || strings.TrimSpace(parts[4]) == "" || strings.TrimSpace(parts[5]) == "" {
		return fmt.Errorf("invalid SNS topic ARN %q", topicARN)
	}
	if parts[3] != region {
		return fmt.Errorf("SNS topic region %q does not match SES region %q", parts[3], region)
	}
	return nil
}
