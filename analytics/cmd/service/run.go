package service

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"

	"github.com/go-redis/redis/v8"
	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/wormhole-foundation/wormhole-explorer/analytics/cmd/token"
	"github.com/wormhole-foundation/wormhole-explorer/analytics/config"
	"github.com/wormhole-foundation/wormhole-explorer/analytics/consumer"
	"github.com/wormhole-foundation/wormhole-explorer/analytics/http"
	"github.com/wormhole-foundation/wormhole-explorer/analytics/http/vaa"
	"github.com/wormhole-foundation/wormhole-explorer/analytics/internal/metrics"
	"github.com/wormhole-foundation/wormhole-explorer/analytics/metric"
	"github.com/wormhole-foundation/wormhole-explorer/analytics/queue"
	wormscanNotionalCache "github.com/wormhole-foundation/wormhole-explorer/common/client/cache/notional"
	"github.com/wormhole-foundation/wormhole-explorer/common/client/parser"
	sqs_client "github.com/wormhole-foundation/wormhole-explorer/common/client/sqs"
	"github.com/wormhole-foundation/wormhole-explorer/common/dbutil"
	"github.com/wormhole-foundation/wormhole-explorer/common/domain"
	health "github.com/wormhole-foundation/wormhole-explorer/common/health"
	"github.com/wormhole-foundation/wormhole-explorer/common/logger"
	"go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/zap"
)

type exitCode int

func handleExit() {
	if r := recover(); r != nil {
		if e, ok := r.(exitCode); ok {
			os.Exit(int(e))
		}
		panic(r) // not an Exit, bubble up
	}
}

func Run() {
	defer handleExit()
	rootCtx, rootCtxCancel := context.WithCancel(context.Background())

	// load configuration
	config, err := config.New(rootCtx)
	if err != nil {
		log.Fatal("Error creating config", err)
	}

	// build logger
	logger := logger.New("wormhole-explorer-analytics", logger.WithLevel(config.LogLevel))
	logger.Info("starting analytics service...")

	// setup DB connection
	logger.Info("connecting to MongoDB...")
	db, err := dbutil.Connect(rootCtx, logger, config.MongodbURI, config.MongodbDatabase, false)
	if err != nil {
		logger.Fatal("failed to connect MongoDB", zap.Error(err))
	}

	// create influxdb client.
	logger.Info("initializing InfluxDB client...")
	influxCli := newInfluxClient(config.InfluxUrl, config.InfluxToken)
	influxCli.Options().SetBatchSize(100)

	// get health check functions.
	logger.Info("creating health check functions...")
	healthChecks, err := newHealthChecks(rootCtx, config, influxCli, db.Database)
	if err != nil {
		logger.Fatal("failed to create health checks", zap.Error(err))
	}

	//create notional cache
	logger.Info("initializing notional cache...")
	notionalCache, err := newNotionalCache(rootCtx, config, logger)
	if err != nil {
		logger.Fatal("failed to create notional cache", zap.Error(err))
	}

	// create prometheus client
	metrics := metrics.NewPrometheusMetrics(config.Environment)

	// create a parserVAAAPIClient
	parserVAAAPIClient, err := parser.NewParserVAAAPIClient(config.VaaPayloadParserTimeout,
		config.VaaPayloadParserURL, logger)
	if err != nil {
		logger.Fatal("failed to create parse vaa api client")
	}

	// create a token resolver
	tokenResolver := token.NewTokenResolver(parserVAAAPIClient, logger)

	// create a token provider
	tokenProvider := domain.NewTokenProvider(config.P2pNetwork)

	// create a metrics instance
	logger.Info("initializing metrics instance...")
	metric, err := metric.New(rootCtx, db.Database, influxCli, config.InfluxOrganization, config.InfluxBucketInfinite,
		config.InfluxBucket30Days, config.InfluxBucket24Hours, notionalCache, metrics, tokenResolver.GetTransferredTokenByVaa, tokenProvider, logger)
	if err != nil {
		logger.Fatal("failed to create metrics instance", zap.Error(err))
	}

	// create and start a vaa consumer.
	logger.Info("initializing vaa consumer...")
	vaaConsumeFunc := newVAAConsumeFunc(rootCtx, config, logger)
	vaaConsumer := consumer.New(vaaConsumeFunc, metric.Push, logger, config.P2pNetwork)
	vaaConsumer.Start(rootCtx)

	// create and start a notification consumer.
	logger.Info("initializing notification consumer...")
	notificationConsumeFunc := newNotificationConsumeFunc(rootCtx, config, logger)
	notificationConsumer := consumer.New(notificationConsumeFunc, metric.Push, logger, config.P2pNetwork)
	notificationConsumer.Start(rootCtx)

	// create and start server.
	logger.Info("initializing infrastructure server...")

	vaaRepository := vaa.NewRepository(db.Database, logger)
	vaaController := vaa.NewController(metric.Push, vaaRepository, logger)
	server := http.NewServer(logger, config.Port, config.PprofEnabled, vaaController, healthChecks...)
	server.Start()

	// Waiting for signal
	logger.Info("waiting for termination signal or context cancellation...")
	sigterm := make(chan os.Signal, 1)
	signal.Notify(sigterm, syscall.SIGINT, syscall.SIGTERM)
	select {
	case <-rootCtx.Done():
		logger.Warn("terminating (root context cancelled)")
	case signal := <-sigterm:
		logger.Info("terminating (signal received)", zap.String("signal", signal.String()))
	}

	logger.Info("cancelling root context...")
	rootCtxCancel()

	logger.Info("closing metrics client...")
	metric.Close()

	logger.Info("closing HTTP server...")
	server.Stop()

	logger.Info("closing MongoDB connection...")
	db.DisconnectWithTimeout(10 * time.Second)

	logger.Info("terminated successfully")
}

