package engine_test

import (
	"context"
	"fmt"
	"maps"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/PuerkitoBio/goquery"
	"github.com/reddec/web-form/internal/engine"
	"github.com/reddec/web-form/internal/schema"
	"github.com/reddec/web-form/internal/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const def = `
name: code-access
table: code
description: Code-based access
fields:
  - name: name
    required: true
    default: "{{.Code}}"
  - name: year
    required: true
  - name: comment
    multiline: true
codes:
  - reddec

---
name: plain
table: plain
fields:
  - name: name
    required: true
  - name: year
    required: true
  - name: comment
    multiline: true
---
name: crash
table: crash
fields:
  - name: name
    required: true
  - name: year
    required: true
  - name: comment
    multiline: true
`

func TestBasic(t *testing.T) {
	storage := &mockStorage{
		failedTables: utils.NewSet("crash"),
	}
	forms, err := schema.FormsFromStream(strings.NewReader(def))
	require.NoError(t, err)

	srv, err := engine.New(engine.Config{
		Forms:   forms,
		Storage: storage,
		Listing: true,
	})
	require.NoError(t, err)

	t.Run("should show listing", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()

		srv.ServeHTTP(rec, req)
		require.Equal(t, http.StatusOK, rec.Code)

		doc, err := goquery.NewDocumentFromReader(rec.Body)
		require.NoError(t, err)
		assertHasElement(t, doc, `a[href="forms/code-access"]`)
	})

	t.Run("should show code", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/forms/code-access", nil)
		rec := httptest.NewRecorder()

		srv.ServeHTTP(rec, req)
		require.Equal(t, http.StatusUnauthorized, rec.Code)

		doc, err := goquery.NewDocumentFromReader(rec.Body)
		require.NoError(t, err)
		assertHasElement(t, doc, `input[name="_xsrf"]`)
		assertHasElement(t, doc, `input[name="accessCode"]`)
		assertHasElement(t, doc, `button[type="submit"]`)
	})

	t.Run("should show form", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/forms/plain", nil)
		rec := httptest.NewRecorder()

		srv.ServeHTTP(rec, req)
		require.Equal(t, http.StatusOK, rec.Code)

		doc, err := goquery.NewDocumentFromReader(rec.Body)
		require.NoError(t, err)
		assertHasElement(t, doc, `input[name="_xsrf"]`)
		assertHasElement(t, doc, `input[name="name"]`)
		assertHasElement(t, doc, `input[name="year"]`)
		assertHasElement(t, doc, `textarea[name="comment"]`)
		assertHasElement(t, doc, `button[type="submit"]`)
	})

	t.Run("should show result (success)", func(t *testing.T) {
		var params = make(url.Values)
		params.Set("_xsrf", "demo")
		params.Set("name", "RedDec")
		params.Set("year", "2023")
		params.Set("comment", "it works!")

		req := httptest.NewRequest(http.MethodPost, "/forms/plain", strings.NewReader(params.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.AddCookie(&http.Cookie{
			Name:  "_xsrf",
			Value: "demo",
		})
		rec := httptest.NewRecorder()

		srv.ServeHTTP(rec, req)
		require.Equal(t, http.StatusOK, rec.Code)

		doc, err := goquery.NewDocumentFromReader(rec.Body)
		require.NoError(t, err)
		assertHasElement(t, doc, `a[href="plain"]`)
	})

	t.Run("should show result (failed)", func(t *testing.T) {
		var params = make(url.Values)
		params.Set("_xsrf", "demo")
		params.Set("name", "RedDec")
		params.Set("year", "2023")
		params.Set("comment", "it works!")

		req := httptest.NewRequest(http.MethodPost, "/forms/crash", strings.NewReader(params.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.AddCookie(&http.Cookie{
			Name:  "_xsrf",
			Value: "demo",
		})
		rec := httptest.NewRecorder()

		srv.ServeHTTP(rec, req)
		require.Equal(t, http.StatusInternalServerError, rec.Code)

		doc, err := goquery.NewDocumentFromReader(rec.Body)
		require.NoError(t, err)
		assertHasElement(t, doc, `a[href="crash"]`)
	})
}

func assertHasElement(t *testing.T, doc *goquery.Document, selector string) {
	assert.True(t, doc.Find(selector).Length() > 0, "exists element: %q", selector)
}

type mockStorage struct {
	tables       sync.Map // table -> *mockTable
	failedTables utils.Set[string]
}

func (ms *mockStorage) Store(_ context.Context, table string, fields map[string]any) (map[string]any, error) {
	if ms.failedTables.Has(table) {
		return nil, fmt.Errorf("simulated error")
	}
	id := ms.getTable(table).Add(fields)
	s := maps.Clone(fields)
	s["id"] = id
	return s, nil
}

func (ms *mockStorage) getTable(name string) *mockTable {
	v, _ := ms.tables.LoadOrStore(name, &mockTable{})
	return v.(*mockTable)
}

type mockTable struct {
	id   atomic.Int64
	rows sync.Map // id -> map[string]any
}

func (mt *mockTable) Add(row map[string]any) int64 {
	id := mt.id.Add(1)
	mt.rows.Store(id, row)
	return id
}
