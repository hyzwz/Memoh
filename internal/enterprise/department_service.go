package enterprise

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"path"
	"strings"

	"github.com/memohai/memoh/internal/config"
	"github.com/memohai/memoh/internal/db"
	"github.com/memohai/memoh/internal/db/sqlc"
	"github.com/memohai/memoh/internal/handlers"
)

const skillsDirPath = config.DefaultDataMount + "/.skills"

// ContainerClient is the subset of mcpclient.Client used for department sync operations.
type ContainerClient interface {
	Mkdir(ctx context.Context, path string) error
	WriteFile(ctx context.Context, path string, content []byte) error
}

// ContainerClientProvider returns a container client for a given bot.
type ContainerClientProvider interface {
	MCPClient(ctx context.Context, botID string) (ContainerClient, error)
}

// DepartmentService implements handlers.DepartmentServiceInterface.
type DepartmentService struct {
	queries  *sqlc.Queries
	logger   *slog.Logger
	manager  ContainerClientProvider
}

func NewDepartmentService(log *slog.Logger, queries *sqlc.Queries, manager ContainerClientProvider) *DepartmentService {
	return &DepartmentService{
		queries: queries,
		logger:  log.With(slog.String("service", "departments")),
		manager: manager,
	}
}

func (s *DepartmentService) ListDepartments(ctx context.Context, botID string) ([]handlers.DepartmentDTO, error) {
	pgBotID, err := db.ParseUUID(botID)
	if err != nil {
		return nil, err
	}
	rows, err := s.queries.ListBotDepartments(ctx, pgBotID)
	if err != nil {
		return nil, err
	}
	out := make([]handlers.DepartmentDTO, len(rows))
	for i, r := range rows {
		out[i] = departmentToDTO(r)
	}
	return out, nil
}

func (s *DepartmentService) CreateDepartment(ctx context.Context, botID string, req handlers.CreateDepartmentRequest) (*handlers.DepartmentDTO, error) {
	pgBotID, err := db.ParseUUID(botID)
	if err != nil {
		return nil, err
	}

	dept, err := s.queries.CreateDepartment(ctx, sqlc.CreateDepartmentParams{
		Name:        req.Name,
		Description: req.Description,
		ParentID:    db.ParseUUIDOrEmpty(req.ParentID),
		Metadata:    []byte("{}"),
	})
	if err != nil {
		return nil, err
	}

	// Associate department with bot.
	if err := s.queries.SetBotDepartmentAccess(ctx, sqlc.SetBotDepartmentAccessParams{
		BotID:        pgBotID,
		DepartmentID: dept.ID,
	}); err != nil {
		return nil, err
	}

	dto := departmentToDTO(dept)
	return &dto, nil
}

func (s *DepartmentService) AddSkillTemplate(ctx context.Context, departmentID, templateID string) error {
	pgDeptID, err := db.ParseUUID(departmentID)
	if err != nil {
		return err
	}
	pgTmplID, err := db.ParseUUID(templateID)
	if err != nil {
		return err
	}
	return s.queries.AddDepartmentSkillTemplate(ctx, sqlc.AddDepartmentSkillTemplateParams{
		DepartmentID: pgDeptID,
		TemplateID:   pgTmplID,
	})
}

func (s *DepartmentService) RemoveSkillTemplate(ctx context.Context, departmentID, templateID string) error {
	pgDeptID, err := db.ParseUUID(departmentID)
	if err != nil {
		return err
	}
	pgTmplID, err := db.ParseUUID(templateID)
	if err != nil {
		return err
	}
	return s.queries.RemoveDepartmentSkillTemplate(ctx, sqlc.RemoveDepartmentSkillTemplateParams{
		DepartmentID: pgDeptID,
		TemplateID:   pgTmplID,
	})
}

func (s *DepartmentService) ListSkillTemplates(ctx context.Context, departmentID string) ([]handlers.SkillTemplateBriefDTO, error) {
	pgDeptID, err := db.ParseUUID(departmentID)
	if err != nil {
		return nil, err
	}
	rows, err := s.queries.ListDepartmentSkillTemplates(ctx, pgDeptID)
	if err != nil {
		return nil, err
	}
	out := make([]handlers.SkillTemplateBriefDTO, len(rows))
	for i, r := range rows {
		out[i] = handlers.SkillTemplateBriefDTO{
			ID:          uuidToString(r.ID),
			Name:        r.Name,
			Description: r.Description,
			Version:     r.Version,
		}
	}
	return out, nil
}

func (s *DepartmentService) GetDirectoryTemplates(ctx context.Context, departmentID string) ([]string, error) {
	pgDeptID, err := db.ParseUUID(departmentID)
	if err != nil {
		return nil, err
	}
	raw, err := s.queries.GetDepartmentDirectoryTemplates(ctx, pgDeptID)
	if err != nil {
		return nil, err
	}
	if len(raw) == 0 || string(raw) == "null" {
		return []string{}, nil
	}
	var paths []string
	if err := json.Unmarshal(raw, &paths); err != nil {
		return nil, fmt.Errorf("unmarshal directory templates: %w", err)
	}
	return paths, nil
}

func (s *DepartmentService) UpdateDirectoryTemplates(ctx context.Context, departmentID string, paths []string) error {
	pgDeptID, err := db.ParseUUID(departmentID)
	if err != nil {
		return err
	}
	for _, p := range paths {
		if !strings.HasPrefix(p, "/data/") {
			return fmt.Errorf("path must start with /data/: %s", p)
		}
		if strings.Contains(p, "..") {
			return fmt.Errorf("path must not contain '..': %s", p)
		}
	}
	data, err := json.Marshal(paths)
	if err != nil {
		return fmt.Errorf("marshal directory templates: %w", err)
	}
	return s.queries.UpdateDepartmentDirectoryTemplates(ctx, sqlc.UpdateDepartmentDirectoryTemplatesParams{
		ID:                 pgDeptID,
		DirectoryTemplates: data,
	})
}

