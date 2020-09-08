package handlers

import (
	"context"
	"net/http"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/dgrijalva/jwt-go"
	"github.com/dung997bn/tronicscorp/config"
	"github.com/dung997bn/tronicscorp/dbiface"
	"github.com/go-playground/validator"
	"github.com/ilyakaznacheev/cleanenv"
	"github.com/labstack/echo/v4"
	"github.com/labstack/gommon/log"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

var (
	cfg config.Properties
)

//User struct
type User struct {
	Email    string `json:"username" bson:"username" validate:"required,email"`
	Password string `json:"password,omitempty" bson:"password" validate:"required,min=8,max=30"`
}

//UserHandler type
type UserHandler struct {
	Col dbiface.CollectionAPI
}

//UserValidator product
type UserValidator struct {
	validator *validator.Validate
}

//Validate validates a product
func (u *UserValidator) Validate(i interface{}) error {
	return u.validator.Struct(i)
}

func insertUser(ctx context.Context, user User, collection dbiface.CollectionAPI) (interface{}, error) {
	var newUser User
	res := collection.FindOne(ctx, bson.M{"username": user.Email})
	err := res.Decode(&newUser)
	if err != nil && err != mongo.ErrNoDocuments {
		log.Errorf("Unable to decode retrived user: %v", err)
		return nil, err
	}
	if newUser.Email != "" {
		log.Errorf("User already exists")
		return nil, echo.NewHTTPError(400, "User already exists")
	}

	hassPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), 8)
	if err != nil {
		log.Errorf("Unable to hash password: %v", err)
		return nil, echo.NewHTTPError(400, "Uable to process password")
	}
	user.Password = string(hassPassword)

	insertRes, err := collection.InsertOne(ctx, user)
	if err != nil {
		log.Errorf("Unable to insert user : %+v", err)
		return nil, echo.NewHTTPError(400, "Unable to insert user")
	}
	return insertRes.InsertedID, nil
}

//CreateUser creates a new user
func (h *UserHandler) CreateUser(c echo.Context) error {
	var user User
	c.Echo().Validator = &UserValidator{validator: v}
	if err := c.Bind(&user); err != nil {
		log.Errorf("Unable to bind user struct")
		return echo.NewHTTPError(400, "Unable to parse the request payload")
	}
	if err := c.Validate(user); err != nil {
		log.Errorf("Unable to validate user", err)
		return echo.NewHTTPError(400, "Unable to validate user")

	}
	insertedUserID, err := insertUser(context.Background(), user, h.Col)
	if err != nil {
		log.Errorf("Unable to insert user to database")
		return err
	}
	token, err := createToken(user.Email)
	if err != nil {
		log.Errorf("Unable to generate token", err)
		return echo.NewHTTPError(400, "Unable to generate token")
	}
	c.Response().Header().Set("X-auth-token", "Bearer "+token)
	return c.JSON(http.StatusCreated, insertedUserID)
}

func isCredValid(givenPwd string, storedPwd string) bool {
	if err := bcrypt.CompareHashAndPassword([]byte(storedPwd), []byte(givenPwd)); err != nil {
		return false
	}
	return true
}

func authenticateUser(ctx context.Context, reqUser User, collection dbiface.CollectionAPI) (User, error) {
	var storedUser User
	//check user exist or not
	res := collection.FindOne(ctx, bson.M{"username": reqUser.Email})
	err := res.Decode(&storedUser)
	if err != nil && err != mongo.ErrNoDocuments {
		log.Errorf("Unable to decode retrived user: %v", err)
		return storedUser, err
	}
	if err == mongo.ErrNoDocuments {
		log.Errorf("User %s does not exist", reqUser.Email)
		return storedUser, err
	}
	if !isCredValid(reqUser.Password, storedUser.Password) {
		return storedUser, echo.NewHTTPError(http.StatusUnauthorized, "Credentials invalid")
	}
	return User{Email: storedUser.Email}, nil
}

func createToken(email string) (string, error) {
	if err := cleanenv.ReadEnv(&cfg); err != nil {
		log.Fatalf("Configuration cannot be read :%v", err)
	}
	claims := jwt.MapClaims{}
	claims["authorized"] = true
	claims["user_id"] = email
	claims["exp"] = time.Now().Add(time.Minute * 15).Unix()
	at := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	token, err := at.SignedString([]byte(cfg.JWTTokenSeCret))
	if err != nil {
		log.Errorf("Unable to generate token: %v", err)
		return "", err
	}
	return token, nil
}

//AuthenUser authenticate user and return token
func (h *UserHandler) AuthenUser(c echo.Context) error {
	var user User
	c.Echo().Validator = &UserValidator{validator: v}
	if err := c.Bind(&user); err != nil {
		log.Errorf("Unable to bind user struct")
		return echo.NewHTTPError(400, "Unable to parse the request payload")
	}

	if err := c.Validate(user); err != nil {
		log.Errorf("Unable to validate request body", err)
		return echo.NewHTTPError(400, "Unable to validate request body")
	}

	user, err := authenticateUser(context.Background(), user, h.Col)
	if err != nil {
		log.Errorf("Unable to authenticate user", err)
		return err
	}

	token, err := createToken(user.Email)
	if err != nil {
		log.Errorf("Unable to generate token", err)
		return echo.NewHTTPError(400, "Unable to generate token")
	}
	c.Response().Header().Set("X-auth-token", "Bearer "+token)
	return c.JSON(http.StatusOK, User{Email: user.Email})
}
