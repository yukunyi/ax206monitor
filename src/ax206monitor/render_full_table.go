package main

import (
	"fmt"
	"strings"

	"github.com/fogleman/gg"
)

type fullTableRowConfig struct {
	Monitor string `json:"monitor"`
	Label   string `json:"label,omitempty"`
}

type fullTableResolvedRow struct {
	fullTableRowConfig
	MonitorSnapshot *RenderMonitorSnapshot
}

type fullTableCellRect struct {
	x float64
	y float64
	w float64
	h float64
}

type FullTableRenderer struct{}

func NewFullTableRenderer() *FullTableRenderer {
	return &FullTableRenderer{}
}

func (r *FullTableRenderer) GetType() string {
	return itemTypeFullTable
}

func (r *FullTableRenderer) RequiresMonitor() bool {
	return false
}

func (r *FullTableRenderer) Render(dc *gg.Context, item *ItemConfig, frame *RenderFrame, fontCache *FontCache, config *MonitorConfig) error {
	if dc == nil || item == nil || fontCache == nil {
		return nil
	}

	cardRadius := resolveItemCardRadius(item, config)
	drawRoundedBackground(dc, item.X, item.Y, item.Width, item.Height, resolveItemBackground(item, config), cardRadius)

	contentPaddingX, contentPaddingY := resolveContentPaddingXY(item, config, 1, 1, 0, 0)
	title := resolveItemTitleText(item, config)
	bodyRect := fullRect{
		x: float64(item.X) + contentPaddingX,
		y: float64(item.Y) + contentPaddingY,
		w: float64(item.Width) - contentPaddingX*2,
		h: float64(item.Height) - contentPaddingY*2,
	}
	if bodyRect.w < 1 {
		bodyRect.w = 1
	}
	if bodyRect.h < 1 {
		bodyRect.h = 1
	}

	if title != "" {
		headerRect, nextBodyRect, labelFace, valueFace := fullBuildHeaderAndBody(item, config, fontCache, title, "", contentPaddingX, contentPaddingY, 4)
		drawFullHeader(
			dc,
			item,
			config,
			headerRect,
			labelFace,
			valueFace,
			title,
			"",
			resolveItemStaticColor(item, config),
			resolveItemStaticColor(item, config),
		)
		bodyRect = nextBodyRect
	}

	textColor := resolveItemStaticColor(item, config)
	unitColor := resolveUnitColor(item, config, textColor)
	rows := resolveFullTableRows(item, frame)
	colCount, rowCount, rowGap, rowRadius, rowBg, rowAltBg, columnGap, labelWidthRatio, showUnits := resolveFullTableLayout(item, config)
	rows = fullTableRowsForGrid(rows, colCount*rowCount)

	labelFace, _ := resolveRoleFontFace(fontCache, item, config, TextRoleText, 16, 8)
	valueFace, _ := resolveRoleFontFace(fontCache, item, config, TextRoleValue, 18, 8)
	unitFace, _ := resolveRoleFontFace(fontCache, item, config, TextRoleUnit, 14, 8)

	availableHeight := bodyRect.h - rowGap*float64(max(0, rowCount-1))
	if availableHeight < 1 {
		availableHeight = 1
	}
	rowHeight := availableHeight / float64(rowCount)
	if rowHeight < 10 {
		rowHeight = 10
	}

	availableWidth := bodyRect.w - columnGap*float64(max(0, colCount-1))
	if availableWidth < 1 {
		availableWidth = 1
	}
	cellWidth := availableWidth / float64(colCount)
	borderWidth := resolveItemBorderWidth(item, config)
	borderColor := resolveItemBorderColor(item, config)

	maxBodyY := bodyRect.y + bodyRect.h
	for idx, row := range rows {
		gridRow := idx / colCount
		gridCol := idx % colCount
		cellX := bodyRect.x + float64(gridCol)*(cellWidth+columnGap)
		cellY := bodyRect.y + float64(gridRow)*(rowHeight+rowGap)
		if cellY >= maxBodyY {
			break
		}
		currentRowHeight := rowHeight
		if cellY+currentRowHeight > maxBodyY {
			currentRowHeight = maxBodyY - cellY
		}
		if currentRowHeight <= 1 {
			break
		}
		rect := fullTableCellRect{
			x: cellX,
			y: cellY,
			w: cellWidth,
			h: currentRowHeight,
		}

		currentBg := rowBg
		if idx%2 == 1 && strings.TrimSpace(rowAltBg) != "" {
			currentBg = rowAltBg
		}
		if strings.TrimSpace(currentBg) != "" {
			drawRoundedRectFill(dc, rect.x, rect.y, rect.w, rect.h, rowRadius, currentBg)
		}

		centerY := rect.y + rect.h/2
		label := resolveFullTableRowLabel(row)
		valueText, unitText, available := resolveFullTableRowDisplay(row, showUnits)
		if !available && strings.TrimSpace(row.Monitor) != "" {
			valueText = "-"
			unitText = ""
		}

		currentTextColor := textColor
		currentUnitColor := unitColor
		if available && row.MonitorSnapshot != nil && row.MonitorSnapshot.value != nil {
			if numberValue, ok := tryGetFloat64(row.MonitorSnapshot.value.Value); ok {
				currentTextColor = resolveMonitorValueColor(item, row.MonitorSnapshot.name, row.MonitorSnapshot.value, numberValue, config)
				currentUnitColor = resolveMonitorUnitColor(item, row.MonitorSnapshot.name, row.MonitorSnapshot.value, numberValue, config)
			}
		}
		if !available {
			currentTextColor = applyAlpha(textColor, 0.55)
			currentUnitColor = applyAlpha(unitColor, 0.5)
		}

		labelWidth := rect.w * labelWidthRatio
		if labelWidth > rect.w-columnGap-24 {
			labelWidth = rect.w - columnGap - 24
		}
		if labelWidth < 24 {
			labelWidth = 24
		}
		valueX := rect.x + labelWidth + columnGap
		valueWidth := rect.w - labelWidth - columnGap
		if valueWidth < 16 {
			valueWidth = 16
		}

		dc.SetColor(parseColor(currentTextColor))
		drawMetricAnchoredText(dc, labelFace, label, rect.x+6, centerY, 0)

		if unitText == "" {
			dc.SetColor(parseColor(currentTextColor))
			drawMetricAnchoredText(dc, valueFace, valueText, valueX+valueWidth-6, centerY, 1)
			continue
		}

		dc.SetFontFace(valueFace)
		valueWidthPx, _ := dc.MeasureString(valueText)
		dc.SetFontFace(unitFace)
		unitWidthPx, _ := dc.MeasureString(unitText)
		gap := 4.0
		totalWidth := valueWidthPx + gap + unitWidthPx
		startX := valueX + valueWidth - 6 - totalWidth
		if startX < valueX {
			startX = valueX
		}

		dc.SetColor(parseColor(currentTextColor))
		drawMetricAnchoredText(dc, valueFace, valueText, startX, centerY, 0)
		dc.SetColor(parseColor(currentUnitColor))
		drawMetricAnchoredText(dc, unitFace, unitText, startX+valueWidthPx+gap, centerY, 0)
	}

	drawFullTableGrid(dc, bodyRect, len(rows), colCount, rowCount, rowHeight, cellWidth, rowGap, columnGap, borderWidth, borderColor)
	drawBaseItemBorder(dc, item, config, cardRadius)
	return nil
}

