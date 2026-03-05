package renderers

// getString safely extracts a string value from a map.
func getString(data map[string]any, key string) string {
	if v, ok := data[key].(string); ok {
		return v
	}
	return ""
}

// getFloat64 safely extracts a float64 value from a map.
func getFloat64(data map[string]any, key string) float64 {
	if v, ok := data[key].(float64); ok {
		return v
	}
	return 0
}

// getMap safely extracts a nested map from a map.
func getMap(data map[string]any, key string) map[string]any {
	if v, ok := data[key].(map[string]any); ok {
		return v
	}
	return nil
}
