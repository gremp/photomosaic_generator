package tilegenerator

import (
	"encoding/gob"
	"fmt"
	"image"
	"image/color"
	"os"
	"path"

	"github.com/disintegration/imaging"
	"github.com/gofiber/fiber/v2/log"
	"github.com/gremp/photomosaic_generator/internal/configuration"

	"github.com/kyroy/kdtree"
	"github.com/kyroy/kdtree/points"
)

type TileInfo struct {
	FileName string
}

type BackupTileInfo struct {
	Red      float64
	Green    float64
	Blue     float64
	FileName string
}

func BaseImageBuilder(resizeAndSave bool, backupFilepath string) (*kdtree.KDTree, error) {
	backup, err := loadFromFile(backupFilepath)
	if err == nil {
		fmt.Println("Backup file loaded")
		tree := kdtree.New([]kdtree.Point{})
		for _, tile := range backup {
			tree.Insert(points.NewPoint([]float64{tile.Red, tile.Green, tile.Blue}, &TileInfo{FileName: tile.FileName}))
		}
		return tree, nil
	}

	config := configuration.GetInstance()

	tileImagesSourceDir := config.TileImages.SourcePath

	images, err := getImages(tileImagesSourceDir)
	if err != nil {
		return nil, fmt.Errorf("failed to get images: %w", err)
	}

	// rgbFilesInfo := make([]*TileInfo, len(images))
	tree := kdtree.New([]kdtree.Point{})
	fmt.Printf("Images found: %d. Starting resizing...\n", len(images))

	tileSize := config.TileImages.TileSize
	destPath := config.TileImages.ConvertedPath

	backup = make([]*BackupTileInfo, len(images))

	for imgIdx, imageFileName := range images {
		if imgIdx%100 == 0 {
			fmt.Printf("Resizing image on index: %d\n", imgIdx)
		}

		imageSource, err := LoadImage(tileImagesSourceDir, imageFileName)
		if err != nil {
			return nil, fmt.Errorf("failed to open image: %w", err)
		}

		imageResized, err := ResizeImage(imageSource, tileSize, tileSize)
		if err != nil {
			return nil, fmt.Errorf("failed to resize image %s: %w", imageFileName, err)
		}

		// save the resulting image as JPEG
		if resizeAndSave {
			err = imaging.Save(imageResized, path.Join(destPath, imageFileName))
			if err != nil {
				log.Warn("failed to save image: %w", err)
			}
		}

		red, green, blue := getTileInfo(imageResized)

		backup[imgIdx] = &BackupTileInfo{Red: float64(red), Green: float64(green), Blue: float64(blue), FileName: imageFileName}
		tree.Insert(points.NewPoint([]float64{backup[imgIdx].Red, backup[imgIdx].Green, backup[imgIdx].Blue}, &TileInfo{FileName: backup[imgIdx].FileName}))
	}

	saveToFile(backupFilepath, backup)
	return tree, nil
}

func LoadImage(directory, fileName string) (*image.Image, error) {
	imagePath := path.Join(directory, fileName)
	imageSource, err := imaging.Open(imagePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open image: %w", err)
	}

	return &imageSource, nil
}

func getTileInfo(image *image.NRGBA) (redScore, greenScore, blueScore int) {
	totalCount := int(image.Bounds().Dy() * image.Bounds().Dx())

	var redSum int
	var greenSum int
	var blueSum int

	for y := 0; y < image.Bounds().Dy(); y++ {
		for x := 0; x < image.Bounds().Dx(); x++ {
			c := image.At(x, y)
			rgba := color.RGBAModel.Convert(c).(color.RGBA)
			redSum += int(rgba.R)
			greenSum += int(rgba.G)
			blueSum += int(rgba.B)
		}
	}

	// In case we need a approximation score
	// redScore = redSum / totalCount
	// greenScore = greenSum / totalCount
	// blueScore = blueSum / totalCount

	// rgbScore := (redScore * redScore) + (greenScore * greenScore) + (blueScore * blueScore)

	return redSum / totalCount, greenSum / totalCount, blueSum / totalCount

}

func ResizeImage(image *image.Image, width, height int) (*image.NRGBA, error) {
	dstImageFill := imaging.Fill(*image, width, height, imaging.Center, imaging.Lanczos)
	return dstImageFill, nil
}

func getImages(directory string) ([]string, error) {
	entries, err := os.ReadDir(directory)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %s", err)
	}

	images := filterImages(entries)
	return images, nil
}

func filterImages(entries []os.DirEntry) []string {
	var images []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if isImage(entry.Name()) {
			images = append(images, entry.Name())
		}
	}
	return images
}

func isImage(name string) bool {
	return name[len(name)-4:] == ".jpg" || name[len(name)-5:] == ".jpeg"
}

func saveToFile(filename string, data []*BackupTileInfo) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := gob.NewEncoder(file)
	err = encoder.Encode(data)
	if err != nil {
		return err
	}

	return nil
}

func loadFromFile(filename string) ([]*BackupTileInfo, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var data []*BackupTileInfo
	decoder := gob.NewDecoder(file)
	err = decoder.Decode(&data)
	if err != nil {
		return nil, err
	}

	return data, nil
}
