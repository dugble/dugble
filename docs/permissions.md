# Permissions and Roles

This document describes the authorization model used by the customer and team dashboard API. It covers roles, permissions, scopes, and how they interact during request authorization.

## Overview

The API uses a role-based access control (RBAC) system combined with fine-grained permissions. Team members are assigned roles, and each role grants a specific set of permissions. API tokens can also be issued with explicit scopes that map directly to permissions.

## Roles

Every team member has one of three roles:

| Role     | Description                                                                 |
|----------|-----------------------------------------------------------------------------|
| `owner`  | Full access to all team resources, including team deletion and role changes |
| `admin`  | Full access except team deletion and changing member roles                  |
| `member` | Read-only access to most resources, limited write capabilities              |

### Role Permissions

#### Owner

Owners have all permissions except:
- `team_members:leave` (owners cannot leave their own team without transferring ownership)

#### Admin

Admins have all permissions except:
- `team:delete`
- `team_members:role`

#### Member

Members have read-only access to most resources and limited write capabilities:

**Read permissions:**
- `team:read`
- `team_members:read`
- `sender_ids:read`
- `sender_domains:read`
- `sms:read`
- `email:read`
- `verify:read`
- `webhooks:read`
- `topics:read`
- `suppressions:read`
- `broadcasts:read`
- `segments:read`
- `templates:read`
- `contacts:read`
- `contact_properties:read`

**Write permissions:**
- `team_members:leave`
- `broadcasts:write`

## Permissions Reference

Permissions follow a `resource:action` format. The following permissions are available:

### Team Management

| Permission                | Description                          |
|---------------------------|--------------------------------------|
| `team:read`               | View team details                    |
| `team:update`             | Update team settings                 |
| `team:delete`             | Delete the team                      |
| `team_members:read`       | View team members                    |
| `team_members:leave`      | Leave the team                       |
| `team_members:invite`     | Invite new members                   |
| `team_members:remove`     | Remove members from the team         |
| `team_members:role`       | Change member roles                  |
| `team_tokens:read`        | View API tokens                      |
| `team_tokens:create`      | Create new API tokens                |
| `team_tokens:update`      | Update API token settings            |
| `team_tokens:revoke`      | Revoke API tokens                    |

### Sender Identity

| Permission                | Description                          |
|---------------------------|--------------------------------------|
| `sender_ids:read`         | View sender IDs                      |
| `sender_ids:create`       | Create new sender IDs                |
| `sender_ids:delete`       | Delete sender IDs                    |
| `sender_domains:read`     | View sender domains                  |
| `sender_domains:create`   | Create new sender domains            |
| `sender_domains:delete`   | Delete sender domains                |

### Messaging

| Permission                | Description                          |
|---------------------------|--------------------------------------|
| `sms:read`                | View SMS messages and history        |
| `sms:send`                | Send SMS messages                    |
| `email:read`              | View email messages and history      |
| `email:send`              | Send email messages                  |

### Verification

| Permission                | Description                          |
|---------------------------|--------------------------------------|
| `verify:read`             | View verification requests           |
| `verify:send`             | Send verification codes              |
| `verify:check`            | Check verification codes             |
| `verify:manage`           | Manage verification settings         |

### Webhooks & Events

| Permission                | Description                          |
|---------------------------|--------------------------------------|
| `webhooks:read`           | View webhook configurations          |
| `webhooks:write`          | Create or update webhooks            |
| `audit_events:read`       | View audit logs                      |

### Wallet & Billing

| Permission                | Description                          |
|---------------------------|--------------------------------------|
| `wallet:read`             | View wallet balance                  |
| `wallet_ledger:read`      | View wallet transaction history      |

### Topics & Segments

| Permission                | Description                          |
|---------------------------|--------------------------------------|
| `topics:read`             | View topics                          |
| `topics:write`            | Create or update topics              |
| `segments:read`           | View segments                        |
| `segments:write`          | Create or update segments            |

### Contacts

| Permission                     | Description                          |
|--------------------------------|--------------------------------------|
| `contacts:read`                | View contacts                        |
| `contacts:write`               | Create, update, or delete contacts   |
| `contact_properties:read`      | View contact properties schema       |
| `contact_properties:write`     | Modify contact properties schema     |

### Templates

| Permission                | Description                          |
|---------------------------|--------------------------------------|
| `templates:read`          | View message templates               |
| `templates:write`         | Create or update templates           |

### Broadcasts

| Permission                | Description                          |
|---------------------------|--------------------------------------|
| `broadcasts:read`         | View broadcast campaigns             |
| `broadcasts:write`        | Create or update broadcasts          |
| `broadcasts:send`         | Send broadcast campaigns             |

### Suppressions

| Permission                | Description                          |
|---------------------------|--------------------------------------|
| `suppressions:read`       | View suppression list                |
| `suppressions:write`      | Add or remove suppressions           |

## Scopes

Scopes are used when creating API tokens to grant specific permissions. Each scope maps directly to a permission with the same name. For example, a token with scope `contacts:read` can read contacts but cannot modify them.

When a first-party user authenticates via session, their effective scopes are determined by their team role. The session automatically includes all permissions granted by that role.

### Token Scopes

API tokens must be explicitly granted scopes at creation time. The token's effective permissions are exactly the scopes assigned to it. Tokens do not inherit role-based permissions.

Supported scopes match the permission names listed above (e.g., `team:read`, `sms:send`, `contacts:write`).

## Authorization Flow

1. **Authentication**: The request is authenticated via session cookie or API token.
2. **Scope Resolution**:
   - Session: Scopes are derived from the user's team role.
   - Token: Scopes are taken from the token's granted scopes.
3. **Permission Check**: The handler checks if the resolved scopes include the required permission(s) for the action.
4. **Authorization Decision**: If the check passes, the request proceeds. Otherwise, a `403 Forbidden` response is returned.

## Error Responses

When authorization fails, the API returns:

```json
{
  "success": false,
  "error": {
    "code": "FORBIDDEN",
    "message": "You do not have permission to perform this action"
  }
}
```

Common scenarios:
- **Insufficient role**: A member trying to delete the team receives `403 Forbidden`.
- **Missing scope**: An API token without `contacts:write` trying to create a contact receives `403 Forbidden`.
- **Team membership lost**: A user whose membership was revoked receives `403 Forbidden` on subsequent team-scoped requests.

## Best Practices

1. **Principle of Least Privilege**: Issue API tokens with only the scopes required for the specific integration.
2. **Role Assignment**: Assign the minimum role necessary for team members to perform their duties.
3. **Error Handling**: Frontend code should handle `403 Forbidden` gracefully, potentially redirecting users or disabling UI elements based on their permissions.
4. **Token Rotation**: Regularly rotate API tokens and review their granted scopes.
