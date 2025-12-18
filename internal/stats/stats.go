package stats

import (
	"time"

	"gorm.io/gorm"
)

// DailySummary is used to prepare data for the admin page.
type DailySummary struct {
	Date           string           `json:"date"`
	NewUsers       int64            `json:"new_users"`
	ActiveUsers    int64            `json:"active_users"`
	OnlineUsers    int64            `json:"online_users"`
	PlatformActive map[string]int64 `json:"platform_active"`
	RegionActive   map[string]int64 `json:"region_active"`
}

// GetLastNDaysSummary queries DB and builds per-day stats including per-platform active users.
func GetLastNDaysSummary(db *gorm.DB, days int) ([]DailySummary, error) {
	today := time.Now().UTC()
	start := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, time.UTC).AddDate(0, 0, -days+1)

	// 1) New users per day.
	type NewRow struct {
		Day time.Time
		Cnt int64
	}
	var newRows []NewRow
	if err := db.Raw(`
        SELECT DATE(first_seen) AS day, COUNT(*) AS cnt
        FROM users
        WHERE first_seen >= ?
        GROUP BY DATE(first_seen)
        ORDER BY day
    `, start).Scan(&newRows).Error; err != nil {
		return nil, err
	}
	newMap := make(map[string]int64)
	for _, r := range newRows {
		newMap[r.Day.Format("2006-01-02")] = r.Cnt
	}

	// 2) Active users per day.
	type ActiveRow struct {
		Day time.Time
		Cnt int64
	}
	var activeRows []ActiveRow
	if err := db.Raw(`
        SELECT DATE(event_time) AS day, COUNT(DISTINCT user_id) AS cnt
        FROM user_events
        WHERE event_time >= ?
        GROUP BY DATE(event_time)
        ORDER BY day
    `, start).Scan(&activeRows).Error; err != nil {
		return nil, err
	}
	activeMap := make(map[string]int64)
	for _, r := range activeRows {
		activeMap[r.Day.Format("2006-01-02")] = r.Cnt
	}

	// 3) Active users per day + platform.
	type PlatformRow struct {
		Day      time.Time
		Platform string
		Cnt      int64
	}
	var pRows []PlatformRow
	if err := db.Raw(`
        SELECT DATE(event_time) AS day, platform, COUNT(DISTINCT user_id) AS cnt
        FROM user_events
        WHERE event_time >= ?
        GROUP BY DATE(event_time), platform
        ORDER BY day, platform
    `, start).Scan(&pRows).Error; err != nil {
		return nil, err
	}
	platformMap := make(map[string]map[string]int64)
	for _, r := range pRows {
		dayStr := r.Day.Format("2006-01-02")
		if platformMap[dayStr] == nil {
			platformMap[dayStr] = make(map[string]int64)
		}
		platformMap[dayStr][r.Platform] = r.Cnt
	}

	// 4) Active users per day + region.
	type RegionRow struct {
		Day    time.Time
		Region string
		Cnt    int64
	}
	var rRows []RegionRow
	if err := db.Raw(`
        SELECT DATE(event_time) AS day, region, COUNT(DISTINCT user_id) AS cnt
        FROM user_events
        WHERE event_time >= ?
        GROUP BY DATE(event_time), region
        ORDER BY day, region
    `, start).Scan(&rRows).Error; err != nil {
		return nil, err
	}
	regionMap := make(map[string]map[string]int64)
	for _, r := range rRows {
		dayStr := r.Day.Format("2006-01-02")
		if regionMap[dayStr] == nil {
			regionMap[dayStr] = make(map[string]int64)
		}
		regionMap[dayStr][r.Region] = r.Cnt
	}

	// 5) Build continuous N days result.
	res := make([]DailySummary, 0, days)
	for d := 0; d < days; d++ {
		day := start.AddDate(0, 0, d)
		dayStr := day.Format("2006-01-02")
		newUsers := newMap[dayStr]
		activeUsers := activeMap[dayStr]
		onlineUsers := activeUsers // simplified: online == active

		res = append(res, DailySummary{
			Date:           dayStr,
			NewUsers:       newUsers,
			ActiveUsers:    activeUsers,
			OnlineUsers:    onlineUsers,
			PlatformActive: platformMap[dayStr],
			RegionActive:   regionMap[dayStr],
		})
	}

	return res, nil
}


