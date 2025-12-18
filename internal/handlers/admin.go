package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"appstats/internal/stats"
)

// AdminPageHandler renders the admin dashboard with embedded stats data for charts.
func AdminPageHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		data, err := stats.GetLastNDaysSummary(db, 7)
		if err != nil {
			c.String(http.StatusInternalServerError, "load stats error: %v", err)
			return
		}

		b, err := json.Marshal(data)
		if err != nil {
			c.String(http.StatusInternalServerError, "json error: %v", err)
			return
		}

		html := fmt.Sprintf(adminHTMLTemplate, string(b))
		c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(html))
	}
}

// adminHTMLTemplate is the HTML template for the admin dashboard, with %s placeholder for JSON stats.
const adminHTMLTemplate = `
<!DOCTYPE html>
<html lang="zh-CN">
<head>
  <meta charset="UTF-8">
  <title>APP 运营统计</title>
  <script src="https://cdn.jsdelivr.net/npm/chart.js"></script>
  <style>
    body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif; padding: 20px; }
    .chart-container { width: 100%%; max-width: 900px; margin-bottom: 40px; }
  </style>
</head>
<body>
  <h2>APP 运营统计（最近 7 天）</h2>

  <div style="margin-bottom: 16px;">
    <label for="viewMode">时间维度：</label>
    <select id="viewMode">
      <option value="day">按日</option>
      <option value="week">按周</option>
      <option value="month">按月</option>
    </select>
  </div>

  <div class="chart-container">
    <canvas id="dailyChart"></canvas>
  </div>

  <div class="chart-container">
    <canvas id="platformChart"></canvas>
  </div>

  <!-- Server-embedded statistics data -->
  <script>
    const DAILY_STATS = %s;
  </script>

  <script>
    let rawStats = DAILY_STATS || [];
    let dailyChartInstance = null;
    let platformChartInstance = null;

    function aggregateStats(mode) {
      if (mode === 'day') {
        return rawStats;
      }

      const map = {};
      for (const d of rawStats) {
        const dateObj = new Date(d.date);
        let key;

        if (mode === 'week') {
          const year = dateObj.getFullYear();
          const firstDay = new Date(year, 0, 1);
          const diff = (dateObj - firstDay) / 86400000;
          const week = Math.ceil((diff + firstDay.getDay() + 1) / 7);
          const weekStr = week < 10 ? '0' + week : '' + week;
          key = year + '-W' + weekStr;
        } else if (mode === 'month') {
          const year = dateObj.getFullYear();
          const month = dateObj.getMonth() + 1;
          const monthStr = month < 10 ? '0' + month : '' + month;
          key = year + '-' + monthStr;
        } else {
          key = d.date;
        }

        let agg = map[key];
        if (!agg) {
          agg = {
            date: key,
            new_users: 0,
            active_users: 0,
            online_users: 0,
            platform_active: {}
          };
          map[key] = agg;
        }

        agg.new_users += d.new_users || 0;
        agg.active_users += d.active_users || 0;
        agg.online_users += d.online_users || 0;

        const pa = d.platform_active || {};
        for (const p in pa) {
          if (!Object.prototype.hasOwnProperty.call(pa, p)) continue;
          agg.platform_active[p] = (agg.platform_active[p] || 0) + (pa[p] || 0);
        }
      }

      const keys = Object.keys(map).sort();
      return keys.map(k => map[k]);
    }

    function renderDailyChart(data) {
      const labels = data.map(d => d.date);
      const newUsers = data.map(d => d.new_users);
      const activeUsers = data.map(d => d.active_users);
      const onlineUsers = data.map(d => d.online_users);

      const ctx = document.getElementById('dailyChart').getContext('2d');
      return new Chart(ctx, {
        type: 'line',
        data: {
          labels,
          datasets: [
            {
              label: '新增用户',
              data: newUsers,
              borderColor: 'rgba(75, 192, 192, 1)',
              backgroundColor: 'rgba(75, 192, 192, 0.2)',
              tension: 0.2,
            },
            {
              label: '活跃用户',
              data: activeUsers,
              borderColor: 'rgba(54, 162, 235, 1)',
              backgroundColor: 'rgba(54, 162, 235, 0.2)',
              tension: 0.2,
            },
            {
              label: '在线用户',
              data: onlineUsers,
              borderColor: 'rgba(255, 159, 64, 1)',
              backgroundColor: 'rgba(255, 159, 64, 0.2)',
              tension: 0.2,
            }
          ]
        },
        options: {
          responsive: true,
          scales: {
            y: { beginAtZero: true, ticks: { precision: 0 } }
          }
        }
      });
    }

    function renderPlatformChart(data) {
      const labels = data.map(d => d.date);

      const platforms = ['ios', 'android', 'web'];
      const colors = {
        ios: 'rgba(54, 162, 235, 0.7)',
        android: 'rgba(75, 192, 192, 0.7)',
        web: 'rgba(255, 206, 86, 0.7)'
      };
      const borderColors = {
        ios: 'rgba(54, 162, 235, 1)',
        android: 'rgba(75, 192, 192, 1)',
        web: 'rgba(255, 206, 86, 1)'
      };

      const datasets = platforms.map(p => {
        const arr = data.map(d => {
          const pa = d.platform_active || {};
          return pa[p] || 0;
        });
        return {
          label: p.toUpperCase() + ' 活跃',
          data: arr,
          backgroundColor: colors[p],
          borderColor: borderColors[p],
          borderWidth: 1
        };
      });

      const ctx = document.getElementById('platformChart').getContext('2d');
      return new Chart(ctx, {
        type: 'bar',
        data: {
          labels,
          datasets: datasets
        },
        options: {
          responsive: true,
          plugins: {
            title: {
              display: true,
              text: '按系统平台的每日活跃用户（堆叠）'
            }
          },
          scales: {
            x: { stacked: true },
            y: { stacked: true, beginAtZero: true, ticks: { precision: 0 } }
          }
        }
      });
    }

    function redrawCharts(mode) {
      const data = aggregateStats(mode);

      if (dailyChartInstance) {
        dailyChartInstance.destroy();
      }
      if (platformChartInstance) {
        platformChartInstance.destroy();
      }

      dailyChartInstance = renderDailyChart(data);
      platformChartInstance = renderPlatformChart(data);
    }

    (function init() {
      const select = document.getElementById('viewMode');
      select.addEventListener('change', function () {
        redrawCharts(this.value);
      });

      // 默认“按日”视图
      redrawCharts('day');
    })();
  </script>
</body>
</html>
`
