package output

import (
	"fmt"
	"sort"
	"strings"
)

const (
	TypeMemImg   = "memimg"
	TypeAX206USB = "ax206usb"
	TypeHTTPPush = "httppush"
	TypeTCPPush  = "tcppush"
)

type ConfigSummary struct {
	Configs   []OutputConfig
	Types     []string
	HasMemImg bool
}

type HTTPKeyValue struct {
	Key   string `json:"key"`
	Value string `json:"value,omitempty"`
}

type OutputConfig struct {
	Type           string         `json:"type"`
	Enabled        *bool          `json:"enabled,omitempty"`
	URL            string         `json:"url,omitempty"`
	Method         string         `json:"method,omitempty"`
	BodyMode       string         `json:"body_mode,omitempty"`
	Format         string         `json:"format,omitempty"`
	Quality        int            `json:"quality,omitempty"`
	ContentType    string         `json:"content_type,omitempty"`
	Headers        []HTTPKeyValue `json:"headers,omitempty"`
	AuthType       string         `json:"auth_type,omitempty"`
	AuthUsername   string         `json:"auth_username,omitempty"`
	AuthPassword   string         `json:"auth_password,omitempty"`
	AuthToken      string         `json:"auth_token,omitempty"`
	UploadToken    string         `json:"upload_token,omitempty"`
	TimeoutMS      int            `json:"timeout_ms,omitempty"`
	IdleTimeoutSec int            `json:"idle_timeout_sec,omitempty"`
	BusyCheckMS    int            `json:"busy_check_ms,omitempty"`
	FileField      string         `json:"file_field,omitempty"`
	FileName       string         `json:"file_name,omitempty"`
	FormFields     []HTTPKeyValue `json:"form_fields,omitempty"`
	SuccessCodes   []int          `json:"success_codes,omitempty"`
	ReconnectMS    int            `json:"reconnect_ms,omitempty"`
}

func normalizeOutputTypeName(typeName string) string {
	return strings.ToLower(strings.TrimSpace(typeName))
}

func normalizeHTTPPushFormat(format string) string {
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "png":
		return "png"
	case "jpeg_baseline", "jpg_baseline", "baseline_jpeg", "baseline_jpg":
		return "jpeg_baseline"
	case "jpg", "jpeg":
		return "jpeg"
	default:
		return "jpeg"
	}
}

func normalizeHTTPPushMethod(method string) string {
	value := strings.ToUpper(strings.TrimSpace(method))
	if value == "" {
		return "POST"
	}
	for _, ch := range value {
		if (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') {
			continue
		}
		switch ch {
		case '!', '#', '$', '%', '&', '\'', '*', '+', '-', '.', '^', '_', '`', '|', '~':
			continue
		default:
			return "POST"
		}
	}
	return value
}

func normalizeHTTPPushBodyMode(mode string) string {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "multipart", "form", "formdata", "multipart_form":
		return "multipart"
	case "binary", "raw", "bytes":
		return "binary"
	default:
		return "binary"
	}
}

func normalizeHTTPPushAuthType(authType string) string {
	switch strings.ToLower(strings.TrimSpace(authType)) {
	case "basic":
		return "basic"
	case "bearer":
		return "bearer"
	default:
		return "none"
	}
}

func normalizeTCPPushFormat(format string) string {
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "jpeg", "jpg", "jpeg_baseline", "jpg_baseline", "baseline_jpeg", "baseline_jpg":
		return "jpeg"
	case "rgb565", "rgb565le":
		return "rgb565le"
	case "rgb565_rle", "rgb565le_rle":
		return "rgb565le_rle"
	case "index8", "index8_rle", "palette8_rle":
		return "index8_rle"
	default:
		return "jpeg"
	}
}

func normalizeTCPPushIdleTimeoutSec(idleTimeoutSec int) int {
	if idleTimeoutSec <= 0 {
		return 120
	}
	if idleTimeoutSec < 5 {
		return 5
	}
	if idleTimeoutSec > 3600 {
		return 3600
	}
	return idleTimeoutSec
}

func normalizeTCPPushBusyCheckMS(busyCheckMS int) int {
	if busyCheckMS <= 0 {
		return 1000
	}
	if busyCheckMS < 100 {
		return 100
	}
	if busyCheckMS > 600000 {
		return 600000
	}
	return busyCheckMS
}

