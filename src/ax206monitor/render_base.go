package main

import (
	"math"
	"strings"

	"github.com/fogleman/gg"
	"golang.org/x/image/font"
)

type BaseTextRole string

const (
	TextRoleValue BaseTextRole = "value"
	TextRoleLabel BaseTextRole = "label"
	TextRoleUnit  BaseTextRole = "unit"
	TextRoleTitle BaseTextRole = "title"
	TextRoleMeta  BaseTextRole = "meta"
)

type BaseAlignH string
type BaseAlignV string

const (
	AlignLeft   BaseAlignH = "left"
	AlignCenter BaseAlignH = "center"
	AlignRight  BaseAlignH = "right"
)

const (
	AlignTop    BaseAlignV = "top"
	AlignMiddle BaseAlignV = "middle"
	AlignBottom BaseAlignV = "bottom"
)

type BaseTextDrawOptions struct {
	Role     BaseTextRole
	FontSize int
	Color    string
	AlignH   BaseAlignH
	AlignV   BaseAlignV
	PaddingX float64
	PaddingY float64
}

func drawBaseItemFrame(dc *gg.Context, item *ItemConfig, config *MonitorConfig) {
	if dc == nil || item == nil {
		return
	}
	radius := resolveItemRadius(item, config, 0)
	drawRoundedBackground(dc, item.X, item.Y, item.Width, item.Height, resolveItemBackground(item, config), radius)
	drawBaseItemBorder(dc, item, config, radius)
}

func drawBaseItemBorder(dc *gg.Context, item *ItemConfig, config *MonitorConfig, radius float64) {
	if dc == nil || item == nil {
		return
	}
	borderWidth := resolveItemBorderWidth(item, config)
	if borderWidth <= 0 {
		return
	}
	if radius < 0 {
		radius = 0
	}
	dc.SetColor(parseColor(resolveItemBorderColor(item, config)))
	dc.SetLineWidth(borderWidth)
	if radius > 0 {
		dc.DrawRoundedRectangle(float64(item.X), float64(item.Y), float64(item.Width), float64(item.Height), radius)
	} else {
		dc.DrawRectangle(float64(item.X), float64(item.Y), float64(item.Width), float64(item.Height))
	}
	dc.Stroke()
}

func drawTextInItemRect(
	dc *gg.Context,
	fontCache *FontCache,
	item *ItemConfig,
	config *MonitorConfig,
	text string,
	x, y, width, height int,
	opts BaseTextDrawOptions,
) {
	if strings.TrimSpace(text) == "" || dc == nil {
		return
	}
	fontSize := opts.FontSize
	if fontSize <= 0 {
		fontSize = resolveFontSizeByTextRole(item, config, opts.Role, 12)
	}
	face := resolveFontFace(fontCache, fontSize)
	dc.SetFontFace(face)

	colorValue := strings.TrimSpace(opts.Color)
	if colorValue == "" {
		colorValue = resolveColorByTextRole(item, config, opts.Role)
	}
	dc.SetColor(parseColor(colorValue))

	left := float64(x) + opts.PaddingX
	right := float64(x+width) - opts.PaddingX
	top := float64(y) + opts.PaddingY
	bottom := float64(y+height) - opts.PaddingY
	if right < left {
		mid := float64(x) + float64(width)/2
		left = mid
		right = mid
	}
	if bottom < top {
		mid := float64(y) + float64(height)/2
		top = mid
		bottom = mid
	}

	anchorX := 0.0
	textX := left
	switch opts.AlignH {
	case AlignRight:
		textX = right
		anchorX = 1
	case AlignCenter:
		textX = (left + right) / 2
		anchorX = 0.5
	default:
		textX = left
		anchorX = 0
	}

	centerY := (top + bottom) / 2
	switch opts.AlignV {
	case AlignTop:
		metrics := baseMeasureText(face, text)
		dc.DrawStringAnchored(text, textX, top+metrics.ascent, anchorX, 0)
		return
	case AlignBottom:
		metrics := baseMeasureText(face, text)
		dc.DrawStringAnchored(text, textX, bottom-metrics.descent, anchorX, 0)
		return
	default:
		drawBaseMetricAnchoredText(dc, face, text, textX, centerY, anchorX)
	}
}

func resolveFontSizeByTextRole(item *ItemConfig, config *MonitorConfig, role BaseTextRole, fallback int) int {
	switch role {
	case TextRoleLabel, TextRoleTitle, TextRoleMeta:
		return resolveLabelFontSize(item, config, fallback)
	case TextRoleUnit:
		return resolveUnitFontSize(item, config, fallback)
	default:
		return resolveItemFontSize(item, config, fallback)
	}
}

func resolveColorByTextRole(item *ItemConfig, config *MonitorConfig, role BaseTextRole) string {
	base := resolveItemStaticColor(item, config)
	if role == TextRoleUnit {
		return resolveUnitColor(item, config, base)
	}
	return base
}

type baseTextMetrics struct {
	ascent  float64
	descent float64
}

func baseMeasureText(face font.Face, text string) baseTextMetrics {
	metrics := baseTextMetrics{
		ascent:  float64(face.Metrics().Ascent.Ceil()),
		descent: float64(face.Metrics().Descent.Ceil()),
	}
	if metrics.ascent <= 0 {
		metrics.ascent = 6
	}
	if metrics.descent < 0 {
		metrics.descent = 0
	}
	bounds, _ := font.BoundString(face, strings.TrimSpace(text))
	minY := float64(bounds.Min.Y) / 64.0
	maxY := float64(bounds.Max.Y) / 64.0
	if minY < 0 {
		metrics.ascent = math.Max(metrics.ascent, -minY)
	}
	if maxY > 0 {
		metrics.descent = math.Max(metrics.descent, maxY)
	}
	return metrics
}

func baseBaselineForCenteredText(face font.Face, text string, centerY float64) float64 {
	metrics := baseMeasureText(face, text)
	return centerY + (metrics.ascent-metrics.descent)/2
}

func drawBaseMetricAnchoredText(dc *gg.Context, face font.Face, text string, x, centerY, anchorX float64) {
	if strings.TrimSpace(text) == "" || dc == nil {
		return
	}
	baseline := baseBaselineForCenteredText(face, text, centerY)
	dc.SetFontFace(face)
	dc.DrawStringAnchored(text, x, baseline, anchorX, 0)
}
