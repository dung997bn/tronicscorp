package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/dung997bn/tronicscorp/dbiface"
	"github.com/go-playground/validator"
	"github.com/labstack/gommon/log"

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
		return echo.NewHTTPError(http.StatusBadRequest, "unable to parse request payload")
	}
	for _, product := range products {
		if err := c.Validate(product); err != nil {
			log.Printf("Unable to validate the product %+v %+v", product, err)
			return echo.NewHTTPError(http.StatusBadRequest, "Unable to validate the product")
		}
	}

	IDs, err := insertProducts(context.Background(), products, h.Col)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Error when insert data")
	}
	return c.JSON(http.StatusCreated, IDs)
}

func findProducts(ctx context.Context, q url.Values, collection dbiface.CollectionAPI) ([]Product, error) {
	var products []Product
	filter := make(map[string]interface{})
	for k, v := range q {
		filter[k] = v[0]
	}

	if filter["_id"] != nil {
		docID, err := primitive.ObjectIDFromHex(filter["_id"].(string))
		if err != nil {
			return products, err
		}
		filter["_id"] = docID
	}
	var cursor, err = collection.Find(ctx, bson.M(filter))
	if err != nil {
		log.Printf("Unable to find products: %v", err)
		return products, err
	}
	err = cursor.All(ctx, &products)
	if err != nil {
		log.Printf("Unable to find products: %v", err)
		return products, err
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

func modifyProduct(ctx context.Context, id string, reqBody io.ReadCloser, collection dbiface.CollectionAPI) (Product, error) {
	var product Product
	//find if product exist, if err return 404
	docID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		log.Errorf("Cannot convert to objectId :%v", err)
		return product, echo.NewHTTPError(500, "Cannot convert to objectId")
	}
	filter := bson.M{"_id": docID}
	res := collection.FindOne(ctx, filter)
	if err := res.Decode(&product); err != nil {
		log.Errorf("Unable to decode product :%v", err)
		return product, err
	}
	//decode the req payload, if err return 500
	if err := json.NewDecoder(reqBody).Decode(&product); err != nil {
		return product, err
	}
	//validate the request,if err return 400
	if err := v.Struct(product); err != nil {
		return product, err
	}
	//update product, if err return 500
	_, err = collection.UpdateOne(ctx, filter, bson.M{"$set": product})
	if err != nil {
		return product, err
	}
	return product, nil
}

//UpdateProduct updates aproduct
func (h *ProductHandler) UpdateProduct(c echo.Context) error {

	product, err := modifyProduct(context.Background(), c.Param("id"), c.Request().Body, h.Col)
	if err != nil {
		log.Errorf("Cannot update product :%v", err)
		return err
	}
	return c.JSON(http.StatusOK, product)
}

func findProduct(ctx context.Context, id string, collection dbiface.CollectionAPI) (Product, error) {
	var product Product
	docID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return product, err
	}
	res := collection.FindOne(ctx, bson.M{"_id": docID})
	if err := res.Decode(&product); err != nil {
		return product, err
	}
	return product, nil
}

//GetSingleProduct gets single product by id
func (h *ProductHandler) GetSingleProduct(c echo.Context) error {
	products, err := findProduct(context.Background(), c.Param("id"), h.Col)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, products)
}

func deleteProduct(ctx context.Context, id string, collection dbiface.CollectionAPI) (int64, error) {
	docID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return 0, err
	}
	resFind := collection.FindOne(ctx, bson.M{"_id": docID})
	if resFind == nil {
		return 0, echo.NewHTTPError(404, "Product not found")
	}
	resDel, err := collection.DeleteOne(ctx, bson.M{"_id": docID})
	if err != nil {
		return 0, echo.NewHTTPError(404, "Cannot delete product")
	}
	return resDel.DeletedCount, nil
}

//DeleteProduct deletes a product by id
func (h *ProductHandler) DeleteProduct(c echo.Context) error {
	delCount, err := deleteProduct(context.Background(), c.Param("id"), h.Col)
	if err != nil {
		return echo.NewHTTPError(404, "Cannot delete product")
	}
	return c.JSON(http.StatusOK, delCount)
}
