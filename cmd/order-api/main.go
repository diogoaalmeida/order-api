package main

import (
	"database/sql"
	"fmt"
	"net"
	"net/http"
	"time"

	graphql_handler "github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/diogoaalmeida/order-api/configs"
	"github.com/diogoaalmeida/order-api/internal/event/handler"
	"github.com/diogoaalmeida/order-api/internal/infra/database"
	"github.com/diogoaalmeida/order-api/internal/infra/graph"
	"github.com/diogoaalmeida/order-api/internal/infra/grpc/pb"
	"github.com/diogoaalmeida/order-api/internal/infra/grpc/service"
	"github.com/diogoaalmeida/order-api/internal/infra/web/webserver"
	"github.com/diogoaalmeida/order-api/pkg/events"
	"github.com/streadway/amqp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	// mysql
	_ "github.com/go-sql-driver/mysql"
)

const (
	dbRetryAttempts       = 15
	dbRetryDelay          = 2 * time.Second
	rabbitMQRetryAttempts = 15
	rabbitMQRetryDelay    = 2 * time.Second
)

func main() {
	configs, err := configs.LoadConfig(".")
	if err != nil {
		panic(err)
	}

	db, err := sql.Open(configs.DBDriver, fmt.Sprintf("%s:%s@tcp(%s:%s)/%s", configs.DBUser, configs.DBPassword, configs.DBHost, configs.DBPort, configs.DBName))
	if err != nil {
		panic(err)
	}
	defer db.Close()

	fmt.Println("Waiting for database to be ready...")
	if err := database.WaitForDB(db, dbRetryAttempts, dbRetryDelay); err != nil {
		panic(err)
	}

	fmt.Println("Running database migrations...")
	if err := database.Migrate(db); err != nil {
		panic(err)
	}

	rabbitMQChannel := getRabbitMQChannel(configs)

	eventDispatcher := events.NewEventDispatcher()
	eventDispatcher.Register("OrderCreated", &handler.OrderCreatedHandler{
		RabbitMQChannel: rabbitMQChannel,
	})

	createOrderUseCase := NewCreateOrderUseCase(db, eventDispatcher)
	listOrdersUseCase := NewListOrdersUseCase(db)

	webserver := webserver.NewWebServer(configs.WebServerPort)
	webOrderHandler := NewWebOrderHandler(db, eventDispatcher)
	webserver.AddHandler(http.MethodPost, "/order", webOrderHandler.Create)
	webserver.AddHandler(http.MethodGet, "/order", webOrderHandler.List)
	fmt.Println("Starting web server on port", configs.WebServerPort)
	go webserver.Start()

	grpcServer := grpc.NewServer()
	createOrderService := service.NewOrderService(*createOrderUseCase, *listOrdersUseCase)
	pb.RegisterOrderServiceServer(grpcServer, createOrderService)
	reflection.Register(grpcServer)

	fmt.Println("Starting gRPC server on port", configs.GRPCServerPort)
	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", configs.GRPCServerPort))
	if err != nil {
		panic(err)
	}
	go grpcServer.Serve(lis)

	srv := graphql_handler.NewDefaultServer(graph.NewExecutableSchema(graph.Config{Resolvers: &graph.Resolver{
		CreateOrderUseCase: *createOrderUseCase,
		ListOrdersUseCase:  *listOrdersUseCase,
	}}))
	http.Handle("/", playground.Handler("GraphQL playground", "/query"))
	http.Handle("/query", srv)

	fmt.Println("Starting GraphQL server on port", configs.GraphQLServerPort)
	http.ListenAndServe(":"+configs.GraphQLServerPort, nil)
}

func getRabbitMQChannel(cfg *configs.Conf) *amqp.Channel {
	url := fmt.Sprintf("amqp://%s:%s@%s:%s/", cfg.RabbitMQUser, cfg.RabbitMQPassword, cfg.RabbitMQHost, cfg.RabbitMQPort)

	var conn *amqp.Connection
	var err error
	for i := 0; i < rabbitMQRetryAttempts; i++ {
		conn, err = amqp.Dial(url)
		if err == nil {
			break
		}
		time.Sleep(rabbitMQRetryDelay)
	}
	if err != nil {
		panic(fmt.Errorf("rabbitmq not reachable after %d attempts: %w", rabbitMQRetryAttempts, err))
	}

	ch, err := conn.Channel()
	if err != nil {
		panic(err)
	}
	return ch
}
