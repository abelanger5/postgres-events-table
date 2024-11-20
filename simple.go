package main

import (
	"context"
	"fmt"
	"math/rand/v2"
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

// simpleCmd represents the simple command
var simpleCmd = &cobra.Command{
	Use:   "simple",
	Short: "simple inserts events into the simple table.",
	Run: func(cmd *cobra.Command, args []string) {
		ctx, cancel := cmdutils.NewInterruptContext()
		defer cancel()

		simple(ctx)
	},
}

var simpleQueryCmd = &cobra.Command{
	Use:   "query",
	Short: "query runs random selects on the simple_events table",
	Run: func(cmd *cobra.Command, args []string) {
		ctx, cancel := cmdutils.NewInterruptContext()
		defer cancel()

		querySimple(ctx)
	},
}

var eventsCount int
var tenants int
var resources int
var querySamples int

func init() {
	rootCmd.AddCommand(simpleCmd)

	simpleCmd.PersistentFlags().IntVarP(
		&tenants,
		"tenants",
		"t",
		10,
		"The number of tenants.",
	)

	simpleCmd.PersistentFlags().IntVarP(
		&resources,
		"resources",
		"r",
		1000,
		"The number of distinct resources to create per tenant.",
	)

	simpleCmd.PersistentFlags().IntVarP(
		&eventsCount,
		"count",
		"c",
		10,
		"The number of events to create per (tenant, resource) tuple.",
	)

	simpleCmd.AddCommand(simpleQueryCmd)

	simpleQueryCmd.PersistentFlags().IntVarP(
		&querySamples,
		"samples",
		"s",
		1000,
		"The number of random selects to run.",
	)
}

func simple(ctx context.Context) {
	// initialize buffer
	opts := buffered.BufferOpts[dbsqlc.InsertSimpleParams, *dbsqlc.SimpleEvent]{
		Name:               "simple_writer",
		MaxCapacity:        1000,
		FlushPeriod:        100 * time.Millisecond,
		MaxDataSizeInQueue: 100,
		FlushFunc:          simpleFlush,
		SizeFunc:           func(item dbsqlc.InsertSimpleParams) int { return len(item.Data) },
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

	totalTasks := tenants * resources * eventsCount
	bar := progressbar.NewOptions(totalTasks, progressbar.OptionSetDescription("Processing events"), progressbar.OptionShowCount())

	for i := 0; i < tenants; i++ {
		tenantId := uuid.New().String()

		for j := 0; j < resources; j++ {
			resourceId := uuid.New().String()

			for k := 0; k < eventsCount; k++ {
				wg.Add(1)

				doneCh, err := b.BuffItem(dbsqlc.InsertSimpleParams{
					CreatedAt:  pgtype.Timestamptz{Time: time.Now(), Valid: true},
					TenantID:   uuidFromStr(tenantId),
					ResourceID: uuidFromStr(resourceId),
					Data:       []byte(fmt.Sprintf("{\"message\": \"message for tenant %s, resource %s, event %d\"}", tenantId, resourceId, k)),
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

	fmt.Println("\nsimple command completed")
}

func querySimple(ctx context.Context) {
	row := pool.QueryRow(ctx, "SELECT MAX(id) FROM simple_events")

	var maxId int
	err := row.Scan(&maxId)

	if err != nil {
		panic(err)
	}

	row = pool.QueryRow(ctx, "SELECT MIN(id) FROM simple_events")

	var minId int
	err = row.Scan(&minId)

	if err != nil {
		panic(err)
	}

	fmt.Printf("max id: %d\n", maxId)
	fmt.Printf("min id: %d\n", minId)

	// run random selects in the range of minId to maxId
	selectParams := make([]dbsqlc.ListEventsByResourceIDParams, querySamples)

	// we prepare a set of tenantId, resourceId pairs to query based on random ids in the range of minId to maxId
	for i := 0; i < querySamples; i++ {
		// choose a random id in the range of minId to maxId
		id := rand.IntN(maxId-minId) + minId

		e, err := queries.GetEventByID(ctx, pool, int64(id))

		if err != nil {
			panic(err)
		}

		selectParams[i] = dbsqlc.ListEventsByResourceIDParams{
			TenantID:   e.TenantID,
			ResourceID: e.ResourceID,
		}
	}

	start := time.Now()

	for i := 0; i < querySamples; i++ {
		_, err := queries.ListEventsByResourceID(ctx, pool, selectParams[i])

		if err != nil {
			panic(err)
		}
	}

	elapsed := time.Since(start)

	fmt.Printf("query samples: %d\n", querySamples)
	fmt.Printf("elapsed time: %s\n", elapsed)
	fmt.Printf("average time per query: %s\n", elapsed/time.Duration(querySamples))
}

func simpleFlush(ctx context.Context, items []dbsqlc.InsertSimpleParams) ([]*dbsqlc.SimpleEvent, error) {
	params := dbsqlc.BulkInsertSimpleParams{
		CreatedAts:  []pgtype.Timestamptz{},
		TenantIds:   []pgtype.UUID{},
		ResourceIds: []pgtype.UUID{},
		Datas:       [][]byte{},
	}

	for _, item := range items {
		params.CreatedAts = append(params.CreatedAts, item.CreatedAt)
		params.TenantIds = append(params.TenantIds, item.TenantID)
		params.ResourceIds = append(params.ResourceIds, item.ResourceID)
		params.Datas = append(params.Datas, item.Data)
	}

	return queries.BulkInsertSimple(ctx, pool, params)
}

func uuidToStr(uuid pgtype.UUID) string {
	return fmt.Sprintf("%x-%x-%x-%x-%x", uuid.Bytes[0:4], uuid.Bytes[4:6], uuid.Bytes[6:8], uuid.Bytes[8:10], uuid.Bytes[10:16])
}

func uuidFromStr(uuid string) pgtype.UUID {
	var pgUUID pgtype.UUID

	if err := pgUUID.Scan(uuid); err != nil {
		panic(err)
	}

	return pgUUID
}
