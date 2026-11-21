package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/loopfz/gadgeto/tonic"
	"github.com/loopfz/gadgeto/tonic/utils/jujerr"
)

type DomainStats struct {
	DomainCount      int   `json:"domain_count"`
	TotalBandwidthMB int64 `json:"total_bandwidth"`
}

type DomainStatsCache struct {
	DomainStats
	LastUpdateTime time.Time
	Valid          bool
}

const query = `query ($since: Time!, $until: Time!) {
  viewer {
    zones {
      httpRequests1dGroups(limit: 100, filter: { date_geq: $since, date_lt: $until }) {
        sum { bytes }
      }
    }
  }
}`

var client = &http.Client{Timeout: 10 * time.Second}
var domainStatsCache = &DomainStatsCache{}

func updateDomainStats() (*DomainStats, error) {
	until := time.Now().UTC().AddDate(0, 0, -1)
	since := until.AddDate(0, 0, -30)

	payload, _ := json.Marshal(map[string]interface{}{
		"query":     query,
		"variables": map[string]string{"since": since.Format("2006-01-02"), "until": until.Format("2006-01-02")},
	})
	req, err := http.NewRequest(http.MethodPost, "https://api.cloudflare.com/client/v4/graphql", bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+os.Getenv("CF_API_TOKEN"))
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var res struct {
		Data struct {
			Viewer struct {
				Zones []struct {
					Groups []struct {
						Sum struct {
							Bytes int64 `json:"bytes"`
						} `json:"sum"`
					} `json:"httpRequests1dGroups"`
				} `json:"zones"`
			} `json:"viewer"`
		} `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, err
	}
	if len(res.Errors) > 0 {
		return nil, fmt.Errorf("%s", res.Errors[0].Message)
	}

	var total int64
	for _, z := range res.Data.Viewer.Zones {
		for _, g := range z.Groups {
			total += g.Sum.Bytes
		}
	}
	return &DomainStats{DomainCount: len(res.Data.Viewer.Zones), TotalBandwidthMB: total / 1048576}, nil
}

func getDomainStats(_ *gin.Context) (*DomainStats, error) {
	if domainStatsCache.LastUpdateTime.Before(time.Now().Add(-time.Hour)) {
		domainStatsCache.Valid = false
	}

	if !domainStatsCache.Valid {
		stats, err := updateDomainStats()
		if err != nil {
			log.Println(err.Error())
			return nil, errors.New("internal server error")
		}

		domainStatsCache.DomainCount = stats.DomainCount
		domainStatsCache.TotalBandwidthMB = stats.TotalBandwidthMB
		domainStatsCache.LastUpdateTime = time.Now()
		domainStatsCache.Valid = true
	}

	return &DomainStats{
		DomainCount:      domainStatsCache.DomainCount,
		TotalBandwidthMB: domainStatsCache.TotalBandwidthMB,
	}, nil
}

func main() {
	_ = godotenv.Load()

	var router = gin.Default()
	tonic.SetErrorHook(jujerr.ErrHook)

	router.GET(
		"/api/domain/stats",
		tonic.Handler(getDomainStats, 200),
	)

	if err := router.Run("0.0.0.0:8080"); err != nil {
		log.Fatalln(err.Error())
	}
}