func normalizeHTTPPushQuality(quality int) int {
	if quality <= 0 {
		return 80
	}
	if quality < 1 {
		return 1
	}
	if quality > 100 {
		return 100
	}
	return quality
}

func normalizeHTTPPushTimeoutMS(timeoutMS int) int {
	if timeoutMS <= 0 {
		return 5000
	}
	if timeoutMS < 100 {
		return 100
	}
	if timeoutMS > 600000 {
		return 600000
	}
	return timeoutMS
}

func normalizeHTTPPushContentType(contentType string) string {
	return strings.TrimSpace(contentType)
}

func normalizeHTTPPushFileField(fileField string) string {
	value := strings.TrimSpace(fileField)
	if value == "" {
		return "file"
	}
	return value
}

func normalizeHTTPPushFileName(fileName string) string {
	return strings.TrimSpace(fileName)
}

func normalizeHTTPPushKeyValues(items []HTTPKeyValue) []HTTPKeyValue {
	if len(items) == 0 {
		return []HTTPKeyValue{}
	}
	normalized := make([]HTTPKeyValue, 0, len(items))
	for _, item := range items {
		key := strings.TrimSpace(item.Key)
		if key == "" {
			continue
		}
		normalized = append(normalized, HTTPKeyValue{
			Key:   key,
			Value: strings.TrimSpace(item.Value),
		})
	}
	if len(normalized) == 0 {
		return []HTTPKeyValue{}
	}
	return normalized
}

func normalizeHTTPPushSuccessCodes(codes []int) []int {
	if len(codes) == 0 {
		return []int{}
	}
	normalized := make([]int, 0, len(codes))
	seen := map[int]struct{}{}
	for _, code := range codes {
		if code < 100 || code > 599 {
			continue
		}
		if _, exists := seen[code]; exists {
			continue
		}
		seen[code] = struct{}{}
		normalized = append(normalized, code)
	}
	if len(normalized) == 0 {
		return []int{}
	}
	sort.Ints(normalized)
	return normalized
}

func normalizeAX206ReconnectMS(reconnectMS int) int {
	if reconnectMS <= 0 {
		return 3000
	}
	if reconnectMS < 100 {
		return 100
	}
	if reconnectMS > 60000 {
		return 60000
	}
	return reconnectMS
}

func cloneEnabledValue(enabled bool) *bool {
	value := enabled
	return &value
}

func isConfigEnabled(cfg OutputConfig) bool {
	return cfg.Enabled == nil || *cfg.Enabled
}

func normalizeSingleConfig(raw OutputConfig) (OutputConfig, bool) {
	cfg := OutputConfig{
		Type:    normalizeOutputTypeName(raw.Type),
		Enabled: cloneEnabledValue(isConfigEnabled(raw)),
	}
	switch cfg.Type {
	case TypeMemImg:
		return cfg, true
	case TypeAX206USB:
		cfg.ReconnectMS = normalizeAX206ReconnectMS(raw.ReconnectMS)
		return cfg, true
	case TypeHTTPPush:
		cfg.URL = strings.TrimSpace(raw.URL)
		cfg.Method = normalizeHTTPPushMethod(raw.Method)
		cfg.BodyMode = normalizeHTTPPushBodyMode(raw.BodyMode)
		cfg.Format = normalizeHTTPPushFormat(raw.Format)
		cfg.Quality = normalizeHTTPPushQuality(raw.Quality)
		cfg.ContentType = normalizeHTTPPushContentType(raw.ContentType)
		cfg.Headers = normalizeHTTPPushKeyValues(raw.Headers)
		cfg.AuthType = normalizeHTTPPushAuthType(raw.AuthType)
		cfg.AuthUsername = strings.TrimSpace(raw.AuthUsername)
		cfg.AuthPassword = strings.TrimSpace(raw.AuthPassword)
		cfg.AuthToken = strings.TrimSpace(raw.AuthToken)
		cfg.TimeoutMS = normalizeHTTPPushTimeoutMS(raw.TimeoutMS)
		cfg.FileField = normalizeHTTPPushFileField(raw.FileField)
		cfg.FileName = normalizeHTTPPushFileName(raw.FileName)
		cfg.FormFields = normalizeHTTPPushKeyValues(raw.FormFields)
		cfg.SuccessCodes = normalizeHTTPPushSuccessCodes(raw.SuccessCodes)
		return cfg, true
	case TypeTCPPush:
		cfg.URL = strings.TrimSpace(raw.URL)
		cfg.Format = normalizeTCPPushFormat(raw.Format)
		cfg.Quality = normalizeHTTPPushQuality(raw.Quality)
		cfg.UploadToken = strings.TrimSpace(raw.UploadToken)
		cfg.TimeoutMS = normalizeHTTPPushTimeoutMS(raw.TimeoutMS)
		cfg.IdleTimeoutSec = normalizeTCPPushIdleTimeoutSec(raw.IdleTimeoutSec)
		cfg.BusyCheckMS = normalizeTCPPushBusyCheckMS(raw.BusyCheckMS)
		cfg.FileName = normalizeHTTPPushFileName(raw.FileName)
		cfg.SuccessCodes = normalizeHTTPPushSuccessCodes(raw.SuccessCodes)
		return cfg, true
	default:
		return OutputConfig{}, false
	}
}

