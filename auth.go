package main

import (
	"log"

	"golang.org/x/crypto/bcrypt"
)

type Account struct {
	Id             string
	Email          string
	HashedPassword []byte
}

func CreateAccount(email, password string) (*Account, error) {
	var id string

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	query := "INSERT INTO account (email, password) VALUES ($1, $2) RETURNING id"
	if err := db.QueryRow(query, email, string(hashedPassword)).Scan(&id); err != nil {
		return nil, err
	}

	return &Account{id, email, hashedPassword}, nil
}

func GetAccount(email string) (*Account, error) {
	var id, password string

	query := "SELECT id, password FROM account WHERE email = $1"
	if err := db.QueryRow(query, email).Scan(&id, &password); err != nil {
		return nil, err
	}

	return &Account{id, email, []byte(password)}, nil
}

func (a *Account) ValidatePassword(password []byte) bool {
	if err := bcrypt.CompareHashAndPassword(a.HashedPassword, password); err != nil {
		log.Print(err)
		return false
	}
	return true
}
