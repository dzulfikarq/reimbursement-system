// Package listq parses list-endpoint query params (docs/04): page, limit,
// search, sort, order. Sort fields must come from a per-route whitelist —
// never interpolated raw into SQL.
package listq

import (
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

const (
	defaultLimit = 10
	maxLimit     = 100
)

type Params struct {
	Page   int
	Limit  int
	Search string
	Sort   string // validated against whitelist; falls back to first allowed
	Order  string // ASC | DESC only
	Offset int
}

// Parse extracts and clamps params. sortWhitelist maps sortable param names to
// actual column expressions (safe: caller-provided literals, not user input).
func Parse(c *gin.Context, defaultSort string, sortWhitelist map[string]string) Params {
	p := Params{Sort: defaultSort, Order: "DESC"}

	if v, err := strconv.Atoi(c.Query("page")); err == nil && v > 0 {
		p.Page = v
	} else {
		p.Page = 1
	}
	if v, err := strconv.Atoi(c.Query("limit")); err == nil && v > 0 {
		p.Limit = min(v, maxLimit)
	} else {
		p.Limit = defaultLimit
	}
	p.Offset = (p.Page - 1) * p.Limit

	p.Search = strings.TrimSpace(c.Query("search"))

	if col, ok := sortWhitelist[c.Query("sort")]; ok {
		p.Sort = col
	}
	switch strings.ToUpper(c.Query("order")) {
	case "ASC":
		p.Order = "ASC"
	}

	return p
}

func (p Params) TotalPages(total int64) int {
	pages := int(total) / p.Limit
	if int(total)%p.Limit > 0 {
		pages++
	}
	if pages == 0 {
		pages = 1
	}
	return pages
}
