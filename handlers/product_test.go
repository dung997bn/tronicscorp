package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/dung997bn/tronicscorp/config"
	"github.com/ilyakaznacheev/cleanenv"
	"github.com/labstack/echo/v4"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var docID string
var (
	c   *mongo.Client
	db  *mongo.Database
	col *mongo.Collection
	cfg config.Properties
	h   ProductHandler
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
	col = db.Collection(cfg.CollectionName)

}
func TestProduct(t *testing.T) {

	t.Run("test create product", func(t *testing.T) {
		body := `
		[
    {
        "product_name":"laptop",
        "price":30000,
        "currency":"USD",
        "quantity":344,
        "discount":0,
        "vendor":"m",
        "accessories":["media","phone"],
        "is_essential":true
    },
    {
        "product_name":"tivi",
        "price":4000,
        "currency":"USD",
        "quantity":34,
        "discount":1000,
        "vendor":"a",
        "accessories":["media","phone"],
        "is_essential":false
    }
]		
		`

		req := httptest.NewRequest("POST", "/products", strings.NewReader(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		res := httptest.NewRecorder()

		e := echo.New()
		c := e.NewContext(req, res)
		h.Col = col
		err := h.CreateProducts(c)
		assert.Nil(t, err)
		assert.Equal(t, http.StatusCreated, res.Code)
	})

	t.Run("get products", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/products", nil)
		res := httptest.NewRecorder()
		e := echo.New()
		c := e.NewContext(req, res)
		h.Col = col
		err := h.GetProducts(c)
		assert.Nil(t, err)
		assert.Equal(t, http.StatusOK, res.Code)

	})

	t.Run("get products with query params", func(t *testing.T) {
		var products []Product
		req := httptest.NewRequest(http.MethodGet, "/products?product_name=laptop", nil)
		res := httptest.NewRecorder()
		e := echo.New()
		c := e.NewContext(req, res)
		h.Col = col
		err := h.GetProducts(c)

		assert.Nil(t, err)
		assert.Equal(t, http.StatusOK, res.Code)

		assert.Equal(t, http.StatusOK, res.Code)
		err = json.Unmarshal(res.Body.Bytes(), &products)
		assert.Nil(t, err)
		for _, product := range products {
			assert.Equal(t, "laptop", product.Name)
		}
	})

}
