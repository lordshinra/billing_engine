package dao

import (
	"time"
)

type Loan struct {
	ID                int64     `gorm:"primaryKey;type:bigint(36)"`
	LoanID            string    `gorm:"unique;not null;type:varchar(100)"`
	CustomerID        string    `gorm:"not null;type:varchar(100)"`
	PrincipalAmount   int64     `gorm:"not null"`
	InterestRate      float64   `gorm:"not null;type:decimal(5,2)"`
	TotalAmount       int64     `gorm:"not null"`
	OutstandingAmount int64     `gorm:"not null"`
	WeeklyInstallment int64     `gorm:"not null"`
	TenureWeeks       int       `gorm:"not null"`
	Status            string    `gorm:"not null;type:varchar(20)"`
	CreatedAt         time.Time `gorm:"not null"`
	UpdatedAt         time.Time `gorm:"not null"`
}

type LoanSchedule struct {
	ID         int64     `gorm:"primaryKey;type:bigint(36)"`
	LoanID     string    `gorm:"not null;type:varchar(100);uniqueIndex:idx_loan_week"`
	WeekNumber int       `gorm:"not null;uniqueIndex:idx_loan_week"`
	DueAmount  int64     `gorm:"not null"`
	DueDate    time.Time `gorm:"not null"`
	Paid       bool      `gorm:"default:false"`
	PaidAt     *time.Time
	CreatedAt  time.Time `gorm:"not null"`
}

type Payment struct {
	ID        int64     `gorm:"primaryKey;type:bigint(36)"`
	LoanID    string    `gorm:"not null;type:varchar(100)"`
	Amount    int64     `gorm:"not null"`
	CreatedAt time.Time `gorm:"not null"`
}
