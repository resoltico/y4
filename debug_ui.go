//go:build debug

package main

import (
	"fmt"
	"log/slog"
	"reflect"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

func DebugLogUILayout(logger *slog.Logger, containerName string, obj fyne.CanvasObject) {
	if obj == nil {
		logger.Debug("ui layout debug", "container", containerName, "status", "nil")
		return
	}

	size := obj.Size()
	pos := obj.Position()
	minSize := obj.MinSize()
	visible := obj.Visible()
	objType := reflect.TypeOf(obj).String()

	logger.Debug("ui layout debug",
		"container", containerName,
		"type", objType,
		"size", formatSize(size),
		"position", formatPosition(pos),
		"min_size", formatSize(minSize),
		"visible", visible,
		"timestamp", time.Now().UnixMilli(),
	)

	// Type-specific debugging
	switch v := obj.(type) {
	case *container.Split:
		logger.Debug("split container details",
			"container", containerName,
			"offset", v.Offset,
			"horizontal", v.Horizontal,
			"leading_size", formatSize(v.Leading.Size()),
			"trailing_size", formatSize(v.Trailing.Size()),
			"leading_visible", v.Leading.Visible(),
			"trailing_visible", v.Trailing.Visible(),
		)
	case *fyne.Container:
		logger.Debug("container details",
			"container", containerName,
			"object_count", len(v.Objects),
			"layout_type", fmt.Sprintf("%T", v.Layout),
			"hidden", v.Hidden,
		)
		for i, child := range v.Objects {
			logger.Debug("container child",
				"parent", containerName,
				"child_index", i,
				"child_type", reflect.TypeOf(child).String(),
				"child_size", formatSize(child.Size()),
				"child_visible", child.Visible(),
			)
		}
	case *widget.Label:
		logger.Debug("label details",
			"container", containerName,
			"text", v.Text,
			"text_style", fmt.Sprintf("%+v", v.TextStyle),
		)
	}
}

func DebugLogContainerHierarchy(logger *slog.Logger, name string, obj fyne.CanvasObject, depth int) {
	if obj == nil {
		return
	}

	indent := ""
	for i := 0; i < depth; i++ {
		indent += "  "
	}

	objType := reflect.TypeOf(obj).String()
	size := obj.Size()

	logger.Debug("ui hierarchy",
		"name", name,
		"depth", depth,
		"type", objType,
		"size", formatSize(size),
		"visible", obj.Visible(),
		"indent", indent,
	)

	if cont, ok := obj.(*fyne.Container); ok {
		for i, child := range cont.Objects {
			childName := fmt.Sprintf("%s[%d]", name, i)
			DebugLogContainerHierarchy(logger, childName, child, depth+1)
		}
	}
}

func DebugLogWindowSizing(logger *slog.Logger, window fyne.Window, context string) {
	if window == nil {
		return
	}

	windowSize := window.Canvas().Size()
	content := window.Content()
	contentSize := fyne.NewSize(0, 0)
	contentMinSize := fyne.NewSize(0, 0)

	if content != nil {
		contentSize = content.Size()
		contentMinSize = content.MinSize()
	}

	logger.Debug("window sizing debug",
		"context", context,
		"window_size", formatSize(windowSize),
		"content_size", formatSize(contentSize),
		"content_min_size", formatSize(contentMinSize),
		"content_type", reflect.TypeOf(content).String(),
		"timestamp", time.Now().UnixMilli(),
	)
}

func DebugLogImageSizing(logger *slog.Logger, name string, img *canvas.Image) {
	if img == nil {
		return
	}

	logger.Debug("image sizing debug",
		"name", name,
		"size", formatSize(img.Size()),
		"min_size", formatSize(img.MinSize()),
		"fill_mode", fmt.Sprintf("%d", img.FillMode),
		"scale_mode", fmt.Sprintf("%d", img.ScaleMode),
		"has_image", img.Image != nil,
		"visible", img.Visible(),
	)

	if img.Image != nil {
		bounds := img.Image.Bounds()
		logger.Debug("underlying image details",
			"name", name,
			"image_width", bounds.Dx(),
			"image_height", bounds.Dy(),
		)
	}
}

func formatSize(size fyne.Size) string {
	return fmt.Sprintf("%.1fx%.1f", size.Width, size.Height)
}

func formatPosition(pos fyne.Position) string {
	return fmt.Sprintf("%.1f,%.1f", pos.X, pos.Y)
}

func DebugTraceUIEvent(event string, containerName string, details map[string]interface{}) {
	debugSystem := GetDebugSystem()
	if debugSystem == nil {
		return
	}

	logArgs := []interface{}{
		"ui_event", event,
		"container", containerName,
		"timestamp", time.Now().UnixMilli(),
	}

	for key, value := range details {
		logArgs = append(logArgs, key, value)
	}

	debugSystem.logger.Debug("ui event trace", logArgs...)
}

func DebugLogLayoutRefresh(logger *slog.Logger, containerName string, obj fyne.CanvasObject, reason string) {
	logger.Debug("layout refresh",
		"container", containerName,
		"reason", reason,
		"size_before", formatSize(obj.Size()),
		"min_size", formatSize(obj.MinSize()),
		"timestamp", time.Now().UnixMilli(),
	)
}
