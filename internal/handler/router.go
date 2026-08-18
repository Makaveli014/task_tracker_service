package handler

import (
	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"

	"TestTask_Bazis/internal/handler/middleware"
	"TestTask_Bazis/internal/repository"
	"TestTask_Bazis/internal/service"
)

func SetupRouter(
	authSvc *service.AuthService,
	teamSvc *service.TeamService,
	taskSvc *service.TaskService,
	analyticsRepo *repository.AnalyticsRepository,
	redisClient *redis.Client,
	rpm int,
) *chi.Mux {
	r := chi.NewRouter()

	r.Use(chimw.Logger)
	r.Use(chimw.Recoverer)
	r.Use(chimw.RequestID)
	r.Use(middleware.MetricsMiddleware)

	r.Handle("/metrics", promhttp.Handler())

	authHandler := NewAuthHandler(authSvc)
	teamHandler := NewTeamHandler(teamSvc)
	taskHandler := NewTaskHandler(taskSvc, redisClient)
	analyticsHandler := NewAnalyticsHandler(analyticsRepo)

	r.Route("/api/v1", func(r chi.Router) {
		r.Post("/register", authHandler.Register)
		r.Post("/login", authHandler.Login)

		r.Group(func(r chi.Router) {
			r.Use(middleware.AuthMiddleware(authSvc))
			r.Use(middleware.RateLimitMiddleware(redisClient, rpm))

			r.Post("/teams", teamHandler.Create)
			r.Get("/teams", teamHandler.List)
			r.Post("/teams/{id}/invite", teamHandler.Invite)

			r.Post("/tasks", taskHandler.Create)
			r.Get("/tasks", taskHandler.List)
			r.Put("/tasks/{id}", taskHandler.Update)
			r.Get("/tasks/{id}/history", taskHandler.GetHistory)
			r.Post("/tasks/{id}/comments", taskHandler.AddComment)
			r.Get("/tasks/{id}/comments", taskHandler.GetComments)

			r.Get("/analytics/team-stats", analyticsHandler.TeamStats)
			r.Get("/analytics/top-creators", analyticsHandler.TopCreators)
			r.Get("/analytics/integrity-check", analyticsHandler.IntegrityCheck)
		})
	})

	return r
}
