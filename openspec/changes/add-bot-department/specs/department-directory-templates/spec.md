## ADDED Requirements

### Requirement: Define directory templates for department
The system SHALL allow department admins to define a list of directory paths that are provisioned into bot containers.

#### Scenario: Set directory templates
- **WHEN** an admin sends `PUT /departments/{department_id}/directory-templates` with `{ "paths": ["/data/reports", "/data/shared/docs", "/data/logs"] }`
- **THEN** the system stores the directory template paths in the department's `directory_templates` JSONB column

#### Scenario: Get directory templates
- **WHEN** a user sends `GET /departments/{department_id}/directory-templates`
- **THEN** the system returns the list of directory paths defined for the department

#### Scenario: Invalid path validation
- **WHEN** an admin sends directory paths containing `..` or paths not starting with `/data/`
- **THEN** the system returns a 400 error indicating invalid path

#### Scenario: Clear directory templates
- **WHEN** an admin sends `PUT /departments/{department_id}/directory-templates` with `{ "paths": [] }`
- **THEN** the system clears the directory templates for the department

### Requirement: Auto-provision directories when bot is added to department
The system SHALL create the department's directory structure inside a bot's container when the bot is associated with the department via `bot_department_access`.

#### Scenario: Bot added to department with directory templates
- **WHEN** a bot is associated with a department that has directory templates `["/data/reports", "/data/logs"]`
- **THEN** the system creates both directories inside the bot's container via gRPC

#### Scenario: Bot added to department without directory templates
- **WHEN** a bot is associated with a department that has no directory templates
- **THEN** no directory provisioning occurs

### Requirement: Bulk sync directories to all department bots
The system SHALL provide an endpoint to create missing department directories across all bots in the department.

#### Scenario: Sync directories
- **WHEN** an admin sends `POST /departments/{department_id}/sync-directories`
- **THEN** the system iterates through all bots in the department (via `bot_department_access`) and creates any directories from the template that don't already exist; existing directories SHALL NOT be modified or deleted

#### Scenario: Sync reports summary
- **WHEN** the sync processes multiple bots
- **THEN** the response includes total bots processed, directories created count, directories skipped (already exist) count, and any errors per bot
