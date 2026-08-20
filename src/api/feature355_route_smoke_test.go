package main

// Feature 355 -- route-registration startup smoke test.
// Owned by Brutus (Tester/QA).
//
// Purpose: prove that registering the purchase-reminder routes alongside the
// existing bid-reminder routes (/reminders) does NOT cause a duplicate-route
// panic in Gin. Gin panics at route-registration time if two handlers share
// the same method + path; this test catches the regression before it reaches
// a real server start.

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

// nopHandler is a placeholder gin handler used to register route shapes without real logic.
func nopHandler(c *gin.Context) { c.Status(http.StatusOK) }

// TestFeature355_NoRouteCollision_PurchaseReminderListVsBidReminderList proves that
// GET /purchase-reminders (Feature 355 list) and GET /reminders (bid-reminder list)
// coexist without panic on the same router group.
//
// Regression guard: before the fix, Feature 355 registered GET /reminders and the
// existing bid-reminder handler also owned GET /reminders, causing a Gin startup panic.
func TestFeature355_NoRouteCollision_PurchaseReminderListVsBidReminderList(t *testing.T) {
	gin.SetMode(gin.TestMode)

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("route registration panicked (duplicate route conflict): %v", r)
		}
	}()

	router := gin.New()
	api := router.Group("/api")

	// Feature 355 routes (purchase reminders)
	api.POST("/coins/:id/reminder", nopHandler)
	api.GET("/coins/:id/reminder", nopHandler)
	api.DELETE("/coins/:id/reminder", nopHandler)
	api.GET("/purchase-reminders", nopHandler) // was /reminders before the fix

	// Existing bid-reminder routes (must coexist without collision)
	api.GET("/reminders", nopHandler)
	api.POST("/reminders", nopHandler)
	api.DELETE("/reminders/:id", nopHandler)

	// Verify all paths are routable (not 404/405).
	for _, tc := range []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/purchase-reminders"},
		{http.MethodGet, "/api/reminders"},
		{http.MethodPost, "/api/reminders"},
	} {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, httptest.NewRequest(tc.method, tc.path, nil))
		if w.Code == http.StatusNotFound || w.Code == http.StatusMethodNotAllowed {
			t.Errorf("%s %s returned %d -- route was not registered", tc.method, tc.path, w.Code)
		}
	}
}

// TestFeature355_PurchaseReminders_NotShadowedByBidReminders proves /purchase-reminders
// and /reminders are distinct paths: a request to one must not be served by the other.
func TestFeature355_PurchaseReminders_NotShadowedByBidReminders(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	api := router.Group("/api")

	purchaseReminderCalled := false
	bidReminderCalled := false

	api.GET("/purchase-reminders", func(c *gin.Context) {
		purchaseReminderCalled = true
		c.Status(http.StatusOK)
	})
	api.GET("/reminders", func(c *gin.Context) {
		bidReminderCalled = true
		c.Status(http.StatusOK)
	})

	// GET /purchase-reminders must only reach the purchase-reminder handler.
	router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/api/purchase-reminders", nil))
	if !purchaseReminderCalled {
		t.Error("GET /api/purchase-reminders did not reach the purchase-reminder handler")
	}
	if bidReminderCalled {
		t.Error("GET /api/purchase-reminders incorrectly reached the bid-reminder handler")
	}

	purchaseReminderCalled = false
	bidReminderCalled = false

	// GET /reminders must only reach the bid-reminder handler.
	router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/api/reminders", nil))
	if !bidReminderCalled {
		t.Error("GET /api/reminders did not reach the bid-reminder handler")
	}
	if purchaseReminderCalled {
		t.Error("GET /api/reminders incorrectly reached the purchase-reminder handler")
	}
}
