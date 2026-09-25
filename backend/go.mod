module pekan/backend

go 1.23

require (
	github.com/DATA-DOG/go-sqlmock v1.5.2
	github.com/aws/aws-sdk-go-v2 v1.32.7
	github.com/aws/aws-sdk-go-v2/config v1.28.7
	github.com/aws/aws-sdk-go-v2/credentials v1.17.48
	github.com/aws/aws-sdk-go-v2/service/s3 v1.72.0
	github.com/go-chi/chi/v5 v5.1.0
	github.com/golang-jwt/jwt/v5 v5.2.1
	github.com/google/uuid v1.6.0
	github.com/jackc/pgx/v5 v5.6.0
	github.com/phpdave11/gofpdf v1.4.3
	github.com/redis/go-redis/v9 v9.5.1
	golang.org/x/crypto v0.36.0
	google.golang.org/api v0.150.0
)

replace google.golang.org/api => google.golang.org/api v0.150.0
