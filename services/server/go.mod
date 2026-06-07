module cultivation-game/services/server

go 1.24.0

toolchain go1.24.4

require (
	cultivation-game/services/auth v0.0.0-00010101000000-000000000000
	cultivation-game/services/combat v0.0.0-00010101000000-000000000000
	cultivation-game/services/cultivation v0.0.0-00010101000000-000000000000
	cultivation-game/services/gateway v0.0.0-00010101000000-000000000000
	cultivation-game/services/player v0.0.0-00010101000000-000000000000
	cultivation-game/services/ranking v0.0.0-00010101000000-000000000000
	cultivation-game/services/social v0.0.0-00010101000000-000000000000
	cultivation-game/services/trade v0.0.0-00010101000000-000000000000
	cultivation-game/services/world v0.0.0-00010101000000-000000000000
	github.com/gin-gonic/gin v1.10.0
	github.com/go-sql-driver/mysql v1.8.1
	github.com/gorilla/websocket v1.5.3
	github.com/redis/go-redis/v9 v9.20.0
	go.mongodb.org/mongo-driver v1.13.1
)

require (
	filippo.io/edwards25519 v1.1.0 // indirect
	github.com/bytedance/sonic v1.14.0 // indirect
	github.com/bytedance/sonic/loader v0.5.0 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/cloudwego/base64x v0.1.5 // indirect
	github.com/gabriel-vasile/mimetype v1.4.12 // indirect
	github.com/gin-contrib/sse v1.1.0 // indirect
	github.com/go-playground/locales v0.14.1 // indirect
	github.com/go-playground/universal-translator v0.18.1 // indirect
	github.com/go-playground/validator/v10 v10.23.0 // indirect
	github.com/goccy/go-json v0.10.5 // indirect
	github.com/golang-jwt/jwt/v5 v5.2.1 // indirect
	github.com/golang/snappy v0.0.4 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/klauspost/compress v1.17.6 // indirect
	github.com/klauspost/cpuid/v2 v2.2.10 // indirect
	github.com/leodido/go-urn v1.4.0 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/montanaflynn/stats v0.0.0-20171201202039-1bf9dbcd8cbe // indirect
	github.com/pelletier/go-toml/v2 v2.2.4 // indirect
	github.com/rs/zerolog v1.33.0 // indirect
	github.com/twitchyliquid64/golang-asm v0.15.1 // indirect
	github.com/ugorji/go/codec v1.3.1 // indirect
	github.com/xdg-go/pbkdf2 v1.0.0 // indirect
	github.com/xdg-go/scram v1.2.0 // indirect
	github.com/xdg-go/stringprep v1.0.4 // indirect
	github.com/youmark/pkcs8 v0.0.0-20181117223130-1be2e3e5546d // indirect
	go.uber.org/atomic v1.11.0 // indirect
	go.uber.org/multierr v1.10.0 // indirect
	go.uber.org/zap v1.28.0 // indirect
	golang.org/x/arch v0.22.0 // indirect
	golang.org/x/crypto v0.28.0 // indirect
	golang.org/x/net v0.30.0 // indirect
	golang.org/x/sync v0.19.0 // indirect
	golang.org/x/sys v0.30.0 // indirect
	golang.org/x/text v0.20.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20241021214115-324edc3d5d38 // indirect
	google.golang.org/grpc v1.67.0 // indirect
	google.golang.org/protobuf v1.36.10 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	gorm.io/gorm v1.31.1 // indirect
)

replace cultivation-game/services/auth => ../auth

replace cultivation-game/services/combat => ../combat

replace cultivation-game/services/cultivation => ../cultivation

replace cultivation-game/services/gateway => ../gateway

replace cultivation-game/services/player => ../player

replace cultivation-game/services/ranking => ../ranking

replace cultivation-game/services/social => ../social

replace cultivation-game/services/trade => ../trade

replace cultivation-game/services/world => ../world

replace cultivation-game/shared => ../../shared
