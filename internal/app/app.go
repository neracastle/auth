package app

import (
	"context"
	"errors"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/IBM/sarama"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/neracastle/go-libs/pkg/kafka"
	"github.com/neracastle/go-libs/pkg/sys/logger"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"golang.org/x/exp/slog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"

	embed "github.com/neracastle/auth/api/user_v1"
	grpc_server "github.com/neracastle/auth/internal/grpc-server"
	"github.com/neracastle/auth/internal/grpc-server/interceptors"
	"github.com/neracastle/auth/internal/tracer"
	"github.com/neracastle/auth/pkg/user_v1"
	sharedinters "github.com/neracastle/auth/pkg/user_v1/auth/grpc-interceptors"
)

// App приложение
type App struct {
	grpc             *grpc.Server
	httpServer       *http.Server
	swaggerServer    *http.Server
	prometheusServer *http.Server
	traceExporter    *otlptrace.Exporter
	srvProvider      *serviceProvider
}

// NewApp новый экземпляр приложения
func NewApp(ctx context.Context) *App {
	app := &App{srvProvider: newServiceProvider()}
	app.init(ctx)
	return app
}

func (a *App) init(ctx context.Context) {
	lg := logger.SetupLogger(a.srvProvider.Config().Env)

	a.initTracing(ctx, "auth-service")
	a.grpc = grpc.NewServer(
		grpc.Creds(insecure.NewCredentials()),
		grpc.StatsHandler(otelgrpc.NewServerHandler()),
		grpc.ChainUnaryInterceptor(
			interceptors.MetricsInterceptor,
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

func (a *App) initTracing(ctx context.Context, serviceName string) {
	//экспортер в jaeger
	exporter, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithInsecure(),
		otlptracegrpc.WithEndpoint(a.srvProvider.Config().Trace.JaegerGRPCAddress))
	if err != nil {
		log.Fatalf("failed to create trace exporter: %v", err)
	}
	a.traceExporter = exporter

	//собиратель трейсов
	r, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(semconv.SchemaURL, semconv.ServiceNameKey.String(serviceName)),
	)
	if err != nil {
		log.Fatalf("failed to create trace provider: %v", err)
	}

	traceProvider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter, sdktrace.WithExportTimeout(time.Second*time.Duration(a.srvProvider.Config().Trace.BatchTimeout))),
		sdktrace.WithResource(r))

	//пробрасываем провайдер для исп. в других местах приложения
	tracer.Init(traceProvider.Tracer(serviceName))
	//регистрируем глобально
	otel.SetTracerProvider(traceProvider)
	prop := propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{})
	otel.SetTextMapPropagator(prop)
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

	if a.httpServer == nil {
		mux := runtime.NewServeMux()
		opts := []grpc.DialOption{
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		}
		_ = user_v1.RegisterUserV1HandlerFromEndpoint(context.Background(), mux, a.srvProvider.Config().GRPC.Address(), opts)

		a.httpServer = &http.Server{
			Addr:              a.srvProvider.Config().HTTP.Address(),
			Handler:           NewCORSMux(mux),
			ReadHeaderTimeout: 5 * time.Second, //защита от Slowloris Attack
		}
	}

	if err := a.httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
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

	if a.swaggerServer == nil {
		mux := http.NewServeMux()
		mux.Handle("/", embed.NewSwaggerFS(a.srvProvider.Config().HTTP.Port))

		a.swaggerServer = &http.Server{
			Addr:              a.srvProvider.Config().Swagger.Address(),
			Handler:           mux,
			ReadHeaderTimeout: 5 * time.Second, //защита от Slowloris Attack
		}
	}

	if err := a.swaggerServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	return nil
}

// StartPrometheusServer запускает сервер prometheus
func (a *App) StartPrometheusServer() error {
	log.Printf("Prometheus server started on %s\n", a.srvProvider.Config().Prometheus.Address())

	if a.prometheusServer == nil {
		mux := http.NewServeMux()
		mux.Handle("/metrics", promhttp.Handler())

		a.prometheusServer = &http.Server{
			Addr:    a.srvProvider.Config().Prometheus.Address(),
			Handler: mux,
		}
	}

	if err := a.prometheusServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
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
		_ = a.httpServer.Close()
		_ = a.swaggerServer.Close()
		a.grpc.GracefulStop()
		_ = a.traceExporter.Shutdown(ctx)
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
