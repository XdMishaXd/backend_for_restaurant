package grpc

import (
	"context"
	"fmt"
	"log/slog"
	"main_service/internal/models"
	"time"

	ssov1 "github.com/XdMishaXd/ProtosForAuthService/gen/go/sso"
	grpclog "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/logging"
	grpcretry "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/retry"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
)

type Client struct {
	api ssov1.AuthClient
	log *slog.Logger
}

func New(
	ctx context.Context,
	log *slog.Logger,
	addr string,
	timeout time.Duration,
	retriesCount int,
) (*Client, error) {
	const op = "grpc.New"

	retryOpts := []grpcretry.CallOption{
		grpcretry.WithCodes(codes.NotFound, codes.Aborted, codes.DeadlineExceeded),
		grpcretry.WithMax(uint(retriesCount)),
		grpcretry.WithPerRetryTimeout(timeout),
	}

	logOpts := []grpclog.Option{
		grpclog.WithLogOnEvents(grpclog.PayloadReceived, grpclog.PayloadSent),
	}

	// caCert, err := os.ReadFile("main_service/internal/clients/sso/certificates")
	// if err != nil {
	// 	return nil, fmt.Errorf("%s: read ca cert: %w", op, err)
	// }
	// certPool := x509.NewCertPool()
	// if !certPool.AppendCertsFromPEM(caCert) {
	// 	return nil, fmt.Errorf("%s: failed to append ca cert", op)
	// }

	// creds := credentials.NewTLS(&tls.Config{
	// 	RootCAs:    certPool,
	// 	ServerName: "localhost", // должен совпадать с CN в server.crt // TODO: удалить insecure и добавить правильные credentials
	// })

	cc, err := grpc.DialContext(
		ctx,
		addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithChainUnaryInterceptor(
			grpclog.UnaryClientInterceptor(interceptorLogger(log), logOpts...),
			grpcretry.UnaryClientInterceptor(retryOpts...),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &Client{
		api: ssov1.NewAuthClient(cc),
		log: log,
	}, nil
}

func (c *Client) IsAdmin(ctx context.Context, userID int64) (bool, error) {
	const op = "grpc.IsAdmin"

	resp, err := c.api.IsAdmin(ctx, &ssov1.IsAdminRequest{
		UserId: userID,
	})
	if err != nil {
		return false, fmt.Errorf("%s: %w", op, err)
	}

	return resp.IsAdmin, nil
}

func (c *Client) GetUserInformation(ctx context.Context, userID int64) (models.User, error) {
	const op = "grpc.GetUserInformation"

	resp, err := c.api.GetUser(ctx, &ssov1.GetUserRequest{
		UserId: userID,
	})
	if err != nil {
		return models.User{}, fmt.Errorf("%s: %w", op, err)
	}

	return models.User{
		Email:      resp.Email,
		First_name: resp.FirstName,
		Last_name:  resp.LastName,
	}, nil
}

func interceptorLogger(l *slog.Logger) grpclog.Logger {
	return grpclog.LoggerFunc(func(ctx context.Context, lvl grpclog.Level, msg string, fields ...any) {
		l.Log(ctx, slog.Level(lvl), msg, fields...)
	})
}
