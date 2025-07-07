//go:build !debug

package main

import (
	"log/slog"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
)

func DebugLogUILayout(logger *slog.Logger, containerName string, obj fyne.CanvasObject)             {}
func DebugLogContainerHierarchy(logger *slog.Logger, name string, obj fyne.CanvasObject, depth int) {}
func DebugLogWindowSizing(logger *slog.Logger, window fyne.Window, context string)                  {}
func DebugLogImageSizing(logger *slog.Logger, name string, img *canvas.Image)                       {}
func DebugTraceUIEvent(event string, containerName string, details map[string]interface{})          {}
func DebugLogLayoutRefresh(logger *slog.Logger, containerName string, obj fyne.CanvasObject, reason string) {
}
