package service

import (
	"couple-mini/backend/internal/model"
	"couple-mini/backend/internal/pkg/adminview"
	applog "couple-mini/backend/internal/pkg/logger"
)

func (s *Service) AdminDashboard() (model.AdminDashboard, error) {
	overview, err := s.repo.AdminOverview()
	if err != nil {
		return model.AdminDashboard{}, err
	}
	recentUsers, err := s.repo.AdminRecentUsers(8)
	if err != nil {
		return model.AdminDashboard{}, err
	}
	recentCouples, err := s.repo.AdminRecentCouples(8)
	if err != nil {
		return model.AdminDashboard{}, err
	}
	errorLogs, err := applog.ReadRecentErrors(20)
	if err != nil {
		errorLogs = []model.AdminErrorLog{}
	}
	return model.AdminDashboard{
		Overview:      overview,
		Runtime:       adminview.RuntimeSummary(),
		Metrics:       adminview.SnapshotHistory(),
		RecentUsers:   recentUsers,
		RecentCouples: recentCouples,
		ErrorLogs:     errorLogs,
	}, nil
}

func (s *Service) AdminErrors(limit int) ([]model.AdminErrorLog, error) {
	return applog.ReadRecentErrors(limit)
}

func (s *Service) AdminCouples(limit int) ([]model.AdminCoupleSummary, error) {
	return s.repo.AdminRecentCouples(limit)
}

func (s *Service) AdminUnpairCouple(coupleID string) (model.UnpairResult, error) {
	return s.repo.AdminUnpairCouple(coupleID)
}
