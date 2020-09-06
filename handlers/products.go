package handlers

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"

	"github.com/dung997bn/tronicscorp/dbiface"
	"github.com/go-playground/validator"

	"github.com/labstack/echo/v4"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

var (
	v = validator.New()
)

//ProductValidator product
type ProductValidator struct {
	validator *validator.Validate
}

//Validate validates a product
func (p *ProductValidator) Validate(i interface{}) error {
	return p.validator.Struct(i)
}

//Product describes an electronic product
type Product struct {
	ID          primitive.ObjectID `json:"_id,omitempty" bson:"_id,omitempty"`
	Name        string             `json:"product_name" bson:"product_name"`
	Price       int                `json:"price" bson:"price"`
	Currency    string             `json:"currency" bson:"currency"`
	Quantity    int                `json:"quantity" bson:"quantity"`
	Discount    int                `json:"discount,omitempty" bson:"discount,omitempty"`
	Vendor      string             `json:"vendor" bson:"vendor" validate:"required"`
	Accessories []string           `json:"accessories,omitempty" bson:"accessories,omitempty"`
	IsEssential bool               `json:"is_essential,omitempty" bson:"is_essential"`
}

//ProductHandler type
type ProductHandler struct {
	Col dbiface.CollectionAPI
}

func insertProducts(ctx context.Context, products []Product, collection dbiface.CollectionAPI) ([]interface{}, error) {
	var insertedIds []interface{}
	for _, product := range products {
		product.ID = primitive.NewObjectID()
		insertID, err := collection.InsertOne(ctx, product)
		if err != nil {
			log.Fatalf("Unable to insert: %v", err)
			return nil, err
		}
		insertedIds = append(insertedIds, insertID.InsertedID)
	}

	return insertedIds, nil

}

//CreateProducts func
func (h *ProductHandler) CreateProducts(c echo.Context) error {
	var products []Product
	c.Echo().Validator = &ProductValidator{validator: v}
	err := c.Bind(&products)
	fmt.Println(products)
	if err != nil {
		log.Printf("Unable to bind : %v", err)
	}
	for _, product := range products {
		if err := c.Validate(product); err != nil {
			log.Printf("Unable to validate the product %+v %+v", product, err)
			return err
		}
	}

	IDs, err := insertProducts(context.Background(), products, h.Col)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusCreated, IDs)
}

func findProducts(ctx context.Context, q url.Values, collection dbiface.CollectionAPI) ([]Product, error) {
	var products []Product
	filter := make(map[string]interface{})
	for k, v := range q {
		filter[k] = v[0]
	}
	var cursor, err = collection.Find(ctx, bson.M(filter))
	if err != nil {
		log.Printf("Unable to find products: %v", err)
	}
	err = cursor.All(ctx, &products)
	if err != nil {
		log.Printf("Unable to find products: %v", err)
	}
	return products, nil
}

//GetProducts get list products
func (h *ProductHandler) GetProducts(c echo.Context) error {

	products, err := findProducts(context.Background(), c.QueryParams(), h.Col)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, products)
}
