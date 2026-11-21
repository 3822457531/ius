// model/price_history.go

package model

import "time"

type PriceHistory struct {
	Id         uint      `gorm:"primarykey"`
	Date       time.Time `gorm:"type:date;not null"`
	Price      float64   `gorm:"type:decimal(20,6);not null"`
	Volume     float64   `gorm:"type:decimal(20,2);default:0"`
	ChangeRate float64   `gorm:"type:decimal(10,2);default:0"`
	CreatedAt  time.Time
	UpdatedAt  time.Time
}
