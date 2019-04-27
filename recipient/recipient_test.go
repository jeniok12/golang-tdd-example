// recipient/recipient_test.go
package recipient

import (
	"database/sql"
	"fmt"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	"testing"
)

var testPersistence *Persistence
var expectedRecipients = []Recipient{
	{
		ID:    1,
		Name:  "user1",
		Email: "user1@testmail.com",
	},
	{
		ID:    2,
		Name:  "user2",
		Email: "user2@testmail.com",
	},
	{
		ID:    3,
		Name:  "user3",
		Email: "user3@testmail.com",
	},
}

func TestMain(m *testing.M) {
	var err error
	testPersistence, err = NewPersistence("localhost", "quotes_test")
	if err != nil {
		panic(err)
	}

	code := m.Run()

	os.Exit(code)
}

func TestAllRecipients(t *testing.T) {
	testCases := []struct {
		name               string
		presetDB           func(db *sql.DB) error
		expectedRecipients []Recipient
		err                error
	}{
		{
			"RecipientsFound",
			func(db *sql.DB) error {
				query := "INSERT INTO recipients (id, name, email) VALUES ($1, $2, $3);"
				tx, err := db.Begin()

				for _, r := range expectedRecipients {
					_, err = tx.Exec(query, r.ID, r.Name, r.Email)
					if err != nil {
						fmt.Println(fmt.Sprintf("Error: %+v", err))
					}
				}

				tx.Commit()
				return err
			},
			expectedRecipients,
			nil,
		},
		{
			"RecipientsNotFound",
			func(db *sql.DB) error {
				return nil
			},
			nil,
			nil,
		},
	}
	for _, tC := range testCases {
		t.Run(tC.name, func(t *testing.T) {
			err := clearDB(testPersistence.DB)
			require.NoErrorf(t, err, "Should have no error when cleaning the DB")

			err = tC.presetDB(testPersistence.DB)
			require.NoErrorf(t, err, "Should have no error when pre-setting the DB")

			recipients, err := testPersistence.AllRecipients()

			assert.Equal(t, err, tC.err, "Error should be as expected")
			assert.ElementsMatch(t, recipients, tC.expectedRecipients, "Response should be as expected")
		})
	}
}

func clearDB(db *sql.DB) error {
	_, err := db.Exec("TRUNCATE TABLE recipients")
	return err
}
