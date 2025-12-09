package bloom

import (
	"database/sql"

	bf "github.com/bits-and-blooms/bloom/v3"
)

var (
	Filter  *bf.BloomFilter
	Enabled bool
)

func InitBloom(capacity uint, falsePositiveRate float64) {
	Filter = bf.NewWithEstimates(capacity, falsePositiveRate)
	Enabled = false
}

func Populate(db *sql.DB) error {
	rows, err := db.Query("SELECT long_url FROM urls")
	if err != nil {
		return err
	}
	defer rows.Close()

	var url string
	var count uint

	for rows.Next() {
		if err := rows.Scan(&url); err != nil {
			continue
		}
		Filter.AddString(url)
		count++
	}
	Enabled = true
	return rows.Err()
}

func Add(url string) {
	if Enabled {
		Filter.AddString(url)
	}
}

func MightExist(url string) bool {
	return Enabled && Filter.TestString(url)
}
