package graph

import (
	"context"
	"errors"
	"net/http"
	"os"
	"time"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/handler/lru"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/google/uuid"
	"github.com/jordanknott/project-citadel/internal/auth"
	"github.com/jordanknott/project-citadel/internal/config"
	"github.com/jordanknott/project-citadel/internal/db"
	"github.com/vektah/gqlparser/v2/gqlerror"
)

// NewHandler returns a new graphql endpoint handler.
func NewHandler(config config.AppConfig, repo db.Repository) http.Handler {
	srv := handler.New(NewExecutableSchema(Config{
		Resolvers: &Resolver{
			Config:     config,
			Repository: repo,
		},
	}))
	srv.AddTransport(transport.Websocket{
		KeepAlivePingInterval: 10 * time.Second,
	})
	srv.AddTransport(transport.Options{})
	srv.AddTransport(transport.GET{})
	srv.AddTransport(transport.POST{})
	srv.AddTransport(transport.MultipartForm{})

	srv.SetQueryCache(lru.New(1000))

	srv.Use(extension.AutomaticPersistedQuery{
		Cache: lru.New(100),
	})
	if isProd := os.Getenv("PRODUCTION") == "true"; isProd {
		srv.Use(extension.FixedComplexityLimit(10))
	} else {
		srv.Use(extension.Introspection{})
	}
	return srv
}

// NewPlaygroundHandler returns a new GraphQL Playground handler.
func NewPlaygroundHandler(endpoint string) http.Handler {
	return playground.Handler("GraphQL Playground", endpoint)
}

func GetUserID(ctx context.Context) (uuid.UUID, bool) {
	userID, ok := ctx.Value("userID").(uuid.UUID)
	return userID, ok
}

func GetUser(ctx context.Context) (uuid.UUID, auth.Role, bool) {
	userID, userOK := ctx.Value("userID").(uuid.UUID)
	role, roleOK := ctx.Value("org_role").(auth.Role)
	return userID, role, userOK && roleOK
}

func GetRestrictedMode(ctx context.Context) (auth.RestrictedMode, bool) {
	restricted, ok := ctx.Value("restricted_mode").(auth.RestrictedMode)
	return restricted, ok
}

func GetProjectRoles(ctx context.Context, r db.Repository, userID uuid.UUID, projectID uuid.UUID) (db.GetUserRolesForProjectRow, error) {
	return r.GetUserRolesForProject(ctx, db.GetUserRolesForProjectParams{UserID: userID, ProjectID: projectID})
}

func ConvertToRoleCode(r string) RoleCode {
	if r == RoleCodeAdmin.String() {
		return RoleCodeAdmin
	}
	if r == RoleCodeMember.String() {
		return RoleCodeMember
	}
	return RoleCodeObserver
}

func RequireTeamAdmin(ctx context.Context, r db.Repository, teamID uuid.UUID) error {
	userID, role, ok := GetUser(ctx)
	if !ok {
		return errors.New("internal: user id is not set")
	}
	teamRole, err := r.GetTeamRoleForUserID(ctx, db.GetTeamRoleForUserIDParams{UserID: userID, TeamID: teamID})
	isAdmin := role == auth.RoleAdmin
	isTeamAdmin := err == nil && ConvertToRoleCode(teamRole.RoleCode) == RoleCodeAdmin
	if !(isAdmin || isTeamAdmin) {
		return &gqlerror.Error{
			Message: "organization or team admin role required",
			Extensions: map[string]interface{}{
				"code": "2-400",
			},
		}
	} else if err != nil {
		return err
	}
	return nil
}
