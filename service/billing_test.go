package service

import (
	"context"
	"testing"
	"time"

	"billing_service/dao"
	"billing_service/repository"
)

type mockRepo struct {
	getLoanFn                func(ctx context.Context, loanID string) (*dao.Loan, error)
	getUnpaidSchedulesFn     func(ctx context.Context, loanID string) ([]*dao.LoanSchedule, error)
	getLastTwoDueSchedulesFn func(ctx context.Context, loanID string) ([]*dao.LoanSchedule, error)
	updatePaymentTxFn        func(ctx context.Context, loan *dao.Loan, schedules []*dao.LoanSchedule, payment *dao.Payment) error
	createLoanTxFn           func(ctx context.Context, loan *dao.Loan, schedules []*dao.LoanSchedule) error
}

func (m *mockRepo) CreateLoanTx(ctx context.Context, loan *dao.Loan, schedules []*dao.LoanSchedule) error {
	if m.createLoanTxFn != nil {
		return m.createLoanTxFn(ctx, loan, schedules)
	}
	return nil
}

func (m *mockRepo) GetLoan(ctx context.Context, loanID string) (*dao.Loan, error) {
	if m.getLoanFn != nil {
		return m.getLoanFn(ctx, loanID)
	}
	return nil, repository.ErrNotFound
}

func (m *mockRepo) GetUnpaidSchedules(ctx context.Context, loanID string) ([]*dao.LoanSchedule, error) {
	if m.getUnpaidSchedulesFn != nil {
		return m.getUnpaidSchedulesFn(ctx, loanID)
	}
	return nil, nil
}

func (m *mockRepo) GetLastTwoDueSchedules(ctx context.Context, loanID string) ([]*dao.LoanSchedule, error) {
	if m.getLastTwoDueSchedulesFn != nil {
		return m.getLastTwoDueSchedulesFn(ctx, loanID)
	}
	return nil, nil
}

func (m *mockRepo) UpdatePaymentTx(ctx context.Context, loan *dao.Loan, schedules []*dao.LoanSchedule, payment *dao.Payment) error {
	if m.updatePaymentTxFn != nil {
		return m.updatePaymentTxFn(ctx, loan, schedules, payment)
	}
	return nil
}

