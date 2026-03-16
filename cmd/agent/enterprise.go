//go:build enterprise

package main

import (
	"context"
	"log/slog"
	"time"

	"go.uber.org/fx"

	"github.com/memohai/memoh/internal/channel/inbound"
	"github.com/memohai/memoh/internal/conversation/flow"
	"github.com/memohai/memoh/internal/enterprise"
	"github.com/memohai/memoh/internal/handlers"
	"github.com/memohai/memoh/internal/mcp"
	authz "github.com/memohai/memoh/internal/rbac"

	"github.com/memohai/memoh/internal/config"
	dbsqlc "github.com/memohai/memoh/internal/db/sqlc"
	"github.com/memohai/memoh/internal/schedule"
)

// enterpriseModule provides all enterprise features (RBAC, audit, budget, departments, etc.)
// when built with -tags enterprise.
var enterpriseModule = fx.Module("enterprise",
	fx.Provide(
		// Enterprise shared dependencies
		provideAuditLogger,
		provideEnterpriseRBACResolver,

		// Enterprise feature handlers (F1-F7)
		provideServerHandler(provideDepartmentHandler),
		provideServerHandler(provideAuditHandler),
		provideServerHandler(provideCockpitHandler),
		provideServerHandler(provideHandsHandler),
		provideServerHandler(provideModelRoutingHandler),
		provideServerHandler(provideCostTrackingHandler),
	),
	fx.Invoke(
		injectBudgetChecker,
		injectModelRouter,
		injectCockpitCron,
	),
)

// ---------------------------------------------------------------------------
// hook injectors — wire enterprise logic into upstream hooks
// ---------------------------------------------------------------------------

func injectBudgetChecker(processor *inbound.ChannelInboundProcessor, log *slog.Logger, queries *dbsqlc.Queries) {
	costSvc := enterprise.NewCostTrackingService(log, queries)
	processor.SetChatPreCheck(func(ctx context.Context, botID, userID string) error {
		adapter := enterprise.NewBudgetCheckerAdapter(log, costSvc, queries)
		result, err := adapter.CheckChatBudget(ctx, botID, userID)
		if err != nil {
			return err
		}
		if !result.Allowed {
			return &inbound.ChatPreCheckDenied{Message: "⚠️ 预算已超限，请联系管理员。"}
		}
		return nil
	})
}

func injectModelRouter(resolver *flow.Resolver, log *slog.Logger, queries *dbsqlc.Queries) {
	adapter := enterprise.NewModelRouterAdapter(log, queries)
	resolver.SetModelOverride(func(ctx context.Context, botID, modelID string) (string, error) {
		entRoutes, err := adapter.ListEnabledRoutes(ctx, botID)
		if err != nil {
			return "", err
		}
		// Convert enterprise route entries to flow route entries for selection.
		flowRoutes := make([]flow.ModelRouteEntry, len(entRoutes))
		for i, r := range entRoutes {
			flowRoutes[i] = flow.ModelRouteEntry{
				ModelID:        r.ModelID,
				Priority:       r.Priority,
				ComplexityTier: r.ComplexityTier,
				IsEnabled:      r.IsEnabled,
			}
		}
		if overrideID, ok := flow.SelectModelRoute(flowRoutes, ""); ok {
			return overrideID, nil
		}
		return "", nil // no override
	})
}

func injectCockpitCron(log *slog.Logger, scheduleService *schedule.Service, queries *dbsqlc.Queries) {
	reportGen := enterprise.NewCockpitReportGenerator(log, queries)
	_ = scheduleService.AddSystemJob("cockpit-daily-report", "0 1 * * *", func() {
		reportCtx := context.Background()
		yesterday := time.Now().UTC().AddDate(0, 0, -1)
		bots, err := queries.ListActiveBotIDsWithOwner(reportCtx)
		if err != nil {
			log.Error("cockpit daily report: failed to list bots", slog.String("error", err.Error()))
			return
		}
		for _, bot := range bots {
			botID := bot.ID.String()
			ownerID := bot.OwnerUserID.String()
			if err := reportGen.GenerateDailyReport(reportCtx, botID, ownerID, yesterday); err != nil {
				log.Error("cockpit daily report: generation failed",
					slog.String("bot_id", botID),
					slog.String("error", err.Error()),
				)
			}
		}
		log.Info("cockpit daily report: completed", slog.Int("bots", len(bots)))
	})
}

// ---------------------------------------------------------------------------
// enterprise feature providers (F1-F7)
// ---------------------------------------------------------------------------

