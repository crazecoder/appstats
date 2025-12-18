package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

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

		// Use a simple placeholder replacement to avoid fmt.Sprintf issues with '%' in JS.
		html := strings.Replace(adminHTMLTemplate, "__DAILY_STATS__", string(b), 1)
		c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(html))
	}
}

// adminHTMLTemplate is the HTML template for the admin dashboard.
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

  <div class="chart-container">
    <canvas id="regionChart"></canvas>
  </div>

  <!-- Server-embedded statistics data -->
  <script>
    const DAILY_STATS = __DAILY_STATS__;
  </script>

  <script>
    let rawStats = DAILY_STATS || [];
    let dailyChartInstance = null;
    let platformChartInstance = null;
    let regionChartInstance = null;

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
            platform_active: {},
            region_active: {}
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

        const ra = d.region_active || {};
        for (const r in ra) {
          if (!Object.prototype.hasOwnProperty.call(ra, r)) continue;
          agg.region_active[r] = (agg.region_active[r] || 0) + (ra[r] || 0);
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

      // 标准平台列表，其他平台统一归为 "other"
      const platformOrder = ['windows', 'linux', 'macos', 'android', 'ios', 'web', 'harmonyos', 'other'];
      const colors = {
        windows: 'rgba(54, 162, 235, 0.7)',
        linux: 'rgba(75, 192, 192, 0.7)',
        macos: 'rgba(153, 102, 255, 0.7)',
        android: 'rgba(255, 206, 86, 0.7)',
        ios: 'rgba(255, 99, 132, 0.7)',
        web: 'rgba(255, 159, 64, 0.7)',
        harmonyos: 'rgba(54, 162, 235, 0.4)',
        other: 'rgba(201, 203, 207, 0.7)'
      };
      const borderColors = {
        windows: 'rgba(54, 162, 235, 1)',
        linux: 'rgba(75, 192, 192, 1)',
        macos: 'rgba(153, 102, 255, 1)',
        android: 'rgba(255, 206, 86, 1)',
        ios: 'rgba(255, 99, 132, 1)',
        web: 'rgba(255, 159, 64, 1)',
        harmonyos: 'rgba(54, 162, 235, 1)',
        other: 'rgba(201, 203, 207, 1)'
      };

      function normalizePlatform(p) {
        if (!p) return 'other';
        const key = p.toLowerCase();
        if (platformOrder.includes(key)) return key;
        return 'other';
      }

      const datasets = platformOrder.map(p => {
        const arr = data.map(d => {
          const pa = d.platform_active || {};
          let sum = 0;
          for (const raw in pa) {
            if (!Object.prototype.hasOwnProperty.call(pa, raw)) continue;
            const norm = normalizePlatform(raw);
            if (norm === p) {
              sum += pa[raw] || 0;
            }
          }
          return sum;
        });

        const hasData = arr.some(v => v > 0);
        if (!hasData) {
          return null;
        }

        const labelMap = {
          windows: 'Windows',
          linux: 'Linux',
          macos: 'macOS',
          android: 'Android',
          ios: 'iOS',
          web: 'Web',
          harmonyos: 'HarmonyOS',
          other: '其它'
        };

        return {
          label: labelMap[p] || p,
          data: arr,
          backgroundColor: colors[p],
          borderColor: borderColors[p],
          borderWidth: 1
        };
      }).filter(ds => ds !== null);

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

    function renderRegionChart(data) {
      const regionTotals = {};
      for (const d of data) {
        const ra = d.region_active || {};
        for (const r in ra) {
          if (!Object.prototype.hasOwnProperty.call(ra, r)) continue;
          regionTotals[r || '未知'] = (regionTotals[r || '未知'] || 0) + (ra[r] || 0);
        }
      }

      // 只取前 5 个地区，剩余合并为“其它”
      const entries = Object.entries(regionTotals).sort((a, b) => b[1] - a[1]);
      const top = entries.slice(0, 5);
      const others = entries.slice(5);

      let labels = top.map(e => e[0]);
      let values = top.map(e => e[1]);

      if (others.length > 0) {
        const otherSum = others.reduce((sum, e) => sum + (e[1] || 0), 0);
        labels.push('其它');
        values.push(otherSum);
      }

      const bgColors = [
        'rgba(54, 162, 235, 0.7)',
        'rgba(255, 99, 132, 0.7)',
        'rgba(255, 206, 86, 0.7)',
        'rgba(75, 192, 192, 0.7)',
        'rgba(153, 102, 255, 0.7)',
        'rgba(255, 159, 64, 0.7)'
      ];
      const borderColors = [
        'rgba(54, 162, 235, 1)',
        'rgba(255, 99, 132, 1)',
        'rgba(255, 206, 86, 1)',
        'rgba(75, 192, 192, 1)',
        'rgba(153, 102, 255, 1)',
        'rgba(255, 159, 64, 1)'
      ];

      const bg = labels.map((_, i) => bgColors[i % bgColors.length]);
      const bd = labels.map((_, i) => borderColors[i % borderColors.length]);

      const ctx = document.getElementById('regionChart').getContext('2d');
      return new Chart(ctx, {
        type: 'pie',
        data: {
          labels,
          datasets: [{
            data: values,
            backgroundColor: bg,
            borderColor: bd,
            borderWidth: 1
          }]
        },
        options: {
          responsive: true,
          plugins: {
            title: {
              display: true,
              text: '用户地区分布（当前时间维度总计）'
            }
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
      if (regionChartInstance) {
        regionChartInstance.destroy();
      }

      dailyChartInstance = renderDailyChart(data);
      platformChartInstance = renderPlatformChart(data);
      regionChartInstance = renderRegionChart(data);
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