func (r *FullTableRenderer) drawEmptyState(dc *gg.Context, item *ItemConfig, fontCache *FontCache, config *MonitorConfig, bodyRect fullRect) {
	textFace, _ := resolveRoleFontFace(fontCache, item, config, TextRoleText, 14, 8)
	dc.SetColor(parseColor(applyAlpha(resolveItemStaticColor(item, config), 0.65)))
	drawBaseMetricAnchoredText(dc, textFace, "No table rows", bodyRect.x+bodyRect.w/2, bodyRect.y+bodyRect.h/2, 0.5)
}

func prepareRenderFullTableRuntime(item *ItemConfig, config *MonitorConfig) renderFullTableRuntime {
	rawRows, hasRows := getItemAttrWithDefaults(item, config, "rows")
	rows := parseFullTableRowsAttr(rawRows, hasRows)
	colCount := resolveFullTableColCount(item, config)
	return renderFullTableRuntime{
		rows:            rows,
		colCount:        colCount,
		rowCount:        resolveFullTableRowCount(item, config, colCount, rows),
		rowGap:          clampMinFloat(getItemAttrFloatCfg(item, config, "table_row_gap", 0), 0),
		rowRadius:       clampMinFloat(getItemAttrFloatCfg(item, config, "table_row_radius", 0), 0),
		rowBg:           getItemAttrColorCfg(item, config, "table_row_bg", ""),
		rowAltBg:        getItemAttrColorCfg(item, config, "table_row_alt_bg", ""),
		columnGap:       clampMinFloat(getItemAttrFloatCfg(item, config, "table_column_gap", 0), 0),
		labelWidthRatio: clampFloat64(getItemAttrFloatCfg(item, config, "table_label_width_ratio", 0.46), 0.2, 0.7),
		showUnits:       getItemAttrBoolCfg(item, config, "table_show_units", true),
	}
}

