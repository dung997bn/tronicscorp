package config

//Properties configuration properties based on env variables
type Properties struct {
	Port           string `env:"MY_APP_PORT" env-default:"8080"`
	Host           string `env:"HOST" env-default:"localhost"`
	DBHost         string `env:"DB_HOST" env-default:"127.0.0.1"`
	DBPort         string `env:"DB_PORT" env-default:"27017"`
	DBName         string `env:"DB_NAME" env-default:"tronics"`
	CollectionName string `env:"DB_HOST" env-default:"products"`
}
