package main

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/spf13/cobra"

	"github.com/abelanger5/postgres-events-table/internal/cmdutils"
	"github.com/abelanger5/postgres-events-table/internal/dbsqlc"
	"github.com/hatchet-dev/buffered"
	"github.com/schollz/progressbar/v3"
)

// metricsCmd represents the simple command
var metricsCmd = &cobra.Command{
	Use:   "metrics",
	Short: "metrics inserts events for metrics into the metric_events table.",
	Run: func(cmd *cobra.Command, args []string) {
		ctx, cancel := cmdutils.NewInterruptContext()
		defer cancel()

		metrics(ctx)
	},
}

var metricsCount int
var metricsTenants int
var metricsResources int

func init() {
	rootCmd.AddCommand(metricsCmd)

	metricsCmd.PersistentFlags().IntVarP(
		&metricsTenants,
		"tenants",
		"t",
		10,
		"The number of tenants.",
	)

	metricsCmd.PersistentFlags().IntVarP(
		&metricsResources,
		"resources",
		"r",
		10,
		"The number of distinct resources to create per tenant.",
	)

	metricsCmd.PersistentFlags().IntVarP(
		&metricsCount,
		"count",
		"c",
		1000,
		"The number of events to create per (tenant, resource) tuple.",
	)
}

func metrics(ctx context.Context) {
	start := time.Now()
	end := time.Now().Add(24 * time.Hour)

	unixSpan := end.UnixNano() - start.UnixNano()

	// initialize buffer
	opts := buffered.BufferOpts[dbsqlc.InsertMetricsParams, *dbsqlc.MetricEvent]{
		Name:               "metric_writer",
		MaxCapacity:        1000,
		FlushPeriod:        100 * time.Millisecond,
		MaxDataSizeInQueue: 100,
		FlushFunc:          metricFlush,
		SizeFunc:           func(item dbsqlc.InsertMetricsParams) int { return len(item.Data) },
	}

	b := buffered.NewBuffer(opts)
	wg := sync.WaitGroup{}

	cleanupBuffer, err := b.Start()

	if err != nil {
		panic(err)
	}

	go func() {
		<-ctx.Done()
		cleanupBuffer()
	}()

	totalTasks := metricsTenants * metricsResources * metricsCount
	bar := progressbar.NewOptions(totalTasks, progressbar.OptionSetDescription("Processing events"), progressbar.OptionShowCount())

	for i := 0; i < metricsTenants; i++ {
		tenantId := uuid.New().String()

		for j := 0; j < metricsResources; j++ {
			resourceId := uuid.New().String()

			for k := 0; k < metricsCount; k++ {
				wg.Add(1)

				eventType := dbsqlc.MetricEventTypeSUCCEEDED

				if k%2 == 0 {
					eventType = dbsqlc.MetricEventTypeFAILED
				}

				created := start.Add(time.Duration(
					int64(k) * (unixSpan / int64(metricsCount)),
				))

				doneCh, err := b.BuffItem(dbsqlc.InsertMetricsParams{
					CreatedAt:  pgtype.Timestamptz{Time: created, Valid: true},
					TenantID:   uuidFromStr(tenantId),
					ResourceID: uuidFromStr(resourceId),
					EventType:  eventType,
					Data:       []byte(fmt.Sprintf("{\"message\": \"message for tenant %s, resource %s, event %d\"}", tenantId, resourceId, k)),
				})

				if err != nil {
					panic(err)
				}

				go func() {
					defer wg.Done()

					resp := <-doneCh

					if resp.Err != nil {
						panic(resp.Err)
					}

					bar.Add(1) // Increment progress bar for each processed event
				}()
			}
		}
	}

	wg.Wait()

	if err := cleanupBuffer(); err != nil {
		panic(err)
	}

	fmt.Println("\nmetrics command completed")
}

func metricFlush(ctx context.Context, items []dbsqlc.InsertMetricsParams) ([]*dbsqlc.MetricEvent, error) {
	params := dbsqlc.BulkInsertMetricsParams{
		CreatedAts:  []pgtype.Timestamptz{},
		TenantIds:   []pgtype.UUID{},
		ResourceIds: []pgtype.UUID{},
		EventTypes:  []string{},
		Datas:       [][]byte{},
	}

	for _, item := range items {
		params.CreatedAts = append(params.CreatedAts, item.CreatedAt)
		params.TenantIds = append(params.TenantIds, item.TenantID)
		params.ResourceIds = append(params.ResourceIds, item.ResourceID)
		params.EventTypes = append(params.EventTypes, string(item.EventType))
		params.Datas = append(params.Datas, item.Data)
	}

	return queries.BulkInsertMetrics(ctx, pool, params)
}