func TestMakePayment(t *testing.T) {
	tests := []struct {
		name          string
		loanID        string
		amount        int64
		setupMock     func(*mockRepo)
		expectedError error
	}{
		{
			name:   "Successful Regular Payment",
			loanID: "L1",
			amount: 5500,
			setupMock: func(m *mockRepo) {
				m.getLoanFn = func(ctx context.Context, loanID string) (*dao.Loan, error) {
					return &dao.Loan{Status: "ACTIVE", OutstandingAmount: 55000, WeeklyInstallment: 5500}, nil
				}
				m.getUnpaidSchedulesFn = func(ctx context.Context, loanID string) ([]*dao.LoanSchedule, error) {
					// 1 due in the future
					return []*dao.LoanSchedule{
						{DueDate: time.Now().Add(24 * time.Hour), Paid: false},
					}, nil
				}
				m.updatePaymentTxFn = func(ctx context.Context, loan *dao.Loan, schedules []*dao.LoanSchedule, payment *dao.Payment) error {
					return nil
				}
			},
			expectedError: nil,
		},
		{
			name:   "Failed Payment Not Multiple of Installment",
			loanID: "L1",
			amount: 6000,
			setupMock: func(m *mockRepo) {
				m.getLoanFn = func(ctx context.Context, loanID string) (*dao.Loan, error) {
					return &dao.Loan{Status: "ACTIVE", OutstandingAmount: 55000, WeeklyInstallment: 5500}, nil
				}
			},
			expectedError: ErrInvalidPaymentAmount,
		},
		{
			name:   "Successful Payment 2 Missed Installments",
			loanID: "L2",
			amount: 11000, // 2 * 5500
			setupMock: func(m *mockRepo) {
				m.getLoanFn = func(ctx context.Context, loanID string) (*dao.Loan, error) {
					return &dao.Loan{Status: "ACTIVE", OutstandingAmount: 55000, WeeklyInstallment: 5500}, nil
				}
				m.getUnpaidSchedulesFn = func(ctx context.Context, loanID string) ([]*dao.LoanSchedule, error) {
					// 2 due in the past (missed)
					return []*dao.LoanSchedule{
						{DueDate: time.Now().Add(-48 * time.Hour), Paid: false},
						{DueDate: time.Now().Add(-24 * time.Hour), Paid: false},
					}, nil
				}
				m.updatePaymentTxFn = func(ctx context.Context, loan *dao.Loan, schedules []*dao.LoanSchedule, payment *dao.Payment) error {
					return nil
				}
			},
			expectedError: nil,
		},
		{
			name:   "Failed Payment Partial Missed Installments",
			loanID: "L2",
			amount: 5500, // Wants to pay 1, but missed 2
			setupMock: func(m *mockRepo) {
				m.getLoanFn = func(ctx context.Context, loanID string) (*dao.Loan, error) {
					return &dao.Loan{Status: "ACTIVE", OutstandingAmount: 55000, WeeklyInstallment: 5500}, nil
				}
				m.getUnpaidSchedulesFn = func(ctx context.Context, loanID string) ([]*dao.LoanSchedule, error) {
					// 2 due in the past (missed)
					return []*dao.LoanSchedule{
						{DueDate: time.Now().Add(-48 * time.Hour), Paid: false},
						{DueDate: time.Now().Add(-24 * time.Hour), Paid: false},
					}, nil
				}
			},
			expectedError: ErrInvalidPaymentAmount,
		},
		{
			name:   "Failed Payment Exceeds Outstanding",
			loanID: "L1",
			amount: 11000,
			setupMock: func(m *mockRepo) {
				m.getLoanFn = func(ctx context.Context, loanID string) (*dao.Loan, error) {
					return &dao.Loan{Status: "ACTIVE", OutstandingAmount: 5500, WeeklyInstallment: 5500}, nil
				}
				m.getUnpaidSchedulesFn = func(ctx context.Context, loanID string) ([]*dao.LoanSchedule, error) {
					// Only 1 left, so paying 2 exceeds
					return []*dao.LoanSchedule{
						{DueDate: time.Now().Add(24 * time.Hour), Paid: false},
					}, nil
				}
			},
			expectedError: ErrPaymentExceedsOutstanding,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := &mockRepo{}
			if tc.setupMock != nil {
				tc.setupMock(repo)
			}
			svc := NewBillingService(repo)

			err := svc.MakePayment(context.Background(), tc.loanID, tc.amount)
			if err != tc.expectedError {
				t.Errorf("expected error %v, got %v", tc.expectedError, err)
			}
		})
	}
}

func TestIsDelinquent(t *testing.T) {
	tests := []struct {
		name          string
		loanID        string
		setupMock     func(*mockRepo)
		expectedVal   bool
		expectedError error
	}{
		{
			name:   "Not Delinquent (0 missed)",
			loanID: "L1",
			setupMock: func(m *mockRepo) {
				m.getLoanFn = func(ctx context.Context, loanID string) (*dao.Loan, error) {
					return &dao.Loan{Status: "ACTIVE"}, nil
				}
				m.getLastTwoDueSchedulesFn = func(ctx context.Context, loanID string) ([]*dao.LoanSchedule, error) {
					return []*dao.LoanSchedule{
						{Paid: true},
						{Paid: true},
					}, nil
				}
			},
			expectedVal:   false,
			expectedError: nil,
		},
		{
			name:   "Not Delinquent (1 missed)",
			loanID: "L1",
			setupMock: func(m *mockRepo) {
				m.getLoanFn = func(ctx context.Context, loanID string) (*dao.Loan, error) {
					return &dao.Loan{Status: "ACTIVE"}, nil
				}
				m.getLastTwoDueSchedulesFn = func(ctx context.Context, loanID string) ([]*dao.LoanSchedule, error) {
					return []*dao.LoanSchedule{
						{Paid: false},
						{Paid: true},
					}, nil
				}
			},
			expectedVal:   false,
			expectedError: nil,
		},
		{
			name:   "Delinquent (2 missed)",
			loanID: "L1",
			setupMock: func(m *mockRepo) {
				m.getLoanFn = func(ctx context.Context, loanID string) (*dao.Loan, error) {
					return &dao.Loan{Status: "ACTIVE"}, nil
				}
				m.getLastTwoDueSchedulesFn = func(ctx context.Context, loanID string) ([]*dao.LoanSchedule, error) {
					return []*dao.LoanSchedule{
						{Paid: false},
						{Paid: false},
					}, nil
				}
			},
			expectedVal:   true,
			expectedError: nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := &mockRepo{}
			if tc.setupMock != nil {
				tc.setupMock(repo)
			}
			svc := NewBillingService(repo)

			val, err := svc.IsDelinquent(context.Background(), tc.loanID)
			if err != tc.expectedError {
				t.Errorf("expected error %v, got %v", tc.expectedError, err)
			}
			if val != tc.expectedVal {
				t.Errorf("expected value %v, got %v", tc.expectedVal, val)
			}
		})
	}
}