func NormalizeConfigs(configs []OutputConfig) []OutputConfig {
	normalized := make([]OutputConfig, 0, len(configs))
	seenSingleton := map[string]struct{}{}

	for _, raw := range configs {
		cfg, ok := normalizeSingleConfig(raw)
		if !ok {
			continue
		}
		if _, exists := seenSingleton[cfg.Type]; exists {
			continue
		}
		seenSingleton[cfg.Type] = struct{}{}
		normalized = append(normalized, cfg)
	}

	if len(normalized) == 0 {
		return []OutputConfig{}
	}
	return normalized
}

func EnabledConfigs(configs []OutputConfig) []OutputConfig {
	normalized := NormalizeConfigs(configs)
	enabled := make([]OutputConfig, 0, len(normalized))
	for _, cfg := range normalized {
		if !isConfigEnabled(cfg) {
			continue
		}
		enabled = append(enabled, cfg)
	}
	if len(enabled) == 0 {
		return []OutputConfig{}
	}
	return enabled
}

func ResolveConfigs(configs []OutputConfig, forceMemImg bool) []OutputConfig {
	resolved := EnabledConfigs(configs)
	if !forceMemImg {
		return resolved
	}
	for _, cfg := range resolved {
		if cfg.Type == TypeMemImg {
			return resolved
		}
	}
	return append(resolved, OutputConfig{Type: TypeMemImg})
}

func ConfigsFromTypes(types []string) []OutputConfig {
	configs := make([]OutputConfig, 0, len(types))
	for _, item := range types {
		configs = append(configs, OutputConfig{Type: item, Enabled: cloneEnabledValue(true)})
	}
	return configs
}

func TypeNames(configs []OutputConfig) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(configs))
	for _, item := range configs {
		typeName := normalizeOutputTypeName(item.Type)
		if typeName == "" {
			continue
		}
		if _, exists := seen[typeName]; exists {
			continue
		}
		seen[typeName] = struct{}{}
		out = append(out, typeName)
	}
	return out
}

func HasType(configs []OutputConfig, typeName string) bool {
	target := normalizeOutputTypeName(typeName)
	if target == "" {
		return false
	}
	for _, item := range configs {
		if normalizeOutputTypeName(item.Type) == target {
			return true
		}
	}
	return false
}

func DescribeConfigs(configs []OutputConfig) ConfigSummary {
	return ConfigSummary{
		Configs:   configs,
		Types:     TypeNames(configs),
		HasMemImg: HasType(configs, TypeMemImg),
	}
}

