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

// logsCmd represents the logs command
var logsCmd = &cobra.Command{
	Use:   "logs",
	Short: "logs inserts logs into the logs table.",
	Run: func(cmd *cobra.Command, args []string) {
		ctx, cancel := cmdutils.NewInterruptContext()
		defer cancel()

		logs(ctx)
	},
}

var logsCount int
var logsTenants int
var logsResources int
var timeStart string
var timeEnd string

func init() {
	rootCmd.AddCommand(logsCmd)

	logsCmd.PersistentFlags().IntVarP(
		&logsTenants,
		"tenants",
		"t",
		10,
		"The number of tenants.",
	)

	logsCmd.PersistentFlags().IntVarP(
		&logsResources,
		"resources",
		"r",
		1,
		"The number of distinct resources to create per tenant.",
	)

	logsCmd.PersistentFlags().IntVarP(
		&logsCount,
		"count",
		"c",
		10000,
		"The number of logs to create per (tenant, resource) tuple.",
	)

	logsCmd.PersistentFlags().StringVarP(
		&timeStart,
		"start",
		"1",
		"2024-01-01T00:00:00Z",
		"The start time for the logs.",
	)

	logsCmd.PersistentFlags().StringVarP(
		&timeEnd,
		"end",
		"2",
		"2025-01-01T00:00:00Z",
		"The end time for the logs.",
	)
}

func logs(ctx context.Context) {
	startLogs, err := time.Parse(time.RFC3339, timeStart)

	if err != nil {
		panic(err)
	}

	endLogs, err := time.Parse(time.RFC3339, timeEnd)

	if err != nil {
		panic(err)
	}

	unixSpan := endLogs.UnixNano() - startLogs.UnixNano()

	// initialize buffer
	opts := buffered.BufferOpts[dbsqlc.BulkInsertLogsParams, dbsqlc.BulkInsertLogsParams]{
		Name:               "logs_writer",
		MaxCapacity:        1000,
		FlushPeriod:        100 * time.Millisecond,
		MaxDataSizeInQueue: 100,
		FlushFunc:          logsFlush,
		SizeFunc:           func(item dbsqlc.BulkInsertLogsParams) int { return len(item.Log.String) },
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

	totalTasks := logsTenants * logsResources * logsCount
	bar := progressbar.NewOptions(totalTasks, progressbar.OptionSetDescription("Processing logs"), progressbar.OptionShowCount())

	for i := 0; i < logsTenants; i++ {
		tenantId := uuid.New().String()

		for j := 0; j < logsResources; j++ {
			resourceId := uuid.New().String()

			for k := 0; k < logsCount; k++ {
				wg.Add(1)

				// choose equal segments in the time span
				created := startLogs.Add(time.Duration(
					int64(k) * (unixSpan / int64(logsCount)),
				))

				doneCh, err := b.BuffItem(dbsqlc.BulkInsertLogsParams{
					CreatedAt:  pgtype.Timestamptz{Time: created, Valid: true},
					TenantID:   uuidFromStr(tenantId),
					ResourceID: uuidFromStr(resourceId),
					Log: pgtype.Text{
						String: fmt.Sprintf("message for tenant %s, resource %s, event %d", tenantId, resourceId, k),
						Valid:  true,
					},
				})

				if err != nil {
					panic(err)
				}

				go func() {
					defer wg.Done()

					<-doneCh
					bar.Add(1) // Increment progress bar for each processed event
				}()
			}
		}
	}

	wg.Wait()

	if err := cleanupBuffer(); err != nil {
		panic(err)
	}

	fmt.Println("\nlogs command completed")
}

func logsFlush(ctx context.Context, items []dbsqlc.BulkInsertLogsParams) ([]dbsqlc.BulkInsertLogsParams, error) {
	_, err := queries.BulkInsertLogs(ctx, pool, items)

	return items, err
}
