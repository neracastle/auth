package app

import (
	"context"
	"log"
	"time"

	"github.com/IBM/sarama"
	redigo "github.com/gomodule/redigo/redis"
	"github.com/neracastle/go-libs/pkg/db"
	"github.com/neracastle/go-libs/pkg/db/pg"
	"github.com/neracastle/go-libs/pkg/kafka"
	"github.com/neracastle/go-libs/pkg/redis"
	redislib "github.com/neracastle/go-libs/pkg/redis/redis"
	"github.com/neracastle/go-libs/pkg/sys/logger"
	"github.com/neracastle/go-libs/pkg/sys/rate_limiter"
	"github.com/neracastle/go-libs/pkg/sys/tracer"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/exp/slog"

	"github.com/neracastle/auth/internal/config"
	"github.com/neracastle/auth/internal/repository/action"
	actionsPg "github.com/neracastle/auth/internal/repository/action/postgres"
	"github.com/neracastle/auth/internal/repository/user"
	usersPg "github.com/neracastle/auth/internal/repository/user/postgres"
	usersRedis "github.com/neracastle/auth/internal/repository/user/redis"
	"github.com/neracastle/auth/internal/usecases"
)

type serviceProvider struct {
	conf           *config.Config
	logger         *slog.Logger
	usecaseService usecases.UserService
	usersRepo      user.Repository
	usersCache     user.Cache
	actionsRepo    action.Repository
	dbc            db.Client
	redis          redis.Client
	consumer       kafka.Consumer
	producer       sarama.SyncProducer
	rateLimiter    *rate_limiter.RateLimiter
	queryLogger    db.QueryLogger
}

func newServiceProvider() *serviceProvider {
	return &serviceProvider{}
}

func (sp *serviceProvider) Config() config.Config {
	if sp.conf == nil {
		cfg := config.MustLoad()
		sp.conf = &cfg
	}

	return *sp.conf
}

func (sp *serviceProvider) DbClient(ctx context.Context) db.Client {
	if sp.dbc == nil {
		client, err := pg.NewClient(ctx, sp.Config().Postgres.DSN())
		if err != nil {
			log.Fatalf("failed to connect to pg: %v", err)
		}

		err = client.DB().Ping(ctx)
		if err != nil {
			log.Fatalf("failed ping to pg: %v", err)
		}

		client.DB().SetQueryLogger(sp.QueryLogger())
		sp.dbc = client
	}

	return sp.dbc
}

func (sp *serviceProvider) RedisClient() redis.Client {
	if sp.redis == nil {
		pool := &redigo.Pool{
			MaxIdle:     sp.Config().Redis.MaxIdle,
			IdleTimeout: time.Duration(sp.Config().Redis.IdleTimeout),
			DialContext: func(ctx context.Context) (redigo.Conn, error) {
				return redigo.DialContext(ctx, "tcp", sp.Config().Redis.Address())
			},
		}

		sp.redis = redislib.NewClient(pool)
	}

	return sp.redis
}

func (sp *serviceProvider) UsersRepository(ctx context.Context) user.Repository {
	if sp.usersRepo == nil {
		sp.usersRepo = usersPg.New(sp.DbClient(ctx))
	}

	return sp.usersRepo
}

func (sp *serviceProvider) UsersCache() user.Cache {
	if sp.usersCache == nil {
		sp.usersCache = usersRedis.New(sp.RedisClient())
	}

	return sp.usersCache
}

func (sp *serviceProvider) ActionsRepository(ctx context.Context) action.Repository {
	if sp.actionsRepo == nil {
		sp.actionsRepo = actionsPg.New(sp.DbClient(ctx))
	}

	return sp.actionsRepo
}

func (sp *serviceProvider) UsersService(ctx context.Context) usecases.UserService {
	if sp.usecaseService == nil {
		sp.usecaseService = usecases.NewService(
			sp.UsersRepository(ctx),
			sp.UsersCache(),
			sp.ActionsRepository(ctx),
			sp.DbClient(ctx).DB(),
			sp.KafkaProducer(),
			sp.KafkaConsumer(),
			usecases.Config{
				CacheTTL:        sp.Config().UsersCacheTTL,
				NewUserTopic:    sp.Config().NewUsersTopic,
				SecretKey:       sp.Config().JWT.SecretKey,
				AccessDuration:  sp.Config().JWT.AccessDuration,
				RefreshDuration: sp.Config().JWT.RefreshDuration,
			})
	}

	return sp.usecaseService
}

func (sp *serviceProvider) KafkaConsumer() kafka.Consumer {
	if sp.consumer == nil {
		cl, err := kafka.NewConsumer(sp.Config().Kafka.Brokers, sp.Config().Kafka.GroupID, sp.Config().Kafka.SaramaConfig())
		if err != nil {
			log.Fatalf("failed to create kafka consumer: %v", err)
		}

		sp.consumer = cl
	}

	return sp.consumer
}

func (sp *serviceProvider) KafkaProducer() sarama.SyncProducer {
	if sp.producer == nil {
		producer, err := sarama.NewSyncProducer(sp.Config().Kafka.Brokers, sp.Config().Kafka.SaramaConfig())
		if err != nil {
			log.Fatalf("failed to create kafka producer: %v", err)
		}

		sp.producer = producer
	}

	return sp.producer
}

func (sp serviceProvider) Logger() *slog.Logger {
	if sp.logger == nil {
		sp.logger = logger.SetupLogger(logger.Env(sp.Config().Env))
	}

	return sp.logger
}

func (sp *serviceProvider) QueryLogger() db.QueryLogger {
	if sp.queryLogger == nil {
		sp.queryLogger = func(ctx context.Context, q db.Query, args ...interface{}) db.LogFlush {
			sp.Logger().Debug("db query", slog.String("method", q.Name), slog.String("query", q.QueryRaw))
			var span trace.Span
			ctx, span = tracer.Span(ctx, q.Name)
			span.SetAttributes(attribute.String("query", q.QueryRaw))
			return tracer.SpanFlush(span)
		}
	}

	return sp.queryLogger
}

func (sp *serviceProvider) RateLimiter() *rate_limiter.RateLimiter {
	if sp.rateLimiter == nil {
		sp.rateLimiter = rate_limiter.New(sp.Config().RateLimiter.Limit, sp.Config().RateLimiter.Period)
	}

	return sp.rateLimiter
}
