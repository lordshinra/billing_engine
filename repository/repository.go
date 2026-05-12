package repository

import (
	"context"
	"errors"

	"billing_service/dao"

	"gorm.io/gorm"
)

var ErrNotFound = errors.New("record not found")

type Repository interface {
	CreateLoanTx(ctx context.Context, loan *dao.Loan, schedules []*dao.LoanSchedule) error
	GetLoan(ctx context.Context, loanID string) (*dao.Loan, error)
	GetUnpaidSchedules(ctx context.Context, loanID string) ([]*dao.LoanSchedule, error)
	GetLastTwoDueSchedules(ctx context.Context, loanID string) ([]*dao.LoanSchedule, error)
	UpdatePaymentTx(ctx context.Context, loan *dao.Loan, schedules []*dao.LoanSchedule, payment *dao.Payment) error
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

func (r *repository) CreateLoanTx(ctx context.Context, loan *dao.Loan, schedules []*dao.LoanSchedule) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(loan).Error; err != nil {
			return err
		}
		if err := tx.Create(&schedules).Error; err != nil {
			return err
		}
		return nil
	})
}

func (r *repository) GetLoan(ctx context.Context, loanID string) (*dao.Loan, error) {
	var loan dao.Loan
	if err := r.db.WithContext(ctx).Where("loan_id = ?", loanID).First(&loan).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &loan, nil
}

func (r *repository) GetUnpaidSchedules(ctx context.Context, loanID string) ([]*dao.LoanSchedule, error) {
	var schedules []*dao.LoanSchedule
	if err := r.db.WithContext(ctx).
		Where("loan_id = ? AND paid = ?", loanID, false).
		Order("week_number ASC").
		Find(&schedules).Error; err != nil {
		return nil, err
	}
	return schedules, nil
}

func (r *repository) GetLastTwoDueSchedules(ctx context.Context, loanID string) ([]*dao.LoanSchedule, error) {
	var schedules []*dao.LoanSchedule
	// Get the last 2 due schedules, ordered by week number descending
	if err := r.db.WithContext(ctx).
		Where("loan_id = ? AND due_date <= ?", loanID, gorm.Expr("CURRENT_TIMESTAMP")).
		Order("week_number DESC").
		Limit(2).
		Find(&schedules).Error; err != nil {
		return nil, err
	}
	return schedules, nil
}

func (r *repository) UpdatePaymentTx(ctx context.Context, loan *dao.Loan, schedules []*dao.LoanSchedule, payment *dao.Payment) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(loan).Error; err != nil {
			return err
		}
		for _, s := range schedules {
			if err := tx.Save(s).Error; err != nil {
				return err
			}
		}
		if err := tx.Create(payment).Error; err != nil {
			return err
		}
		return nil
	})
}