func TestCreateLoan(t *testing.T) {
	tests := []struct {
		name            string
		loanID          string
		customerID      string
		principalAmount int64
		tenureWeeks     int
		interestRate    float64
		setupMock       func(*mockRepo)
		expectedError   error
	}{
		{
			name:            "Successful Create Loan",
			loanID:          "L3",
			customerID:      "C1",
			principalAmount: 5000000,
			tenureWeeks:     50,
			interestRate:    10.0,
			setupMock: func(m *mockRepo) {
				m.createLoanTxFn = func(ctx context.Context, loan *dao.Loan, schedules []*dao.LoanSchedule) error {
					return nil
				}
			},
			expectedError: nil,
		},
		{
			name:            "Failed Create Loan Invalid Input",
			loanID:          "",
			customerID:      "C1",
			principalAmount: 5000000,
			tenureWeeks:     50,
			interestRate:    10.0,
			setupMock:       nil,
			expectedError:   ErrInvalidLoanProduct,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := &mockRepo{}
			if tc.setupMock != nil {
				tc.setupMock(repo)
			}
			svc := NewBillingService(repo)

			err := svc.CreateLoan(context.Background(), tc.loanID, tc.customerID, tc.principalAmount, tc.tenureWeeks, tc.interestRate)
			if err != tc.expectedError {
				t.Errorf("expected error %v, got %v", tc.expectedError, err)
			}
		})
	}
}

func TestGetOutstanding(t *testing.T) {
	tests := []struct {
		name          string
		loanID        string
		setupMock     func(*mockRepo)
		expectedVal   int64
		expectedError error
	}{
		{
			name:   "Successful Get Outstanding",
			loanID: "L1",
			setupMock: func(m *mockRepo) {
				m.getLoanFn = func(ctx context.Context, loanID string) (*dao.Loan, error) {
					return &dao.Loan{OutstandingAmount: 55000}, nil
				}
			},
			expectedVal:   55000,
			expectedError: nil,
		},
		{
			name:   "Failed Get Outstanding Loan Not Found",
			loanID: "L2",
			setupMock: func(m *mockRepo) {
				m.getLoanFn = func(ctx context.Context, loanID string) (*dao.Loan, error) {
					return nil, repository.ErrNotFound
				}
			},
			expectedVal:   0,
			expectedError: ErrLoanNotFound,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := &mockRepo{}
			if tc.setupMock != nil {
				tc.setupMock(repo)
			}
			svc := NewBillingService(repo)

			val, err := svc.GetOutstanding(context.Background(), tc.loanID)
			if err != tc.expectedError {
				t.Errorf("expected error %v, got %v", tc.expectedError, err)
			}
			if val != tc.expectedVal {
				t.Errorf("expected value %v, got %v", tc.expectedVal, val)
			}
		})
	}
}
