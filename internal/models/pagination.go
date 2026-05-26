package models

import (
	"net/http"
	"strconv"
)

type PageMeta struct {
	Page       int `json:"page"`
	PageSize   int `json:"pageSize"`
	Total      int `json:"total"`
	TotalPages int `json:"totalPages"`
}

type PagedResult[T any] struct {
	Data []T      `json:"data"`
	Meta PageMeta `json:"meta"`
}

func ParsePageQuery(r *http.Request, defaultSize, maxSize int) (page, pageSize int, paginate bool) {
	if r.URL.Query().Get("page") == "" && r.URL.Query().Get("pageSize") == "" {
		return 1, defaultSize, false
	}
	page = queryInt(r, "page", 1)
	pageSize = queryInt(r, "pageSize", defaultSize)
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = defaultSize
	}
	if pageSize > maxSize {
		pageSize = maxSize
	}
	return page, pageSize, true
}

func queryInt(r *http.Request, key string, fallback int) int {
	v := r.URL.Query().Get(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}

func PaginateSlice[T any](items []T, page, pageSize int) PagedResult[T] {
	total := len(items)
	if pageSize < 1 {
		pageSize = 50
	}
	if page < 1 {
		page = 1
	}
	start := (page - 1) * pageSize
	if start > total {
		start = total
	}
	end := start + pageSize
	if end > total {
		end = total
	}
	totalPages := 0
	if pageSize > 0 {
		totalPages = (total + pageSize - 1) / pageSize
	}
	return PagedResult[T]{
		Data: items[start:end],
		Meta: PageMeta{
			Page:       page,
			PageSize:   pageSize,
			Total:      total,
			TotalPages: totalPages,
		},
	}
}
