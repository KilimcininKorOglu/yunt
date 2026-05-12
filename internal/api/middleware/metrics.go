package middleware

import (
	"fmt"
	"strings"
	"time"

	"github.com/labstack/echo/v4"

	"yunt/internal/metrics"
)

// Metrics returns an Echo middleware that records Prometheus HTTP metrics.
func Metrics() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			metrics.HTTPActiveRequests.Inc()
			defer metrics.HTTPActiveRequests.Dec()

			start := time.Now()
			err := next(c)

			duration := time.Since(start).Seconds()
			status := c.Response().Status
			if err != nil {
				if he, ok := err.(*echo.HTTPError); ok {
					status = he.Code
				}
			}

			path := normalizePath(c.Path())
			method := c.Request().Method

			metrics.HTTPRequestsTotal.WithLabelValues(method, path, fmt.Sprintf("%d", status)).Inc()
			metrics.HTTPRequestDuration.WithLabelValues(method, path).Observe(duration)

			return err
		}
	}
}

// normalizePath replaces dynamic path segments with placeholders to prevent high cardinality.
func normalizePath(path string) string {
	if path == "" {
		return "/"
	}
	parts := strings.Split(path, "/")
	for i, part := range parts {
		if strings.HasPrefix(part, ":") {
			parts[i] = "{" + part[1:] + "}"
		}
	}
	return strings.Join(parts, "/")
}
