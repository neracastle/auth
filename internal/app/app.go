package app

import (
	"context"
	"errors"
	"log"
	"net"

	"github.com/IBM/sarama"
	"github.com/neracastle/go-libs/pkg/kafka"
	"github.com/neracastle/go-libs/pkg/sys/logger"
	"golang.org/x/exp/slog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"

	grpc_server "github.com/neracastle/auth/internal/grpc-server"
	"github.com/neracastle/auth/internal/grpc-server/interceptors"
	"github.com/neracastle/auth/pkg/user_v1"
	sharedinters "github.com/neracastle/auth/pkg/user_v1/auth/grpc-interceptors"
)

// App приложение
type App struct {
	grpc        *grpc.Server
	srvProvider *serviceProvider
}

// NewApp новый экземпляр приложения
func NewApp(ctx context.Context) *App {
	app := &App{srvProvider: newServiceProvider()}
	app.init(ctx)
	return app
}

func (a *App) init(ctx context.Context) {
	lg := logger.SetupLogger(a.srvProvider.Config().Env)
	a.grpc = grpc.NewServer(
		grpc.Creds(insecure.NewCredentials()),
		grpc.ChainUnaryInterceptor(
			interceptors.RequestIDInterceptor,
			interceptors.NewLoggerInterceptor(lg),
			sharedinters.NewAccessInterceptor([]string{
				user_v1.UserV1_Get_FullMethodName,
				user_v1.UserV1_Update_FullMethodName,
				user_v1.UserV1_Delete_FullMethodName,
			}, a.srvProvider.Config().JWT.SecretKey)),
	)

	reflection.Register(a.grpc)
	user_v1.RegisterUserV1Server(a.grpc, grpc_server.NewServer(a.srvProvider.UsersService(ctx)))
}

// Start запускает сервис на прием запросов
func (a *App) Start() error {
	conn, err := net.Listen("tcp", a.srvProvider.Config().GRPC.Address())
	if err != nil {
		return err
	}

	log.Printf("UserAPI service started on %s\n", a.srvProvider.Config().GRPC.Address())

	if err = a.grpc.Serve(conn); err != nil {
		return err
	}

	return nil
}

// StartHTTP запускает http сервис на прием запросов
func (a *App) StartHTTP() error {
	log.Printf("UserAPI HTTP started on %s\n", a.srvProvider.Config().HTTP.Address())

	if err := a.srvProvider.HTTPServer().ListenAndServe(); err != nil {
		return err
	}

	return nil
}

// RunTopicLogger запускает прослушку сообщений кафки и просто их логгирует
func (a *App) RunTopicLogger(ctx context.Context) {
	topic := a.srvProvider.Config().NewUsersTopic
	lg := logger.SetupLogger(a.srvProvider.Config().Env)
	lg = lg.With(slog.String("topic", topic))

	cons := a.srvProvider.KafkaConsumer()
	cons.GroupHandler().SetMessageHandler(func(ctx context.Context, msg *sarama.ConsumerMessage) error {
		lg.Info("received message", slog.String("value", string(msg.Value)))

		return nil
	})

	for {
		err := cons.RunConsume(ctx, topic)
		if err != nil {
			if errors.Is(kafka.ErrHandlerError, err) {
				lg.Info("consumer handler error", slog.String("error", err.Error()))
				continue
			}

			lg.Info("consumer error", slog.String("error", err.Error()))
			break
		}
	}

}

// StartSwaggerServer запускает сервер со swagger-документацией
func (a *App) StartSwaggerServer() error {
	log.Printf("Swagger server started on %s\n", a.srvProvider.Config().Swagger.Address())
	if err := a.srvProvider.SwaggerServer().ListenAndServe(); err != nil {
		return err
	}

	return nil
}

// Shutdown мягко закрывает все соединения и службы
func (a *App) Shutdown(ctx context.Context) {

	allClosed := make(chan struct{})
	go func() {
		_ = a.srvProvider.consumer.Close()
		_ = a.srvProvider.producer.Close()
		_ = a.srvProvider.dbc.Close()
		_ = a.srvProvider.httpServer.Close()
		_ = a.srvProvider.swaggerServer.Close()
		a.grpc.GracefulStop()
		close(allClosed)
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case <-allClosed:
			return
		}
	}
}
