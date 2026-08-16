package ses

import (
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sesv2"
)

// Service is Dugble's region-scoped AWS SES integration.
type Service struct {
	client *sesv2.Client
	region string
}

// New constructs an SES service using the region already resolved in cfg.
func New(cfg aws.Config) (*Service, error) {
	return NewForRegion(cfg, cfg.Region)
}
