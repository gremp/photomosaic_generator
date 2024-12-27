package mosaicgenerator

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"path"

	"github.com/disintegration/imaging"
	"github.com/gofiber/fiber/v2/log"
	"github.com/gremp/photomosaic_generator/internal/configuration"
	"github.com/gremp/photomosaic_generator/internal/tilegenerator"
	"github.com/kyroy/kdtree"
	"github.com/kyroy/kdtree/points"
)

type MosaicTile struct {
	Red       int
	Green     int
	Blue      int
	RGBScore  int
	PositionX int
	PositionY int
	FileName  string
}

func MainImageBuilder(tree *kdtree.KDTree) error {
	config := configuration.GetInstance()
	mainImagePixelBlock := config.CollageImage.PixelBlock

	mainImage, err := imaging.Open(config.MainImage.SourcePath)
	if err != nil {
		return fmt.Errorf("failed to open main image: %w", err)
	}

	mainImageWidth := config.MainImage.Width
	mainImageHeight := config.MainImage.Height

	imageWidth := (mainImageWidth / mainImagePixelBlock) * mainImagePixelBlock
	imageHeight := (mainImageHeight / mainImagePixelBlock) * mainImagePixelBlock

	resizedImage, err := tilegenerator.ResizeImage(&mainImage, imageWidth, imageHeight)
	if err != nil {
		return fmt.Errorf("failed to resize main image: %w", err)
	}

	buildImage(resizedImage, tree)
	return nil

}

func buildImage(resizedImage image.Image, tree *kdtree.KDTree) error {
	config := configuration.GetInstance()
	convertedImagePath := config.CollageImage.CollagePath

	sameTileDistance := config.CollageImage.SameTileDistance
	pixelBlock := config.CollageImage.PixelBlock
	tileSize := config.TileImages.TileSize

	generatedImageSizeWidth := resizedImage.Bounds().Dx() * tileSize / pixelBlock
	generatedImageSizeHeight := resizedImage.Bounds().Dy() * tileSize / pixelBlock

	drawableImage := imaging.New(generatedImageSizeWidth, generatedImageSizeHeight, color.RGBA{0, 0, 0, 0})

	fmt.Printf("Generating mosaic with dimensions: %dx%d\n", generatedImageSizeWidth, generatedImageSizeHeight)
	mosaicTileSize := config.TileImages.TileSize
	imageTileMatrix := buildImageTileMatrix(resizedImage, pixelBlock)

	for tilePosX := 0; tilePosX < len(imageTileMatrix); tilePosX++ { // 0-160
		for tilePosY := 0; tilePosY < len(imageTileMatrix[tilePosX]); tilePosY++ { // 0-120
			red, green, blue := getAverageColorForPixelBlock(resizedImage, tilePosX, tilePosY, pixelBlock)
			neighboursImageFileNames := getNeighbourImagesMap(imageTileMatrix, tilePosX, tilePosY, int(sameTileDistance))

			mosaicTile, err := getMatchingImageExcludingNeighbours(tree, red, green, blue, neighboursImageFileNames)
			if err != nil {
				return fmt.Errorf("failed to get closest image: %w", err)
			}

			mosaicTile.PositionX = tilePosX
			mosaicTile.PositionY = tilePosY
			imageTileMatrix[tilePosX][tilePosY] = mosaicTile

			tileImage, err := imaging.Open(path.Join(config.TileImages.ConvertedPath, mosaicTile.FileName))
			if err != nil {
				return fmt.Errorf("failed to open tile image: %w", err)
			}

			draw.Draw(drawableImage, tileImage.Bounds().Add(image.Pt(mosaicTile.PositionX*mosaicTileSize, mosaicTile.PositionY*mosaicTileSize)), tileImage, image.Point{}, draw.Over)

			// if tilePosX > 2 || tilePosY > 2 {
			// break
			// }
		}
	}

	if err := imaging.Save(drawableImage, path.Join(convertedImagePath, "..", "output.png")); err != nil {
		return fmt.Errorf("failed to save image: %w", err)
	}

	return nil
}

func getMatchingImageExcludingNeighbours(tree *kdtree.KDTree, red, green, blue int, neighboursImageFileNames map[string]bool) (*MosaicTile, error) {
	matchingTiles := tree.KNN(&points.Point{Coordinates: []float64{float64(red), float64(green), float64(blue)}}, len(neighboursImageFileNames)+1)
	for _, tile := range matchingTiles {
		titleInfo := tile.(*points.Point).Data.(*tilegenerator.TileInfo)
		if _, ok := neighboursImageFileNames[titleInfo.FileName]; !ok {
			return &MosaicTile{
				Red:      int(tile.(*points.Point).Coordinates[0]),
				Green:    int(tile.(*points.Point).Coordinates[1]),
				Blue:     int(tile.(*points.Point).Coordinates[2]),
				FileName: titleInfo.FileName,
			}, nil
		}
	}

	return nil, fmt.Errorf("no matching tile found")
}

func getAverageColorForPixelBlock(sourceImage image.Image, tilePosX, tilePosY, mainImageTileStep int) (red, green, blue int) {
	scanStartX := tilePosX * mainImageTileStep
	scanStartY := tilePosY * mainImageTileStep
	scanEndX := scanStartX + mainImageTileStep
	scanEndY := scanStartY + mainImageTileStep

	for i := scanStartX; i < scanEndX; i++ {
		for j := scanStartY; j < scanEndY; j++ {
			c := sourceImage.At(i, j)
			rgba := color.RGBAModel.Convert(c).(color.RGBA)
			red += int(rgba.R)
			green += int(rgba.G)
			blue += int(rgba.B)
		}
	}

	totalPixels := mainImageTileStep * mainImageTileStep
	red = red / totalPixels
	green = green / totalPixels
	blue = blue / totalPixels

	return red, green, blue
}

func getNeighbourImagesMap(imageTileMatrix [][]*MosaicTile, tilePosX, tilePosY, sameTileDistance int) map[string]bool {
	closeImageFileNames := make(map[string]bool)

	for i := tilePosX - sameTileDistance; i < tilePosX+sameTileDistance; i++ {
		// Prevent out of bounds -- TODO: can be optimized on the for loop
		if i < 0 || i >= len(imageTileMatrix) {
			continue
		}

		for j := tilePosY - sameTileDistance; j < tilePosY+sameTileDistance; j++ {
			// Prevent out of bounds -- TODO: can be optimized on the for loop
			if j < 0 || j >= len(imageTileMatrix[i]) {
				continue
			}

			// if there is no image tile in the matrix, means it is not set yet
			// TODO: can be optimized not to check positions after the current tilePosX, tilePosY since its sure that they are not set yet
			if imageTileMatrix[i][j] == nil {
				break
			}

			closeImageFileNames[imageTileMatrix[i][j].FileName] = true
		}
	}

	return closeImageFileNames
}

func buildImageTileMatrix(image image.Image, mainImageTileStep int) [][]*MosaicTile {
	numOfWidthTiles := image.Bounds().Dx() / mainImageTileStep
	numOfHeightTiles := image.Bounds().Dy() / mainImageTileStep
	log.Info("WIDTH", numOfWidthTiles)
	log.Info("HEIGHT", numOfHeightTiles)
	imageTileMatrix := make([][]*MosaicTile, numOfWidthTiles)
	for i := range imageTileMatrix {
		imageTileMatrix[i] = make([]*MosaicTile, numOfHeightTiles)
	}

	log.Info("item", len(imageTileMatrix), "x", len(imageTileMatrix[0]))
	return imageTileMatrix
}