func provideAuditLogger(log *slog.Logger, queries *dbsqlc.Queries) *enterprise.AuditLogger {
	return enterprise.NewAuditLogger(log, queries)
}

func provideEnterpriseRBACResolver(queries *dbsqlc.Queries) *authz.Resolver {
	return authz.NewResolver(authz.NewSQLCStore(queries))
}

// mcpManagerAdapter wraps *mcp.Manager to satisfy enterprise.ContainerClientProvider.
type mcpManagerAdapter struct {
	m *mcp.Manager
}

func (a *mcpManagerAdapter) MCPClient(ctx context.Context, botID string) (enterprise.ContainerClient, error) {
	return a.m.MCPClient(ctx, botID)
}

func provideDepartmentHandler(log *slog.Logger, queries *dbsqlc.Queries, al *enterprise.AuditLogger, resolver *authz.Resolver, mgr *mcp.Manager) *handlers.DepartmentHandler {
	svc := enterprise.NewDepartmentService(log, queries, &mcpManagerAdapter{m: mgr})
	botAuthMw := enterprise.RequireBotPermission(
		enterprise.NewBotAuthorizerAdapter(resolver),
		authz.ResourceMembers,
		enterprise.ResolveMethodAction(authz.ActionRead, authz.ActionWrite),
	)
	globalAuthMw := enterprise.RequireRole(queries, log, "admin")
	return handlers.NewDepartmentHandler(svc, botAuthMw,
		handlers.WithDepartmentAudit(al),
		handlers.WithDepartmentGlobalMiddleware(globalAuthMw),
	)
}

func provideAuditHandler(log *slog.Logger, queries *dbsqlc.Queries, resolver *authz.Resolver) *handlers.AuditHandler {
	svc := enterprise.NewAuditQueryService(log, queries)
	authMw := enterprise.RequireBotPermission(
		enterprise.NewBotAuthorizerAdapter(resolver),
		authz.ResourceSettings,
		enterprise.ResolveMethodAction(authz.ActionRead, authz.ActionRead),
	)
	return handlers.NewAuditHandler(svc, authMw)
}

func provideCockpitHandler(log *slog.Logger, queries *dbsqlc.Queries, resolver *authz.Resolver) *handlers.CockpitHandler {
	svc := enterprise.NewCockpitService(log, queries)
	reportGen := enterprise.NewCockpitReportGenerator(log, queries)
	authMw := enterprise.RequireBotPermission(
		enterprise.NewBotAuthorizerAdapter(resolver),
		authz.ResourceSettings,
		enterprise.ResolveCockpitAction,
	)
	return handlers.NewCockpitHandler(svc, reportGen, authMw)
}

func provideHandsHandler(log *slog.Logger, queries *dbsqlc.Queries, flowResolver *flow.Resolver, cfg config.Config, al *enterprise.AuditLogger, resolver *authz.Resolver) *handlers.HandsHandler {
	chatExec := enterprise.NewResolverChatExecutor(flowResolver)
	svc := enterprise.NewHandsService(log, queries, chatExec, cfg.Auth.JWTSecret)
	authMw := enterprise.RequireBotPermission(
		enterprise.NewBotAuthorizerAdapter(resolver),
		authz.ResourceTools,
		enterprise.ResolveHandsAction,
	)
	return handlers.NewHandsHandler(svc, authMw, handlers.WithAuditLogger(al))
}

func provideModelRoutingHandler(log *slog.Logger, queries *dbsqlc.Queries, al *enterprise.AuditLogger, resolver *authz.Resolver) *handlers.ModelRoutingHandler {
	svc := enterprise.NewModelRoutingService(log, queries)
	authMw := enterprise.RequireBotPermission(
		enterprise.NewBotAuthorizerAdapter(resolver),
		authz.ResourceSettings,
		enterprise.ResolveMethodAction(authz.ActionRead, authz.ActionWrite),
	)
	return handlers.NewModelRoutingHandler(svc, authMw, handlers.WithModelRoutingAudit(al))
}

func provideCostTrackingHandler(log *slog.Logger, queries *dbsqlc.Queries, al *enterprise.AuditLogger) *handlers.CostTrackingHandler {
	svc := enterprise.NewCostTrackingService(log, queries)
	adminMw := enterprise.RequireRole(queries, log, "admin")
	return handlers.NewCostTrackingHandler(svc, adminMw, handlers.WithCostTrackingAudit(al))
}
