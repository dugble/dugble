package ses

import (
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sesv2"
)

// NewForRegion constructs an SES service scoped to region.
func NewForRegion(cfg aws.Config, region string) (*Service, error) {
	region = strings.TrimSpace(region)
	if region == "" {
		return nil, fmt.Errorf("ses: region is required")
	}

	cfg.Region = region
	return &Service{
		client: sesv2.NewFromConfig(cfg),
		region: region,
	}, nil
}

// Region returns the AWS region used by the SES service.
func (s *Service) Region() string {
	if s == nil {
		return ""
	}
	return s.region
}