func resolveFullTableRows(item *ItemConfig, frame *RenderFrame) []fullTableResolvedRow {
	configs := resolveFullTableRowConfigs(item)
	if len(configs) == 0 {
		return nil
	}
	rows := make([]fullTableResolvedRow, 0, len(configs))
	for _, cfg := range configs {
		var snapshot *RenderMonitorSnapshot
		if frame != nil {
			snapshot = frame.ResolveMonitor(cfg.Monitor)
		}
		rows = append(rows, fullTableResolvedRow{
			fullTableRowConfig: cfg,
			MonitorSnapshot:    snapshot,
		})
	}
	return rows
}

func resolveFullTableLayout(item *ItemConfig, config *MonitorConfig) (int, int, float64, float64, string, string, float64, float64, bool) {
	if item != nil && item.runtime.prepared {
		runtime := item.runtime.fullTable
		return runtime.colCount, runtime.rowCount, runtime.rowGap, runtime.rowRadius, runtime.rowBg, runtime.rowAltBg, runtime.columnGap, runtime.labelWidthRatio, runtime.showUnits
	}
	colCount := resolveFullTableColCount(item, config)
	rows := resolveFullTableRowConfigs(item)
	return colCount,
		resolveFullTableRowCount(item, config, colCount, rows),
		clampMinFloat(getItemAttrFloatCfg(item, config, "table_row_gap", 0), 0),
		clampMinFloat(getItemAttrFloatCfg(item, config, "table_row_radius", 0), 0),
		getItemAttrColorCfg(item, config, "table_row_bg", ""),
		getItemAttrColorCfg(item, config, "table_row_alt_bg", ""),
		clampMinFloat(getItemAttrFloatCfg(item, config, "table_column_gap", 0), 0),
		clampFloat64(getItemAttrFloatCfg(item, config, "table_label_width_ratio", 0.46), 0.2, 0.7),
		getItemAttrBoolCfg(item, config, "table_show_units", true)
}

func resolveFullTableColCount(item *ItemConfig, config *MonitorConfig) int {
	colCount := getItemAttrIntCfg(item, config, "col_count", 1)
	if colCount < 1 {
		return 1
	}
	return colCount
}

func resolveFullTableRowCount(item *ItemConfig, config *MonitorConfig, colCount int, rows []fullTableRowConfig) int {
	if colCount < 1 {
		colCount = 1
	}
	explicit := getItemAttrIntCfg(item, config, "row_count", 0)
	computed := 1
	if len(rows) > 0 {
		computed = (len(rows) + colCount - 1) / colCount
	}
	if explicit < computed {
		return computed
	}
	if explicit < 1 {
		return computed
	}
	return explicit
}

func resolveFullTableRowLabel(row fullTableResolvedRow) string {
	if label := strings.TrimSpace(row.Label); label != "" {
		return label
	}
	if row.MonitorSnapshot != nil && strings.TrimSpace(row.MonitorSnapshot.label) != "" {
		return strings.TrimSpace(row.MonitorSnapshot.label)
	}
	return strings.TrimSpace(row.Monitor)
}

func resolveFullTableRowDisplay(row fullTableResolvedRow, showUnits bool) (string, string, bool) {
	if strings.TrimSpace(row.Monitor) == "" {
		return "", "", false
	}
	if row.MonitorSnapshot == nil || !row.MonitorSnapshot.available || row.MonitorSnapshot.value == nil {
		return "-", "", false
	}
	valueText, unitText := FormatCollectValueParts(row.MonitorSnapshot.value, "")
	if !showUnits {
		unitText = ""
	}
	return valueText, unitText, true
}

func resolveFullTableRowConfigs(item *ItemConfig) []fullTableRowConfig {
	if item == nil {
		return nil
	}
	if item.runtime.prepared {
		return append([]fullTableRowConfig(nil), item.runtime.fullTable.rows...)
	}

	rawRows, hasRows := getItemAttr(item, "rows")
	rows := parseFullTableRowsAttr(rawRows, hasRows)
	if len(rows) > 0 {
		return rows
	}

	monitor := normalizeMonitorAlias(item.Monitor)
	if monitor == "" {
		return nil
	}
	return []fullTableRowConfig{{
		Monitor: monitor,
		Label:   strings.TrimSpace(resolveItemLabelText(item, nil)),
	}}
}

func normalizeFullTableItemAttrs(item *ItemConfig) {
	if item == nil {
		return
	}
	if item.RenderAttrsMap == nil {
		item.RenderAttrsMap = map[string]interface{}{}
	}
	rawRows, hasRows := getItemAttr(item, "rows")
	rows := parseFullTableRowsAttr(rawRows, hasRows)
	colCount := resolveFullTableColCount(item, nil)
	rowCount := resolveFullTableRowCount(item, nil, colCount, rows)
	if len(rows) > 0 {
		item.RenderAttrsMap["rows"] = fullTableRowsToAttr(rows)
	} else {
		delete(item.RenderAttrsMap, "rows")
	}
	if colCount > 1 {
		item.RenderAttrsMap["col_count"] = colCount
	} else {
		delete(item.RenderAttrsMap, "col_count")
	}
	if rowCount > 1 {
		item.RenderAttrsMap["row_count"] = rowCount
	} else {
		delete(item.RenderAttrsMap, "row_count")
	}
}

