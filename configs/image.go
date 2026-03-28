package configs

import (
	"os"
	"path/filepath"
)

const (
	ImagesDir          = "xiaohongshu_images"
	ContainerImagesDir = "/app/images"
)

func GetImagesPath() string {
	if path := os.Getenv("XHS_STAGE_IMAGE_DIR"); path != "" {
		return path
	}
	if stat, err := os.Stat(ContainerImagesDir); err == nil && stat.IsDir() {
		return ContainerImagesDir
	}
	return filepath.Join(os.TempDir(), ImagesDir)
}