// Creates a callbacks depending on whether the execution is local (memory queue) or not (SQS queue)
func newVAAConsumeFunc(appCtx context.Context, config *config.Configuration, logger *zap.Logger) queue.ConsumeFunc {
	sqsConsumer, err := newSQSConsumer(appCtx, config, config.PipelineSQSUrl)
	if err != nil {
		logger.Fatal("failed to create sqs consumer", zap.Error(err))
	}

	vaaQueue := queue.NewEventSqs(sqsConsumer, queue.NewVaaConverter(logger), logger)
	return vaaQueue.Consume
}

func newNotificationConsumeFunc(ctx context.Context, cfg *config.Configuration, logger *zap.Logger) queue.ConsumeFunc {

	sqsConsumer, err := newSQSConsumer(ctx, cfg, cfg.NotificationsSQSUrl)
	if err != nil {
		logger.Fatal("failed to create sqs consumer", zap.Error(err))
	}

	vaaQueue := queue.NewEventSqs(sqsConsumer, queue.NewNotificationEvent(logger), logger)
	return vaaQueue.Consume
}

func newSQSConsumer(appCtx context.Context, config *config.Configuration, sqsUrl string) (*sqs_client.Consumer, error) {
	awsconfig, err := newAwsConfig(appCtx, config)
	if err != nil {
		return nil, err
	}

	return sqs_client.NewConsumer(awsconfig, sqsUrl,
		sqs_client.WithMaxMessages(10),
		sqs_client.WithVisibilityTimeout(120))
}

func newAwsConfig(appCtx context.Context, cfg *config.Configuration) (aws.Config, error) {
	region := cfg.AwsRegion

	if cfg.AwsAccessKeyID != "" && cfg.AwsSecretAccessKey != "" {
		credentials := credentials.NewStaticCredentialsProvider(cfg.AwsAccessKeyID, cfg.AwsSecretAccessKey, "")
		customResolver := aws.EndpointResolverFunc(func(service, region string) (aws.Endpoint, error) {
			if cfg.AwsEndpoint != "" {
				return aws.Endpoint{
					PartitionID:   "aws",
					URL:           cfg.AwsEndpoint,
					SigningRegion: region,
				}, nil
			}

			return aws.Endpoint{}, &aws.EndpointNotFoundError{}
		})

		awsCfg, err := awsconfig.LoadDefaultConfig(appCtx,
			awsconfig.WithRegion(region),
			awsconfig.WithEndpointResolver(customResolver),
			awsconfig.WithCredentialsProvider(credentials),
		)
		return awsCfg, err
	}
	return awsconfig.LoadDefaultConfig(appCtx, awsconfig.WithRegion(region))
}

func newInfluxClient(url, token string) influxdb2.Client {
	return influxdb2.NewClient(url, token)
}

func newHealthChecks(
	ctx context.Context,
	config *config.Configuration,
	influxCli influxdb2.Client,
	db *mongo.Database,
) ([]health.Check, error) {

	awsConfig, err := newAwsConfig(ctx, config)
	if err != nil {
		return nil, err
	}

	healthChecks := []health.Check{
		health.SQS(awsConfig, config.PipelineSQSUrl),
		health.SQS(awsConfig, config.NotificationsSQSUrl),
		health.Influx(influxCli),
		health.Mongo(db),
	}
	return healthChecks, nil
}

func newNotionalCache(
	ctx context.Context,
	cfg *config.Configuration,
	logger *zap.Logger,
) (wormscanNotionalCache.NotionalLocalCacheReadable, error) {

	// use a distributed cache and for notional a pubsub to sync local cache.
	redisClient := redis.NewClient(&redis.Options{Addr: cfg.CacheURL})

	// get notional cache client and init load to local cache
	notionalCache, err := wormscanNotionalCache.NewNotionalCache(ctx, redisClient, cfg.CachePrefix, cfg.CacheChannel, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create notional cache client: %w", err)
	}
	notionalCache.Init(ctx)

	return notionalCache, nil
}
