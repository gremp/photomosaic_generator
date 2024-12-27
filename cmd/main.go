package main

import (
	"fmt"

	"github.com/gremp/photomosaic_generator/internal/mosaicgenerator"
	"github.com/gremp/photomosaic_generator/internal/tilegenerator"
	"github.com/kyroy/kdtree"
)

func main() {
	var tree *kdtree.KDTree
	backupFile := "./backup.gob"
	tree, err := tilegenerator.BaseImageBuilder(true, backupFile)
	if err != nil {
		panic(fmt.Errorf("cannot create kdtree: %w", err))
	}

	err2 := mosaicgenerator.MainImageBuilder(tree)
	if err2 != nil {
		panic(fmt.Errorf("cannot create main image: %w", err))
	}

}
