package repository

import (
	"context"
	"database/sql"
	"log"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

func SeedAdmin(ctx context.Context, db *sql.DB, email, password string) {
	var exists bool
	queryCheck := `SELECT EXISTS(SELECT 1 FROM users WHERE LOWER(email) = LOWER($1))`
	err := db.QueryRowContext(ctx, queryCheck, email).Scan(&exists)
	if err != nil {
		log.Printf("Gagal cek admin seed: %v", err)
		return
	}

	if !exists {
		hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		queryInsert := `INSERT INTO users (name, email, password_hash, role) VALUES ($1, $2, $3, 'admin')`
		_, err = db.ExecContext(ctx, queryInsert, "Super Admin", strings.ToLower(email), string(hashedPassword))
		if err != nil {
			log.Printf("Gagal membuat admin seed: %v", err)
		} else {
			log.Println("Akun admin pertama berhasil dibuat")
		}
	} else {
		log.Println("Akun admin sudah ada, skip seeding")
	}
}
