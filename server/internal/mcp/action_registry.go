package mcp

import (
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/labstack/echo/v4"
)

type ActionDefinition struct {
	ID          string   `json:"id"`
	Method      string   `json:"method"`
	Path        string   `json:"path"`
	Description string   `json:"description"`
	PathParams  []string `json:"path_params"`
	ReadOnly    bool     `json:"read_only"`
	Destructive bool     `json:"destructive"`
}

type ActionRegistry struct {
	actions []ActionDefinition
	byID    map[string]ActionDefinition
}

func NewActionRegistry(e *echo.Echo) *ActionRegistry {
	routes := e.Routes()
	actions := make([]ActionDefinition, 0, len(routes))
	seen := make(map[string]int)
	for _, route := range routes {
		if !strings.HasPrefix(route.Path, "/api") {
			continue
		}
		if route.Method == echo.RouteNotFound {
			continue
		}
		id := actionID(route.Method, route.Path)
		if n := seen[id]; n > 0 {
			seen[id] = n + 1
			id = id + "_" + strconv.Itoa(n+1)
		} else {
			seen[id] = 1
		}
		actions = append(actions, ActionDefinition{
			ID:          id,
			Method:      route.Method,
			Path:        route.Path,
			Description: route.Method + " " + route.Path,
			PathParams:  extractPathParams(route.Path),
			ReadOnly:    route.Method == "GET" || route.Method == "HEAD" || route.Method == "OPTIONS",
			Destructive: route.Method == "DELETE",
		})
	}
	sort.Slice(actions, func(i, j int) bool {
		if actions[i].Path == actions[j].Path {
			return actions[i].Method < actions[j].Method
		}
		return actions[i].Path < actions[j].Path
	})
	byID := make(map[string]ActionDefinition, len(actions))
	for _, action := range actions {
		byID[action.ID] = action
	}
	return &ActionRegistry{actions: actions, byID: byID}
}

func (r *ActionRegistry) All() []ActionDefinition {
	result := make([]ActionDefinition, len(r.actions))
	copy(result, r.actions)
	return result
}

func (r *ActionRegistry) Get(id string) (ActionDefinition, bool) {
	action, ok := r.byID[id]
	return action, ok
}

func (r *ActionRegistry) List(filter string, limit, offset int) ([]ActionDefinition, int) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	filter = strings.ToLower(strings.TrimSpace(filter))
	filtered := make([]ActionDefinition, 0, len(r.actions))
	for _, action := range r.actions {
		if filter == "" || strings.Contains(strings.ToLower(action.ID), filter) || strings.Contains(strings.ToLower(action.Path), filter) || strings.Contains(strings.ToLower(action.Method), filter) {
			filtered = append(filtered, action)
		}
	}
	total := len(filtered)
	if offset >= total {
		return []ActionDefinition{}, total
	}
	end := offset + limit
	if end > total {
		end = total
	}
	return filtered[offset:end], total
}

func actionID(method, path string) string {
	parts := []string{strings.ToLower(method)}
	trimmed := strings.Trim(strings.TrimPrefix(path, "/api"), "/")
	if trimmed == "" {
		return strings.ToLower(method) + "_api_root"
	}
	for _, segment := range strings.Split(trimmed, "/") {
		if segment == "" {
			continue
		}
		if strings.HasPrefix(segment, ":") {
			parts = append(parts, "by", sanitizeIDSegment(strings.TrimPrefix(segment, ":")))
			continue
		}
		parts = append(parts, sanitizeIDSegment(segment))
	}
	return strings.Join(parts, "_")
}

var idSegmentPattern = regexp.MustCompile(`[^a-zA-Z0-9]+`)

func sanitizeIDSegment(segment string) string {
	segment = idSegmentPattern.ReplaceAllString(segment, "_")
	segment = strings.Trim(segment, "_")
	if segment == "" {
		return "segment"
	}
	return strings.ToLower(segment)
}

func extractPathParams(path string) []string {
	var params []string
	for _, segment := range strings.Split(path, "/") {
		if strings.HasPrefix(segment, ":") {
			params = append(params, strings.TrimPrefix(segment, ":"))
		}
	}
	return params
}
