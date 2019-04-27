// recipient/recipient.go

package recipient

import (
	"database/sql"
	"fmt"
)

// Recipient ...
type Recipient struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

// Persistence ...
type Persistence struct {
	DB *sql.DB
}

// NewPersistence ...
func NewPersistence(host, dbName string) (*Persistence, error) {
	db, err := sql.Open("postgres", fmt.Sprintf("dbname=%s host=%s sslmode=disable", dbName, host))
	if err != nil {
		return nil, err
	}

	return &Persistence{
		DB: db,
	}, nil
}

// AllRecipients ...
func (p *Persistence) AllRecipients() ([]Recipient, error) {
	var recipients []Recipient

	rows, err := p.DB.Query("select * from recipients")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var r Recipient
		if err := rows.Scan(&r.ID, &r.Name, &r.Email); err == nil {
			recipients = append(recipients, r)
		}
	}

	return recipients, nil
}
