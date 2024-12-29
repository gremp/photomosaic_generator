package configuration

import (
	"fmt"
	"os"
	"strconv"
	"sync"

	"github.com/joho/godotenv"
)

var lock = &sync.Mutex{}

var configInstance *Config

type S3Store struct {
	AccessKeyID     string
	SecretAccessKey string
	Region          string
	EndpointURLS3   string
	BucketName      string
}

type Server struct {
	HttpPort    string
	DatabaseURL string
}

type MainImage struct {
	SourcePath string
}

type TileImages struct {
	SourcePath    string
	ConvertedPath string
	TileSize      int
}

type CollageImage struct {
	PixelBlock       int
	SameTileDistance int
	CollagePath      string
	Width            int
	Height           int
}

type Config struct {
	S3Store                 S3Store
	Server                  Server
	MainImage               MainImage
	TileImages              TileImages
	CollageImage            CollageImage
	ResizeAndSaveBaseImages bool
	BuildMainImage          bool
}

func GetInstance() *Config {
	if configInstance == nil {
		lock.Lock()
		defer lock.Unlock()
		loadEnvironment()
		configInstance = &Config{}
		setEnvVariables(configInstance)
	}

	return configInstance
}

func loadEnvironment() {
	env := os.Getenv("GO_ENV")
	err := godotenv.Load(fmt.Sprintf("00-%s.env", env))
	if err != nil {
		panic(fmt.Errorf("failed to load .env file: %w", err))
	}

}

func setEnvVariables(configInstance *Config) {
	// Server
	configInstance.Server.DatabaseURL = os.Getenv("DATABASE_URL")
	configInstance.Server.HttpPort = os.Getenv("HTTP_PORT")

	// S3Store
	configInstance.S3Store.AccessKeyID = os.Getenv("AWS_ACCESS_KEY_ID")
	configInstance.S3Store.SecretAccessKey = os.Getenv("AWS_SECRET_ACCESS_KEY")
	configInstance.S3Store.Region = os.Getenv("AWS_REGION")
	configInstance.S3Store.EndpointURLS3 = os.Getenv("AWS_ENDPOINT_URL_S3")
	configInstance.S3Store.BucketName = os.Getenv("BUCKET_NAME")

	// MainImage
	configInstance.MainImage.SourcePath = os.Getenv("TARGET_IMAGE_PATH")
	configInstance.CollageImage.CollagePath = os.Getenv("COLLAGE_IMAGE_PATH")

	// TileImages
	configInstance.TileImages.SourcePath = os.Getenv("TILE_IMAGES_SOURCE_DIR")
	configInstance.TileImages.ConvertedPath = os.Getenv("TILE_IMAGES_CONVERTED_DIR")
	tileSize, err := strconv.Atoi(os.Getenv("IMAGE_TILE_SIZE"))
	if err != nil {
		panic(fmt.Errorf("failed to parse image tile size: %w", err))
	}
	configInstance.TileImages.TileSize = tileSize

	// CollageImage
	width, err := strconv.Atoi(os.Getenv("GENERATED_IMAGE_WIDTH"))
	if err != nil {
		panic(fmt.Errorf("failed to parse main image width: %w", err))
	}
	configInstance.CollageImage.Width = width

	height, err := strconv.Atoi(os.Getenv("GENERATED_IMAGE_HEIGHT"))
	if err != nil {
		panic(fmt.Errorf("failed to parse main image height: %w", err))
	}

	configInstance.CollageImage.Height = height
	pixelBlock, err := strconv.Atoi(os.Getenv("COLLAGE_IMAGE_PIXEL_BLOCK"))
	if err != nil {
		panic(fmt.Errorf("failed to parse main image tile step: %w", err))
	}
	configInstance.CollageImage.PixelBlock = pixelBlock

	sameTileDistance, err := strconv.Atoi(os.Getenv("SAME_TILE_DISTANCE"))
	if err != nil {
		panic(fmt.Errorf("failed to parse same tile distance: %w", err))
	}
	configInstance.CollageImage.SameTileDistance = sameTileDistance

	// Flags
	configInstance.ResizeAndSaveBaseImages = os.Getenv("RESIZE_AND_SAVE_BASE_IMAGES") == "true"
	configInstance.BuildMainImage = os.Getenv("BUILD_MAIN_IMAGE") == "true"
}
