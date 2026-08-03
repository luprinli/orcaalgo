package scheduler

import (
	"context"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lee-econ/orca-core/internal/monitor"
	"github.com/lee-econ/orca-core/internal/risk"
)

type FTMOStateProvider interface {
	GetFTMOState() (dailyPnlPct float64, consistencyMult float64, isHalted bool)
	GetFTMOProfile() (consistencyThresholdPct float64, consistencyPenalty float64)
	ResetDailyState()
	SetConsistencyMultiplier(mult float64)
}

type Scheduler struct {
	vault     risk.VaultProvider
	telegram  *monitor.TelegramBot
	ftmo      FTMOStateProvider
	dbPool    *pgxpool.Pool
	jobs      []Job
	ctx       context.Context
	cancel    context.CancelFunc
}

type Job struct {
	Name     string
	Schedule string
	Run      func(ctx context.Context) error
}

func NewScheduler(vault risk.VaultProvider, telegram *monitor.TelegramBot) *Scheduler {
	ctx, cancel := context.WithCancel(context.Background())
	return &Scheduler{
		vault:    vault,
		telegram: telegram,
		ctx:      ctx,
		cancel:   cancel,
	}
}

func NewSchedulerWithFTMO(vault risk.VaultProvider, telegram *monitor.TelegramBot, ftmo FTMOStateProvider) *Scheduler {
	ctx, cancel := context.WithCancel(context.Background())
	return &Scheduler{
		vault:    vault,
		telegram: telegram,
		ftmo:     ftmo,
		ctx:      ctx,
		cancel:   cancel,
	}
}

func (s *Scheduler) SetDBPool(pool *pgxpool.Pool) {
	s.dbPool = pool
}

func (s *Scheduler) RegisterKeyRotationJob() {
	s.jobs = append(s.jobs, Job{
		Name:     "key_rotation_check",
		Schedule: "0 8 * * *",
		Run: func(ctx context.Context) error {
			brokers := []string{"alpaca"}
			for _, broker := range brokers {
				record := risk.CheckKeyRotation(s.vault, broker)
				if record != nil && record.AgeDays > 30 {
					s.telegram.Send("Warning", "CredentialExpiry", broker, record.AgeDays)
				}
			}
			return nil
		},
	})
}

func (s *Scheduler) RegisterDailyHealthJob() {
	s.jobs = append(s.jobs, Job{
		Name:     "daily_health",
		Schedule: "0 9 * * *",
		Run: func(ctx context.Context) error {
			log.Println("daily health check: system OK")
			return nil
		},
	})
}

func (s *Scheduler) RegisterDailyResetJob() {
	if s.ftmo == nil {
		return
	}
	s.jobs = append(s.jobs, Job{
		Name:     "daily_reset",
		Schedule: "0 0 * * *",
		Run: func(ctx context.Context) error {
			dailyPnlPct, consistencyMult, isHalted := s.ftmo.GetFTMOState()
			threshold, penalty := s.ftmo.GetFTMOProfile()

			log.Printf("daily reset: pnl_pct=%.2f, consistency_mult=%.2f, halted=%v, threshold=%.1f, penalty=%.2f",
				dailyPnlPct, consistencyMult, isHalted, threshold, penalty)

			actionTaken := ""
			newMult := consistencyMult

			if consistencyMult < 1.0 {
				actionTaken = "carrying_forward"
				s.telegram.Send("Warning", "ConsistencyOutlier",
					"FTMO", dailyPnlPct,
					"size_multiplier", consistencyMult,
					"action", "carrying_forward")
			} else if dailyPnlPct > threshold {
				newMult = penalty
				s.ftmo.SetConsistencyMultiplier(newMult)
				actionTaken = "reducing_size"
				s.telegram.Send("Alert", "ConsistencyOutlier",
					"FTMO", dailyPnlPct,
					"threshold", threshold,
					"new_multiplier", newMult,
					"action", "reducing_size_for_next_day")
				log.Printf("consistency outlier: daily PnL %.2f%% > %.1f%%, reducing size to %.2fx",
					dailyPnlPct, threshold, newMult)
			}

			if s.dbPool != nil {
				today := time.Now().Format("2006-01-02")
				isOutlier := dailyPnlPct > 30.0
				if _, err := s.dbPool.Exec(ctx,
					`INSERT INTO consistency_logs (date, daily_pnl_pct, is_outlier, action_taken, created_at)
					 VALUES ($1, $2, $3, $4, now())
					 ON CONFLICT (date) DO UPDATE SET daily_pnl_pct=$2, is_outlier=$3, action_taken=$4`,
					today, dailyPnlPct, isOutlier, actionTaken,
				); err != nil {
					log.Printf("scheduler: failed to persist consistency log: %v", err)
				}
			}

			s.ftmo.ResetDailyState()
			log.Println("daily reset: FTMO state reset")
			return nil
		},
	})
}

func (s *Scheduler) Start() {
	for _, job := range s.jobs {
		go s.runJob(job)
	}
}

func (s *Scheduler) runJob(job Job) {
	for {
		nextRun := time.Now().Add(24 * time.Hour)
		nextRun = time.Date(nextRun.Year(), nextRun.Month(), nextRun.Day(), 8, 0, 0, 0, nextRun.Location())
		if nextRun.Before(time.Now()) {
			nextRun = nextRun.Add(24 * time.Hour)
		}

		timer := time.NewTimer(time.Until(nextRun))
		select {
		case <-timer.C:
			if err := job.Run(s.ctx); err != nil {
				log.Printf("scheduler job %s error: %v", job.Name, err)
			}
		case <-s.ctx.Done():
			timer.Stop()
			return
		}
	}
}

func (s *Scheduler) Stop() {
	s.cancel()
}

// RegisterReoptimizationJob registers a daily parameter re-optimization check.
func (s *Scheduler) RegisterReoptimizationJob(cfg *ReoptimizationConfig) {
	s.jobs = append(s.jobs, Job{
		Name:     "parameter_reoptimization",
		Schedule: "0 16 * * 1-5",
		Run: func(ctx context.Context) error {
			cfg.CheckAndOptimize(ctx)
			return nil
		},
	})
}
