package app

import (
	"context"
	"database/sql"
	"net/http"
	"time"

	"github.com/DrummDaddy/task_service/internal/auth"
	"github.com/DrummDaddy/task_service/internal/cache"
	"github.com/DrummDaddy/task_service/internal/config"
	"github.com/DrummDaddy/task_service/internal/core/ports"
	"github.com/DrummDaddy/task_service/internal/core/usecase"
	"github.com/DrummDaddy/task_service/internal/db"
	"github.com/DrummDaddy/task_service/internal/email"
	"github.com/DrummDaddy/task_service/internal/handler"
	"github.com/DrummDaddy/task_service/internal/infra/jwt"
	"github.com/DrummDaddy/task_service/internal/repo"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

type Server struct {
	Router chi.Router
	cfg    config.Config
	log    *zap.Logger
	mysql  *sql.DB
	redis  *redis.Client
}

func NewServer(cfg config.Config, log *zap.Logger) (*Server, error) {
	mysqlDB, err := db.OpenMySQL(cfg)
	if err != nil {
		return nil, err
	}
	redisClient, err := db.OpenRedis(cfg)
	if err != nil {
		_ = mysqlDB.Close()
		return nil, err
	}

	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(NewZapLoggerMiddleware(log))
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(30 * time.Second))

	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodOptions},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	r.Use(NewPrometheusMiddleware())

	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		if err := mysqlDB.PingContext(r.Context()); err != nil {
			http.Error(w, "db not ready", http.StatusServiceUnavailable)
			return
		}
		if err := redisClient.Ping(r.Context()).Err(); err != nil {
			http.Error(w, "redis not ready", http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	r.Handle("/metrics", promhttp.Handler())

	var userRepo ports.UserRepository = repo.NewUserRepo(mysqlDB)
	var teamRepo ports.TeamRepository = repo.NewTeamRepo(mysqlDB)
	var taskRepo ports.TaskRepository = repo.NewTaskRepo(mysqlDB)
	var reportsRepo ports.ReportsRepository = repo.NewReportsRepo(mysqlDB)

	var tasksCache ports.TasksCache = cache.NewTaskCache(redisClient, cfg.Cache.TasksTTL)
	var emailClient ports.EmailSender = email.NewClient(cfg.Email.BaseUrl, cfg.Email.Timeout)
	tokenIssuer := jwt.NewJWTIssuer([]byte(cfg.Auth.JWTSecret), cfg.Auth.AccessTokenTTL)

	authUC := usecase.NewAuthUsecase(
		usecase.AuthConfig{
			PasswordPepper:   cfg.Auth.PasswordPepper,
			PasswordHashCost: cfg.Auth.PasswordHashCost,
		},
		userRepo,
		tokenIssuer,
	)
	teamUC := usecase.NewTeamUsecase(teamRepo, emailClient)
	taskUC := usecase.NewTaskUsecase(taskRepo, teamRepo, tasksCache)
	reportUC := usecase.NewReportUsecase(reportsRepo)

	authH := handler.NewAuthHandler(authUC)
	teamH := handler.NewTeamHandler(teamUC)
	taskH := handler.NewTaskHandler(taskUC)
	reportH := handler.NewReportHandler(reportUC)

	r.Route("/api/v1", func(api chi.Router) {
		api.Get("/ping", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("pong"))
		})

		api.Post("/register", authH.Register)
		api.Post("/login", authH.Login)

		api.Group(func(pr chi.Router) {
			pr.Use(auth.Middleware([]byte(cfg.Auth.JWTSecret)))
			pr.Use(NewRateLimitMiddleware(redisClient, cfg.RateLimit.PerUserPerMinute))

			pr.Post("/teams", teamH.CreateTeam)
			pr.Get("/teams", teamH.ListTeams)
			pr.Post("/teams/{id}/invite", teamH.Invite)

			pr.Post("/tasks", taskH.Create)
			pr.Get("/tasks", taskH.List)
			pr.Put("/tasks/{id}", taskH.Update)
			pr.Get("/tasks/{id}/history", taskH.History)
			pr.Post("/tasks/{id}/comments", taskH.AddComment)
			pr.Get("/tasks/{id}/comments", taskH.ListComments)

			pr.Get("/reports/team-stats", reportH.TeamStats)
			pr.Get("/reports/top-creators", reportH.TopCreators)
			pr.Get("/reports/integrity/invalid-assignees", reportH.IntegrityInvalidAssignees)
		})
	})

	return &Server{Router: r, cfg: cfg, log: log, mysql: mysqlDB, redis: redisClient}, nil
}

func (s *Server) Close(ctx context.Context) error {
	_ = ctx
	if s.redis != nil {
		_ = s.redis.Close()
	}
	if s.mysql != nil {
		return s.mysql.Close()
	}
	return nil
}
