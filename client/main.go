package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"time"

	pb "billing_service/pb"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func createLoan(ctx context.Context, client pb.BillingServiceClient, loanID, customerID string, amount int64, tenureWeeks int, interestRate float64) {
	req := &pb.CreateLoanRequest{
		LoanId:          loanID,
		CustomerId:      customerID,
		PrincipalAmount: amount,
		TenureWeeks:     int32(tenureWeeks),
		InterestRate:    interestRate,
	}
	fmt.Printf("Calling CreateLoan for LoanID: %s\n", req.GetLoanId())
	res, err := client.CreateLoan(ctx, req)
	if err != nil {
		log.Fatalf("could not create loan: %v", err)
	}
	fmt.Printf("CreateLoan Response - Success: %v\n", res.GetSuccess())
}

func getOutstanding(ctx context.Context, client pb.BillingServiceClient, loanID string) {
	req := &pb.GetOutstandingRequest{
		LoanId: loanID,
	}
	fmt.Printf("Calling GetOutstanding for LoanID: %s\n", req.GetLoanId())
	res, err := client.GetOutstanding(ctx, req)
	if err != nil {
		log.Fatalf("could not get outstanding: %v", err)
	}
	fmt.Printf("GetOutstanding Response - Amount: %d\n", res.GetOutstandingAmount())
}

func makePayment(ctx context.Context, client pb.BillingServiceClient, loanID string, amount int64) {
	req := &pb.MakePaymentRequest{
		LoanId: loanID,
		Amount: amount,
	}
	fmt.Printf("Calling MakePayment for LoanID: %s, Amount: %d\n", req.GetLoanId(), req.GetAmount())
	res, err := client.MakePayment(ctx, req)
	if err != nil {
		log.Fatalf("could not make payment: %v", err)
	}
	fmt.Printf("MakePayment Response - Success: %v\n", res.GetSuccess())
}

func isDelinquent(ctx context.Context, client pb.BillingServiceClient, loanID string) {
	req := &pb.IsDelinquentRequest{
		LoanId: loanID,
	}
	fmt.Printf("Calling IsDelinquent for LoanID: %s\n", req.GetLoanId())
	res, err := client.IsDelinquent(ctx, req)
	if err != nil {
		log.Fatalf("could not check delinquency: %v", err)
	}
	fmt.Printf("IsDelinquent Response - Delinquent: %v\n", res.GetDelinquent())
}

func main() {
	method := flag.String("method", "", "RPC method to call: create, outstanding, payment, delinquent")
	loanID := flag.String("loan_id", "loan-33211", "Loan ID")
	customerID := flag.String("customer_id", "cust-55321", "Customer ID")
	amount := flag.Int64("amount", 5000000, "Amount (Principal or Payment)")
	tenureWeeks := flag.Int("tenure_weeks", 50, "Tenure in weeks")
	interestRate := flag.Float64("interest_rate", 10.0, "Interest rate in percentage")

	flag.Parse()

	if *method == "" {
		log.Fatal("Please specify a method using -method flag (create, outstanding, payment, delinquent)")
	}

	conn, err := grpc.NewClient("localhost:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()

	client := pb.NewBillingServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	switch *method {
	case "create":
		createLoan(ctx, client, *loanID, *customerID, *amount, *tenureWeeks, *interestRate)
	case "outstanding":
		getOutstanding(ctx, client, *loanID)
	case "payment":
		makePayment(ctx, client, *loanID, *amount)
	case "delinquent":
		isDelinquent(ctx, client, *loanID)
	default:
		log.Fatalf("Unknown method: %s", *method)
	}
}
