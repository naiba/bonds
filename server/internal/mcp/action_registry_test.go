package mcp

import (
	"net/http"
	"testing"

	"github.com/labstack/echo/v4"
)

func TestActionRegistryDiscoversAPIRoutesOnly(t *testing.T) {
	e := echo.New()
	e.GET("/api/vaults", func(c echo.Context) error { return c.NoContent(http.StatusOK) })
	e.POST("/api/vaults/:vault_id/contacts", func(c echo.Context) error { return c.NoContent(http.StatusCreated) })
	e.GET("/swagger/*", func(c echo.Context) error { return c.NoContent(http.StatusOK) })
	e.POST("/mcp", func(c echo.Context) error { return c.NoContent(http.StatusOK) })

	registry := NewActionRegistry(e)
	actions := registry.All()
	if len(actions) != 2 {
		t.Fatalf("expected 2 API actions, got %d: %+v", len(actions), actions)
	}
	for _, action := range actions {
		if action.Method == echo.RouteNotFound {
			t.Fatalf("registry must skip Echo RouteNotFound entries, got %+v", action)
		}
	}
	if _, ok := registry.Get("get_vaults"); !ok {
		t.Fatal("expected get_vaults action")
	}
	action, ok := registry.Get("post_vaults_by_vault_id_contacts")
	if !ok {
		t.Fatal("expected post_vaults_by_vault_id_contacts action")
	}
	if len(action.PathParams) != 1 || action.PathParams[0] != "vault_id" {
		t.Fatalf("expected vault_id path param, got %+v", action.PathParams)
	}
	if action.ReadOnly {
		t.Fatal("POST action must not be marked read-only")
	}
}
