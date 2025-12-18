package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"appstats/internal/models"
)

// ReportEventRequest is the payload for the write-only event reporting API.
type ReportEventRequest struct {
	UserID     string     `json:"user_id" binding:"required"`
	Platform   string     `json:"platform" binding:"required"` // ios/android/web
	Region     string     `json:"region"`
	AppVersion string     `json:"app_version"` // 可选，用于统计 app 版本分布
	EventTime  *time.Time `json:"event_time"`
}

// ReportEventHandler accepts event reports and writes them into the database.
func ReportEventHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req ReportEventRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		eventTime := time.Now().UTC()
		if req.EventTime != nil {
			eventTime = req.EventTime.UTC()
		}

		// Insert event
		evt := models.UserEvent{
			UserID:     req.UserID,
			AppVersion: req.AppVersion,
			Platform:   req.Platform,
			Region:     req.Region,
			EventTime:  eventTime,
		}
		if err := db.Create(&evt).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create event"})
			return
		}

		// Ensure user exists; create if new to track "new users".
		var user models.User
		if err := db.Where("user_id = ?", req.UserID).First(&user).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				user = models.User{
					UserID:    req.UserID,
					FirstSeen: eventTime,
					Platform:  req.Platform,
					Region:    req.Region,
				}
				if err := db.Create(&user).Error; err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create user"})
					return
				}
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "db error"})
				return
			}
		} else {
			updates := map[string]interface{}{}
			if req.Platform != "" && user.Platform != req.Platform {
				updates["platform"] = req.Platform
			}
			if req.Region != "" && user.Region != req.Region {
				updates["region"] = req.Region
			}
			if len(updates) > 0 {
				db.Model(&user).Updates(updates)
			}
		}

		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	}
}