func fullTableMonitorRefs(item *ItemConfig) []string {
	rows := resolveFullTableRowConfigs(item)
	if len(rows) == 0 {
		return nil
	}
	result := make([]string, 0, len(rows))
	seen := make(map[string]struct{}, len(rows))
	for _, row := range rows {
		name := normalizeMonitorAlias(row.Monitor)
		if name == "" {
			continue
		}
		if _, exists := seen[name]; exists {
			continue
		}
		seen[name] = struct{}{}
		result = append(result, name)
	}
	return result
}

func parseFullTableRowConfig(raw interface{}) (fullTableRowConfig, bool) {
	rowMap, ok := raw.(map[string]interface{})
	if !ok {
		return fullTableRowConfig{}, false
	}
	return fullTableRowConfig{
		Monitor: normalizeMonitorAlias(anyToString(rowMap["monitor"])),
		Label:   strings.TrimSpace(anyToString(rowMap["label"])),
	}, true
}

func parseFullTableRowsAttr(raw interface{}, exists bool) []fullTableRowConfig {
	if !exists || raw == nil {
		return nil
	}
	rows := make([]fullTableRowConfig, 0)
	switch value := raw.(type) {
	case []interface{}:
		for _, entry := range value {
			if row, ok := parseFullTableRowConfig(entry); ok {
				rows = append(rows, row)
			}
		}
	case []map[string]interface{}:
		for _, entry := range value {
			if row, ok := parseFullTableRowConfig(entry); ok {
				rows = append(rows, row)
			}
		}
	case []fullTableRowConfig:
		for _, row := range value {
			rows = append(rows, fullTableRowConfig{
				Monitor: normalizeMonitorAlias(row.Monitor),
				Label:   strings.TrimSpace(row.Label),
			})
		}
	}
	return rows
}

func fullTableRowsToAttr(rows []fullTableRowConfig) []map[string]interface{} {
	if len(rows) == 0 {
		return nil
	}
	out := make([]map[string]interface{}, 0, len(rows))
	for _, row := range rows {
		entry := map[string]interface{}{}
		if name := normalizeMonitorAlias(row.Monitor); name != "" {
			entry["monitor"] = name
		}
		if label := strings.TrimSpace(row.Label); label != "" {
			entry["label"] = label
		}
		out = append(out, entry)
	}
	return out
}

func fullTableRowsForGrid(rows []fullTableResolvedRow, cellCount int) []fullTableResolvedRow {
	if cellCount < 1 {
		cellCount = 1
	}
	out := make([]fullTableResolvedRow, cellCount)
	copy(out, rows)
	return out
}

func anyToString(raw interface{}) string {
	switch value := raw.(type) {
	case nil:
		return ""
	case string:
		return strings.TrimSpace(value)
	case fmt.Stringer:
		return strings.TrimSpace(value.String())
	default:
		return strings.TrimSpace(fmt.Sprintf("%v", raw))
	}
}

func drawFullTableGrid(
	dc *gg.Context,
	bodyRect fullRect,
	itemCount int,
	columns int,
	rowCount int,
	rowHeight float64,
	cellWidth float64,
	rowGap float64,
	columnGap float64,
	borderWidth float64,
	borderColor string,
) {
	if dc == nil || itemCount <= 0 || columns <= 0 || rowCount <= 0 || borderWidth <= 0 {
		return
	}
	dc.SetColor(parseColor(borderColor))
	dc.SetLineWidth(borderWidth)

	gridWidth := cellWidth*float64(columns) + columnGap*float64(max(0, columns-1))
	gridHeight := rowHeight*float64(rowCount) + rowGap*float64(max(0, rowCount-1))
	if gridWidth > bodyRect.w {
		gridWidth = bodyRect.w
	}
	if gridHeight > bodyRect.h {
		gridHeight = bodyRect.h
	}
	dc.DrawRectangle(bodyRect.x, bodyRect.y, gridWidth, gridHeight)
	dc.Stroke()

	for col := 1; col < columns; col++ {
		x := bodyRect.x + float64(col)*cellWidth + float64(col-1)*columnGap + columnGap/2
		dc.DrawLine(x, bodyRect.y, x, bodyRect.y+gridHeight)
		dc.Stroke()
	}
	for row := 1; row < rowCount; row++ {
		y := bodyRect.y + float64(row)*rowHeight + float64(row-1)*rowGap + rowGap/2
		dc.DrawLine(bodyRect.x, y, bodyRect.x+gridWidth, y)
		dc.Stroke()
	}
}
