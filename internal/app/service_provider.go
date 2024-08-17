package app

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/IBM/sarama"
	redigo "github.com/gomodule/redigo/redis"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/neracastle/go-libs/pkg/db"
	"github.com/neracastle/go-libs/pkg/db/pg"
	"github.com/neracastle/go-libs/pkg/kafka"
	"github.com/neracastle/go-libs/pkg/redis"
	redislib "github.com/neracastle/go-libs/pkg/redis/redis"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	api "github.com/neracastle/auth/api/user_v1"
	"github.com/neracastle/auth/internal/config"
	"github.com/neracastle/auth/internal/repository/action"
	actionsPg "github.com/neracastle/auth/internal/repository/action/postgres"
	"github.com/neracastle/auth/internal/repository/user"
	usersPg "github.com/neracastle/auth/internal/repository/user/postgres"
	usersRedis "github.com/neracastle/auth/internal/repository/user/redis"
	"github.com/neracastle/auth/internal/usecases"
	"github.com/neracastle/auth/pkg/user_v1"
)

type serviceProvider struct {
	conf           *config.Config
	usecaseService usecases.UserService
	usersRepo      user.Repository
	usersCache     user.Cache
	actionsRepo    action.Repository
	dbc            db.Client
	redis          redis.Client
	consumer       kafka.Consumer
	producer       sarama.SyncProducer
	httpServer     *http.Server
	swaggerServer  *http.Server
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
				CacheTTL:        time.Second * time.Duration(sp.Config().UsersCacheTTL),
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

func (sp *serviceProvider) HTTPServer() *http.Server {
	if sp.httpServer == nil {
		mux := runtime.NewServeMux()
		opts := []grpc.DialOption{
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		}
		_ = user_v1.RegisterUserV1HandlerFromEndpoint(context.Background(), mux, sp.Config().GRPC.Address(), opts)

		sp.httpServer = &http.Server{
			Addr:              sp.Config().HTTP.Address(),
			Handler:           NewCORSMux(mux),
			ReadHeaderTimeout: 5 * time.Second, //защита от Slowloris Attack
		}
	}

	return sp.httpServer
}

func (sp *serviceProvider) SwaggerServer() *http.Server {
	if sp.swaggerServer == nil {
		mux := http.NewServeMux()
		mux.Handle("/", api.NewSwaggerFS(sp.Config().HTTP.Port))

		sp.swaggerServer = &http.Server{
			Addr:              sp.Config().Swagger.Address(),
			Handler:           mux,
			ReadHeaderTimeout: 5 * time.Second, //защита от Slowloris Attack
		}
	}

	return sp.swaggerServer
}
