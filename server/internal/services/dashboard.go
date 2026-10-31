package services

import (
	"context"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/yannick2009/yily/internal/domain/model"
	"github.com/yannick2009/yily/internal/repository"
)

// dashboardListLimit caps how many items the dashboard's recent activity and projects lists return.
const dashboardListLimit = 10

// DashboardStats holds the aggregate counts shown at the top of the dashboard.
type DashboardStats struct {
	Projects     int `json:"projects"`     // Projects is the total number of projects.
	Environments int `json:"environments"` // Environments is the total number of environments.
	Variables    int `json:"variables"`    // Variables is the total number of variables (secrets).
	Tokens       int `json:"tokens"`       // Tokens is the total number of the user's personal access tokens.
}

// ProjectSummary is a project with its environment and variable counts.
type ProjectSummary struct {
	ID               uuid.UUID `json:"id"`                // Unique identifier of the project.
	Name             string    `json:"name"`              // Name of the project.
	EnvironmentCount int       `json:"environment_count"` // Number of environments in the project.
	VariableCount    int       `json:"variable_count"`    // Number of variables across the project's environments.
	UpdatedAt        time.Time `json:"updated_at"`        // Timestamp when the project was last updated.
}

// Dashboard aggregates the counts, recent activity and project summaries shown on the dashboard.
type Dashboard struct {
	Stats          DashboardStats     `json:"stats"`           // Stats holds the aggregate counts.
	RecentActivity []model.AuditEvent `json:"recent_activity"` // RecentActivity lists the latest audit events, newest first.
	Projects       []ProjectSummary   `json:"projects"`        // Projects lists the most recently updated projects with their counts.
}

// DashboardService defines the interface for dashboard aggregation.
type DashboardService interface {
	GetDashboard(ctx context.Context, userID uuid.UUID) (*Dashboard, error) // GetDashboard aggregates the dashboard data for a user.
}

// dashboardService implements DashboardService.
type dashboardService struct {
	projectRepository     repository.ProjectRepository
	environmentRepository repository.EnvironmentRepository
	variableRepository    repository.VariableRepository
	patRepository         repository.PATRepository
	audit                 AuditService
}

// NewDashboardService creates a new dashboardService with the provided repositories and audit service.
func NewDashboardService(projectRepository repository.ProjectRepository, environmentRepository repository.EnvironmentRepository, variableRepository repository.VariableRepository, patRepository repository.PATRepository, audit AuditService) DashboardService {
	return &dashboardService{projectRepository, environmentRepository, variableRepository, patRepository, audit}
}

// GetDashboard aggregates the dashboard data for the authenticated user.
func (s *dashboardService) GetDashboard(ctx context.Context, userID uuid.UUID) (*Dashboard, error) {
	projects, err := s.projectRepository.GetAll(ctx)
	if err != nil {
		return nil, err
	}

	tokenCount, err := s.patRepository.CountByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	recent, err := s.audit.List(ctx, AuditFilter{Limit: dashboardListLimit})
	if err != nil {
		return nil, err
	}

	summaries, environmentCount, variableCount, err := s.projectSummaries(ctx, projects)
	if err != nil {
		return nil, err
	}

	return &Dashboard{
		Stats: DashboardStats{
			Projects:     len(projects),
			Environments: environmentCount,
			Variables:    variableCount,
			Tokens:       int(tokenCount),
		},
		RecentActivity: recent.Events,
		Projects:       summaries,
	}, nil
}

// projectSummaries builds a summary per project (environment and variable counts), newest first,
// capped at dashboardListLimit, and returns the totals across all projects.
func (s *dashboardService) projectSummaries(ctx context.Context, projects []model.Project) ([]ProjectSummary, int, int, error) {
	summaries := make([]ProjectSummary, 0, len(projects))
	totalEnvironments, totalVariables := 0, 0
	for _, project := range projects {
		environmentCount, err := s.environmentRepository.CountByProjectID(ctx, project.ID)
		if err != nil {
			return nil, 0, 0, err
		}
		variableCount, err := s.variableRepository.CountByProjectID(ctx, project.ID)
		if err != nil {
			return nil, 0, 0, err
		}

		totalEnvironments += int(environmentCount)
		totalVariables += int(variableCount)

		summaries = append(summaries, ProjectSummary{
			ID:               project.ID,
			Name:             project.Name,
			EnvironmentCount: int(environmentCount),
			VariableCount:    int(variableCount),
			UpdatedAt:        project.UpdatedAt,
		})
	}
	sort.Slice(summaries, func(i, j int) bool { return summaries[i].UpdatedAt.After(summaries[j].UpdatedAt) })
	if len(summaries) > dashboardListLimit {
		summaries = summaries[:dashboardListLimit]
	}
	return summaries, totalEnvironments, totalVariables, nil
}
