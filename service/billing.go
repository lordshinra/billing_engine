package service

import (
	"context"
	"errors"
	"time"

	"billing_service/dao"
	"billing_service/repository"

	"github.com/bwmarrin/snowflake"
)

var (
	ErrInvalidLoanProduct        = errors.New("INVALID_LOAN_PRODUCT")
	ErrLoanNotFound              = errors.New("LOAN_NOT_FOUND")
	ErrInvalidPaymentAmount      = errors.New("INVALID_PAYMENT_AMOUNT")
	ErrPaymentExceedsOutstanding = errors.New("PAYMENT_EXCEEDS_OUTSTANDING")
	ErrLoanAlreadyClosed         = errors.New("LOAN_ALREADY_CLOSED")
)

type BillingService interface {
	CreateLoan(ctx context.Context, loanID, customerID string, principalAmount int64, tenureWeeks int, interestRate float64) error
	GetOutstanding(ctx context.Context, loanID string) (int64, error)
	IsDelinquent(ctx context.Context, loanID string) (bool, error)
	MakePayment(ctx context.Context, loanID string, amount int64) error
}

type billingService struct {
	repo repository.Repository
}

func NewBillingService(repo repository.Repository) BillingService {
	return &billingService{repo: repo}
}

func (s *billingService) CreateLoan(ctx context.Context, loanID, customerID string, principalAmount int64, tenureWeeks int, interestRate float64) error {
	if loanID == "" || customerID == "" || principalAmount <= 0 {
		return ErrInvalidLoanProduct
	}

	if tenureWeeks < 20 {
		tenureWeeks = 20
	}
	if interestRate < 5 {
		interestRate = 5
	}

	interest := int64(float64(principalAmount) * (interestRate / 100))
	totalAmount := principalAmount + interest
	weeklyInstallment := totalAmount / int64(tenureWeeks)

	now := time.Now()

	node, _ := snowflake.NewNode(1)
	id := node.Generate()

	loan := &dao.Loan{
		ID:                id.Int64(),
		LoanID:            loanID,
		CustomerID:        customerID,
		PrincipalAmount:   principalAmount,
		InterestRate:      interestRate,
		TotalAmount:       totalAmount,
		OutstandingAmount: totalAmount,
		WeeklyInstallment: weeklyInstallment,
		TenureWeeks:       tenureWeeks,
		Status:            "ACTIVE",
		CreatedAt:         now,
		UpdatedAt:         now,
	}

	schedules := make([]*dao.LoanSchedule, 0, tenureWeeks)
	for week := 1; week <= tenureWeeks; week++ {
		schId := node.Generate()
		dueDate := now.Add(time.Duration(week*7*24) * time.Hour)
		schedules = append(schedules, &dao.LoanSchedule{
			ID:         schId.Int64(),
			LoanID:     loanID,
			WeekNumber: week,
			DueAmount:  weeklyInstallment,
			DueDate:    dueDate,
			Paid:       false,
			CreatedAt:  now,
		})
	}

	return s.repo.CreateLoanTx(ctx, loan, schedules)
}

func (s *billingService) GetOutstanding(ctx context.Context, loanID string) (int64, error) {
	loan, err := s.repo.GetLoan(ctx, loanID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return 0, ErrLoanNotFound
		}
		return 0, err
	}
	return loan.OutstandingAmount, nil
}

func (s *billingService) IsDelinquent(ctx context.Context, loanID string) (bool, error) {
	_, err := s.repo.GetLoan(ctx, loanID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return false, ErrLoanNotFound
		}
		return false, err
	}

	// Check if last 2 schedule payment is missed (unpaid and past due date)
	lastDueSchedules, err := s.repo.GetLastTwoDueSchedules(ctx, loanID)
	if err != nil {
		return false, err
	}

	if len(lastDueSchedules) < 2 {
		// Cannot have missed 2 due payments if less than 2 payments are due
		return false, nil
	}

	missed := 0
	for _, schedule := range lastDueSchedules {
		if !schedule.Paid {
			missed++
		}
	}

	return missed >= 2, nil
}

func (s *billingService) MakePayment(ctx context.Context, loanID string, amount int64) error {
	loan, err := s.repo.GetLoan(ctx, loanID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return ErrLoanNotFound
		}
		return err
	}

	if loan.Status == "CLOSED" || loan.OutstandingAmount == 0 {
		return ErrLoanAlreadyClosed
	}

	if amount%loan.WeeklyInstallment != 0 {
		return ErrInvalidPaymentAmount
	}

	if amount > loan.OutstandingAmount {
		return ErrPaymentExceedsOutstanding
	}

	unpaidSchedules, err := s.repo.GetUnpaidSchedules(ctx, loanID)
	if err != nil {
		return err
	}

	// Check how many payments are already due and ensure payment amount matches the number of due payments
	now := time.Now()
	missedCount := 0
	for _, sch := range unpaidSchedules {
		if !sch.DueDate.After(now) {
			missedCount++
		}
	}

	if missedCount > 0 && amount != int64(missedCount)*loan.WeeklyInstallment {
		return ErrInvalidPaymentAmount
	}

	paymentCount := int(amount / loan.WeeklyInstallment)
	if paymentCount > len(unpaidSchedules) {
		return ErrPaymentExceedsOutstanding
	}

	var schedulesToUpdate []*dao.LoanSchedule

	for i := 0; i < paymentCount; i++ {
		unpaidSchedules[i].Paid = true
		unpaidSchedules[i].PaidAt = &now
		schedulesToUpdate = append(schedulesToUpdate, unpaidSchedules[i])
	}

	loan.OutstandingAmount -= amount
	loan.UpdatedAt = now
	if loan.OutstandingAmount == 0 {
		loan.Status = "CLOSED"
	}

	node, _ := snowflake.NewNode(1)
	id := node.Generate()

	payment := &dao.Payment{
		ID:        id.Int64(),
		LoanID:    loanID,
		Amount:    amount,
		CreatedAt: now,
	}

	return s.repo.UpdatePaymentTx(ctx, loan, schedulesToUpdate, payment)
}
