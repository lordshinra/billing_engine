package handler

import (
	"context"

	pb "billing_service/pb"
	"billing_service/service"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type handler struct {
	pb.UnimplementedBillingServiceServer
	svc service.BillingService
}

func NewHandler(svc service.BillingService) pb.BillingServiceServer {
	return &handler{svc: svc}
}

func (h *handler) CreateLoan(ctx context.Context, req *pb.CreateLoanRequest) (*pb.CreateLoanResponse, error) {
	err := h.svc.CreateLoan(ctx, req.GetLoanId(), req.GetCustomerId(), req.GetPrincipalAmount(), int(req.GetTenureWeeks()), req.GetInterestRate())
	if err != nil {
		if err == service.ErrInvalidLoanProduct {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &pb.CreateLoanResponse{Success: true}, nil
}

func (h *handler) GetOutstanding(ctx context.Context, req *pb.GetOutstandingRequest) (*pb.GetOutstandingResponse, error) {
	amount, err := h.svc.GetOutstanding(ctx, req.GetLoanId())
	if err != nil {
		if err == service.ErrLoanNotFound {
			return nil, status.Error(codes.NotFound, err.Error())
		}
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &pb.GetOutstandingResponse{OutstandingAmount: amount}, nil
}

func (h *handler) IsDelinquent(ctx context.Context, req *pb.IsDelinquentRequest) (*pb.IsDelinquentResponse, error) {
	delinquent, err := h.svc.IsDelinquent(ctx, req.GetLoanId())
	if err != nil {
		if err == service.ErrLoanNotFound {
			return nil, status.Error(codes.NotFound, err.Error())
		}
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &pb.IsDelinquentResponse{Delinquent: delinquent}, nil
}

func (h *handler) MakePayment(ctx context.Context, req *pb.MakePaymentRequest) (*pb.MakePaymentResponse, error) {
	err := h.svc.MakePayment(ctx, req.GetLoanId(), req.GetAmount())
	if err != nil {
		if err == service.ErrLoanNotFound {
			return nil, status.Error(codes.NotFound, err.Error())
		}
		if err == service.ErrLoanAlreadyClosed || err == service.ErrInvalidPaymentAmount || err == service.ErrPaymentExceedsOutstanding {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &pb.MakePaymentResponse{Success: true}, nil
}
