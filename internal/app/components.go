package app

import (
	"fmt"
	"math"

	"github.com/diamondburned/gotk4/pkg/cairo"
	gtk "github.com/diamondburned/gotk4/pkg/gtk/v4"
)


func (app *App) createScoreCircle(score float64) *gtk.DrawingArea {
	area := gtk.NewDrawingArea()
	area.SetSizeRequest(60, 60)

	area.SetDrawFunc(func(area *gtk.DrawingArea, cr *cairo.Context, width, height int) {
		centerX := float64(width) / 2
		centerY := float64(height) / 2
		radius := math.Min(float64(width), float64(height))/2 - 4

		var r, g, b float64
		if score < 30 {
			r, g, b = 0.8, 0.2, 0.2
		} else if score < 75 {
			r, g, b = 0.9, 0.7, 0.1
		} else {
			r, g, b = 0.2, 0.7, 0.2
		}

		cr.SetSourceRGB(r, g, b)
		cr.Arc(centerX, centerY, radius, 0, 2*math.Pi)
		cr.Fill()

		cr.SetSourceRGB(1, 1, 1)
		cr.SelectFontFace("Sans", cairo.FontSlantNormal, cairo.FontWeightBold)
		cr.SetFontSize(16)

		text := fmt.Sprintf("%.0f", score)
		textExtents := cr.TextExtents(text)
		cr.MoveTo(centerX-textExtents.Width/2, centerY+textExtents.Height/2)
		cr.ShowText(text)
	})

	return area
}

func (app *App) formatValue(value float64, unit string) string {
	if value == float64(int(value)) {
		return fmt.Sprintf("%d %s", int(value), unit)
	}
	return fmt.Sprintf("%.1f %s", value, unit)
}
