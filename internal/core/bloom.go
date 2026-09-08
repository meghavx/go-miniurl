package core

import (
	"database/sql"

	bf "github.com/bits-and-blooms/bloom/v3"
)

var (
	Filter       *bf.BloomFilter
	EnabledBloom bool
)

func InitBloom(capacity uint, falsePositiveRate float64) {
	Filter = bf.NewWithEstimates(capacity, falsePositiveRate)
	EnabledBloom = false
}

func PopulateBloom(db *sql.DB) error {
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
	EnabledBloom = true
	return rows.Err()
}

func AddToBloom(url string) {
	if EnabledBloom {
		Filter.AddString(url)
	}
}

func MightExistInBloom(url string) bool {
	return EnabledBloom && Filter.TestString(url)
}
