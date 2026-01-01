//go:build tools

// Package tools imports dependencies that are used indirectly by the project.
// This file ensures that go mod tidy keeps these dependencies in go.mod.
// See: https://github.com/golang/go/wiki/Modules#how-can-i-track-tool-dependencies-for-a-module
package tools

import (
	// SMTP server
	_ "github.com/emersion/go-sasl"
	_ "github.com/emersion/go-smtp"

	// IMAP server
	_ "github.com/emersion/go-imap/v2"

	// Mail parsing
	_ "github.com/emersion/go-message"
	_ "github.com/jhillyerd/enmime"

	// HTTP router
	_ "github.com/labstack/echo/v4"

	// Database - SQL
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"

	// Database - MongoDB
	_ "go.mongodb.org/mongo-driver/mongo"

	// Authentication
	_ "github.com/golang-jwt/jwt/v5"
	_ "golang.org/x/crypto/bcrypt"

	// Configuration
	_ "github.com/spf13/cobra"
	_ "github.com/spf13/viper"

	// Logging
	_ "github.com/rs/zerolog"

	// Utilities
	_ "github.com/google/uuid"
	_ "github.com/gorilla/websocket"
)
