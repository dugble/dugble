package emailtenant

import (
	"errors"

	"github.com/google/uuid"
)

func ValidateProvisioningCommand(command ProvisioningCommand) error {
	if command.EventID == uuid.Nil || command.TenantID == uuid.Nil || command.TeamID == uuid.Nil {
		return errors.New("email tenant provisioning command identifiers are required")
	}
	if command.SchemaVersion != 1 {
		return errors.New("unsupported email tenant provisioning schema version")
	}
	return nil
}
