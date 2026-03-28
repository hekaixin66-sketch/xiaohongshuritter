package downloader

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/hekaixin66-sketch/xiaohongshuritter/configs"
)

// ImageProcessor 图片处理器
type ImageProcessor struct {
	downloader *ImageDownloader
}

// NewImageProcessor 创建图片处理器
func NewImageProcessor() *ImageProcessor {
	return &ImageProcessor{
		downloader: NewImageDownloader(configs.GetImagesPath()),
	}
}

// ProcessImages 处理图片列表，返回本地文件路径
// 支持两种输入格式：
// 1. URL格式 (http/https开头) - 自动下载到本地
// 2. 本地文件路径 - 直接使用
// 保持原始图片顺序，如果下载失败直接返回错误
func (p *ImageProcessor) ProcessImages(images []string) ([]string, error) {
	localPaths := make([]string, 0, len(images))

	// 按顺序处理每张图片
	for _, image := range images {
		if IsImageURL(image) {
			// URL图片：立即下载，失败直接返回错误
			localPath, err := p.downloader.DownloadImage(image)
			if err != nil {
				return nil, fmt.Errorf("下载图片失败 %s: %w", image, err)
			}
			localPaths = append(localPaths, localPath)
		} else {
			stagedPath, err := p.stageLocalImage(image)
			if err != nil {
				return nil, err
			}
			localPaths = append(localPaths, stagedPath)
		}
	}

	if len(localPaths) == 0 {
		return nil, fmt.Errorf("no valid images found")
	}

	return localPaths, nil
}

func (p *ImageProcessor) stageLocalImage(imagePath string) (string, error) {
	if strings.TrimSpace(imagePath) == "" {
		return "", fmt.Errorf("image path is empty")
	}

	info, err := os.Stat(imagePath)
	if err != nil {
		return "", fmt.Errorf("image path inaccessible %s: %w", imagePath, err)
	}
	if info.IsDir() {
		return "", fmt.Errorf("image path is a directory: %s", imagePath)
	}

	cleanPath := filepath.Clean(imagePath)
	stageDir := filepath.Clean(configs.GetImagesPath())
	if samePathRoot(cleanPath, stageDir) {
		return cleanPath, nil
	}

	if err := os.MkdirAll(stageDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create image staging dir %s: %w", stageDir, err)
	}

	ext := filepath.Ext(cleanPath)
	if ext == "" {
		ext = ".img"
	}
	stagedName := buildStagedFileName(cleanPath, ext)
	stagedPath := filepath.Join(stageDir, stagedName)

	src, err := os.Open(cleanPath)
	if err != nil {
		return "", fmt.Errorf("failed to open image %s: %w", cleanPath, err)
	}
	defer src.Close()

	dst, err := os.Create(stagedPath)
	if err != nil {
		return "", fmt.Errorf("failed to create staged image %s: %w", stagedPath, err)
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		return "", fmt.Errorf("failed to copy image %s to %s: %w", cleanPath, stagedPath, err)
	}

	return stagedPath, nil
}

func buildStagedFileName(path, ext string) string {
	sum := sha256.Sum256([]byte(path))
	return fmt.Sprintf("stage_%x%s", sum[:8], ext)
}

func samePathRoot(path, root string) bool {
	if path == root {
		return true
	}
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}