func (s *DepartmentService) SyncSkills(ctx context.Context, departmentID string) (*handlers.SyncResultDTO, error) {
	pgDeptID, err := db.ParseUUID(departmentID)
	if err != nil {
		return nil, err
	}

	// Get department's skill templates.
	templates, err := s.queries.ListDepartmentSkillTemplates(ctx, pgDeptID)
	if err != nil {
		return nil, err
	}

	// Get bots in the department.
	bots, err := s.queries.ListDepartmentBots(ctx, pgDeptID)
	if err != nil {
		return nil, err
	}

	result := &handlers.SyncResultDTO{TotalBots: len(bots)}

	for _, bot := range bots {
		botIDStr := uuidToString(bot.ID)

		// Get existing installs for this bot.
		installs, err := s.queries.ListBotSkillInstalls(ctx, bot.ID)
		if err != nil {
			result.Errors = append(result.Errors, handlers.BotSyncError{
				BotID:   botIDStr,
				Message: fmt.Sprintf("list installs: %v", err),
			})
			continue
		}

		// Build set of already-installed template IDs and customized ones.
		installedSet := make(map[string]bool)
		customizedSet := make(map[string]bool)
		for _, inst := range installs {
			tid := uuidToString(inst.TemplateID)
			installedSet[tid] = true
			if inst.Customized {
				customizedSet[tid] = true
			}
		}

		client, clientErr := s.manager.MCPClient(ctx, botIDStr)
		botInstalled := 0

		for _, tmpl := range templates {
			tmplIDStr := uuidToString(tmpl.ID)

			// Skip if customized by user.
			if customizedSet[tmplIDStr] {
				result.Skipped++
				continue
			}

			// Skip if already installed (not customized — could update, but for now skip).
			if installedSet[tmplIDStr] {
				result.Skipped++
				continue
			}

			if clientErr != nil {
				result.Errors = append(result.Errors, handlers.BotSyncError{
					BotID:   botIDStr,
					Message: fmt.Sprintf("container not reachable: %v", clientErr),
				})
				break
			}

			// Use slug as skill name (same approach as Install handler).
			skillName := tmpl.Slug
			dirPath := path.Join(skillsDirPath, skillName)
			if err := client.Mkdir(ctx, dirPath); err != nil {
				result.Errors = append(result.Errors, handlers.BotSyncError{
					BotID:   botIDStr,
					Message: fmt.Sprintf("mkdir %s: %v", dirPath, err),
				})
				continue
			}
			filePath := path.Join(dirPath, "SKILL.md")
			if err := client.WriteFile(ctx, filePath, []byte(tmpl.Content)); err != nil {
				result.Errors = append(result.Errors, handlers.BotSyncError{
					BotID:   botIDStr,
					Message: fmt.Sprintf("write %s: %v", filePath, err),
				})
				continue
			}

			// Create install record.
			_, err := s.queries.CreateBotSkillInstall(ctx, sqlc.CreateBotSkillInstallParams{
				BotID:            bot.ID,
				TemplateID:       tmpl.ID,
				InstalledVersion: tmpl.Version,
				SkillName:        skillName,
			})
			if err != nil {
				// Conflict means already installed via another path — skip.
				if strings.Contains(err.Error(), "bot_skill_installs_bot_template") {
					result.Skipped++
					continue
				}
				result.Errors = append(result.Errors, handlers.BotSyncError{
					BotID:   botIDStr,
					Message: fmt.Sprintf("create install: %v", err),
				})
				continue
			}
			botInstalled++
		}
		result.Installed += botInstalled
	}

	return result, nil
}

func (s *DepartmentService) SyncDirectories(ctx context.Context, departmentID string) (*handlers.SyncResultDTO, error) {
	pgDeptID, err := db.ParseUUID(departmentID)
	if err != nil {
		return nil, err
	}

	// Get directory templates.
	paths, err := s.GetDirectoryTemplates(ctx, departmentID)
	if err != nil {
		return nil, err
	}

	// Get bots in the department.
	bots, err := s.queries.ListDepartmentBots(ctx, pgDeptID)
	if err != nil {
		return nil, err
	}

	result := &handlers.SyncResultDTO{TotalBots: len(bots)}

	for _, bot := range bots {
		botIDStr := uuidToString(bot.ID)

		client, err := s.manager.MCPClient(ctx, botIDStr)
		if err != nil {
			result.Errors = append(result.Errors, handlers.BotSyncError{
				BotID:   botIDStr,
				Message: fmt.Sprintf("container not reachable: %v", err),
			})
			continue
		}

		botCreated := 0
		for _, p := range paths {
			if err := client.Mkdir(ctx, p); err != nil {
				result.Errors = append(result.Errors, handlers.BotSyncError{
					BotID:   botIDStr,
					Message: fmt.Sprintf("mkdir %s: %v", p, err),
				})
				continue
			}
			botCreated++
		}
		result.Installed += botCreated
	}

	return result, nil
}

func departmentToDTO(d sqlc.Department) handlers.DepartmentDTO {
	return handlers.DepartmentDTO{
		ID:          uuidToString(d.ID),
		Name:        d.Name,
		Description: d.Description,
		ParentID:    uuidToString(d.ParentID),
	}
}
