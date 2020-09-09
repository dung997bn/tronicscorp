package main

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/dgrijalva/jwt-go"

	"github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/log"
	"github.com/labstack/gommon/random"

	"github.com/dung997bn/tronicscorp/handlers"

	"github.com/dung997bn/tronicscorp/config"
	"github.com/ilyakaznacheev/cleanenv"
	"github.com/labstack/echo/v4"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	//CorrelationID header
	CorrelationID = "X-Request-Id"
)

var (
	c       *mongo.Client
	db      *mongo.Database
	prodCol *mongo.Collection
	userCol *mongo.Collection
	cfg     config.Properties
)

func init() {
	if err := cleanenv.ReadEnv(&cfg); err != nil {
		fmt.Printf("Configuration cannot be read : %v", err)
	}

	connectURI := fmt.Sprintf("mongodb://%s:%s", cfg.DBHost, cfg.DBPort)
	c, err := mongo.Connect(context.Background(), options.Client().ApplyURI(connectURI))
	if err != nil {
		fmt.Printf("Unable to connect database: %v", err)
	}
	db = c.Database(cfg.DBName)
	prodCol = db.Collection(cfg.ProductCollection)
	userCol = db.Collection(cfg.UsersCollection)

	isUserIndexUnique := true
	indexModel := mongo.IndexModel{
		Keys: bson.D{{"username", 1}},
		Options: &options.IndexOptions{
			Unique: &isUserIndexUnique,
		},
	}
	_, err = userCol.Indexes().CreateOne(context.Background(), indexModel)
	if err != nil {
		log.Fatalf("Unable to create an index: %+v", err)
	}
}

//custom header middleware: X-Request-Id
func addCorrelationID(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		//generate correlation id
		id := c.Request().Header.Get(CorrelationID)
		var newID string
		if id == "" {
			//generate radom number
			newID = random.String(12)
		} else {
			newID = id
		}
		c.Request().Header.Set(CorrelationID, newID)
		c.Response().Header().Set(CorrelationID, newID)

		return next(c)
	}
}

func adminMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		hToken := c.Request().Header.Get("X-auth-token")
		token := strings.Split(hToken, " ")[1]

		claims := jwt.MapClaims{}
		_, err := jwt.ParseWithClaims(token, claims, func(*jwt.Token) (interface{}, error) {
			return []byte(cfg.JWTTokenSeCret), nil
		})
		if err != nil {
			return echo.NewHTTPError(http.StatusUnauthorized, "Unable to parse token")
		}
		fmt.Println(claims)
		if !claims["authorized"].(bool) {
			return echo.NewHTTPError(http.StatusForbidden, "Not authorized")
		}

		return next(c)
	}
}

func main() {
	e := echo.New()
	e.Logger.SetLevel(log.ERROR)
	//middlware
	jwtMiddleware := middleware.JWTWithConfig(middleware.JWTConfig{
		SigningKey:  []byte(cfg.JWTTokenSeCret),
		TokenLookup: "header:X-auth-token",
	})
	e.Pre(middleware.RemoveTrailingSlash())
	e.Pre(addCorrelationID)
	e.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{
		Format: `${time_rfc3339_nano} ${remote_ip} ${host} ${header:X-correlation-ID} ${method} ${uri} ${user_agent} ${status} ${error} ${latency_human}` + "\n",
	}))

	h := handlers.ProductHandler{Col: prodCol}
	uh := handlers.UserHandler{Col: userCol}
	//routes
	//Products
	e.GET("/products", h.GetProducts)
	e.GET("/products/:id", h.GetSingleProduct)
	e.POST("/products", h.CreateProducts, middleware.BodyLimit("1M"), jwtMiddleware)

	e.PUT("/products/:id", h.UpdateProduct, middleware.BodyLimit("1M"), jwtMiddleware)
	e.DELETE("/products/:id", h.DeleteProduct, jwtMiddleware, adminMiddleware)

	//Users
	e.POST("/users", uh.CreateUser)
	e.POST("/auth", uh.AuthenUser)
	e.Logger.Infof("Listening on %s:%s", cfg.Host, cfg.Port)
	e.Logger.Fatal(e.Start(fmt.Sprintf("%s:%s", cfg.Host, cfg.Port)))
}