func EqualConfigs(left, right []OutputConfig) bool {
	if len(left) != len(right) {
		return false
	}
	for idx := range left {
		lCfg, lOK := normalizeSingleConfig(left[idx])
		rCfg, rOK := normalizeSingleConfig(right[idx])
		if !lOK || !rOK {
			return false
		}
		if lCfg.Type != rCfg.Type {
			return false
		}
		if isConfigEnabled(lCfg) != isConfigEnabled(rCfg) {
			return false
		}
		if lCfg.URL != rCfg.URL {
			return false
		}
		if lCfg.Method != rCfg.Method {
			return false
		}
		if lCfg.BodyMode != rCfg.BodyMode {
			return false
		}
		if lCfg.Format != rCfg.Format {
			return false
		}
		if lCfg.Quality != rCfg.Quality {
			return false
		}
		if lCfg.ContentType != rCfg.ContentType {
			return false
		}
		if !equalHTTPKeyValues(lCfg.Headers, rCfg.Headers) {
			return false
		}
		if lCfg.AuthType != rCfg.AuthType {
			return false
		}
		if lCfg.AuthUsername != rCfg.AuthUsername {
			return false
		}
		if lCfg.AuthPassword != rCfg.AuthPassword {
			return false
		}
		if lCfg.AuthToken != rCfg.AuthToken {
			return false
		}
		if lCfg.UploadToken != rCfg.UploadToken {
			return false
		}
		if lCfg.TimeoutMS != rCfg.TimeoutMS {
			return false
		}
		if lCfg.IdleTimeoutSec != rCfg.IdleTimeoutSec {
			return false
		}
		if lCfg.FileField != rCfg.FileField {
			return false
		}
		if lCfg.FileName != rCfg.FileName {
			return false
		}
		if !equalHTTPKeyValues(lCfg.FormFields, rCfg.FormFields) {
			return false
		}
		if !equalIntSlice(lCfg.SuccessCodes, rCfg.SuccessCodes) {
			return false
		}
		if lCfg.ReconnectMS != rCfg.ReconnectMS {
			return false
		}
	}
	return true
}

func equalHTTPKeyValues(left, right []HTTPKeyValue) bool {
	if len(left) != len(right) {
		return false
	}
	for idx := range left {
		if left[idx].Key != right[idx].Key {
			return false
		}
		if left[idx].Value != right[idx].Value {
			return false
		}
	}
	return true
}

func equalIntSlice(left, right []int) bool {
	if len(left) != len(right) {
		return false
	}
	for idx := range left {
		if left[idx] != right[idx] {
			return false
		}
	}
	return true
}

func NormalizeTypes(types []string) []string {
	return TypeNames(NormalizeConfigs(ConfigsFromTypes(types)))
}

func EnabledTypeNames(configs []OutputConfig) []string {
	return TypeNames(EnabledConfigs(configs))
}

func ResolveTypes(types []string, forceMemImg bool) []string {
	return TypeNames(ResolveConfigs(ConfigsFromTypes(types), forceMemImg))
}

func ResolveConfigSummary(configs []OutputConfig, forceMemImg bool) ConfigSummary {
	return DescribeConfigs(ResolveConfigs(configs, forceMemImg))
}

func BuildManager(configs []OutputConfig, forceMemImg bool) (*OutputManager, []OutputConfig) {
	summary := ResolveConfigSummary(configs, forceMemImg)
	manager := NewOutputManager()

	httpPushIndex := 0
	tcpPushIndex := 0
	for _, cfg := range summary.Configs {
		switch cfg.Type {
		case TypeMemImg:
			manager.AddHandler(NewMemImgOutputHandler())
		case TypeAX206USB:
			handler, err := NewAX206USBOutputHandler(cfg)
			if err != nil {
				logErrorModule("ax206usb", "Handler creation failed: %v", err)
				continue
			}
			manager.AddHandler(handler)
		case TypeHTTPPush:
			httpPushIndex++
			typeName := TypeHTTPPush
			if httpPushIndex > 1 {
				typeName = fmt.Sprintf("%s_%d", TypeHTTPPush, httpPushIndex)
			}
			handler := NewHTTPPushOutputHandler(cfg, typeName)
			manager.AddHandler(handler)
		case TypeTCPPush:
			tcpPushIndex++
			typeName := TypeTCPPush
			if tcpPushIndex > 1 {
				typeName = fmt.Sprintf("%s_%d", TypeTCPPush, tcpPushIndex)
			}
			handler := NewTCPPushOutputHandler(cfg, typeName)
			manager.AddHandler(handler)
		}
	}

	return manager, summary.Configs
}
