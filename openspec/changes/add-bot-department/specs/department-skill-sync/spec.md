## ADDED Requirements

### Requirement: Associate skill templates with department
The system SHALL allow department admins to associate skill templates with a department. These define which skills are auto-installed for bots in the department.

#### Scenario: Add skill template to department
- **WHEN** an admin sends `POST /departments/{department_id}/skill-templates` with `{ "template_id": "<uuid>" }`
- **THEN** the system creates the association in `department_skill_templates` and returns success

#### Scenario: Add duplicate skill template
- **WHEN** an admin adds a skill template that is already associated with the department
- **THEN** the system returns a 409 conflict or ignores the duplicate (idempotent)

#### Scenario: Remove skill template from department
- **WHEN** an admin sends `DELETE /departments/{department_id}/skill-templates/{template_id}`
- **THEN** the system removes the association; already-installed skills on bots are not affected

#### Scenario: List department skill templates
- **WHEN** a user sends `GET /departments/{department_id}/skill-templates`
- **THEN** the system returns all skill templates associated with the department, including name, version, and description from the `skill_templates` table

### Requirement: Auto-install department skills when bot is added to department
The system SHALL automatically install all department skill templates into a bot's container when the bot is associated with a department via `bot_department_access`.

#### Scenario: Bot added to department with 3 skill templates
- **WHEN** a bot is associated with a department that has 3 skill templates (via `SetBotDepartmentAccess`)
- **THEN** the system installs all 3 skill templates into the bot's container, creating `bot_skill_installs` records for each

#### Scenario: Bot already has some department skills installed
- **WHEN** a bot is added to a department and already has 1 of 3 department skills installed
- **THEN** the system installs only the 2 missing skills; the existing one is not reinstalled

### Requirement: Bulk sync department skills to all bots
The system SHALL provide an endpoint to install/update department skill templates across all bots in the department.

#### Scenario: Sync skills to department bots
- **WHEN** an admin sends `POST /departments/{department_id}/sync-skills`
- **THEN** the system iterates through all bots in the department (via `bot_department_access`) and installs any missing skill templates; skills marked as `customized` in `bot_skill_installs` SHALL be skipped

#### Scenario: Sync reports summary
- **WHEN** the sync processes multiple bots
- **THEN** the response includes: total bots processed, skills installed count, skills skipped (customized) count, and any errors per bot
