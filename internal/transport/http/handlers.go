// обработчики HTTP
package http

import (
	nethttp "net/http"

	"github.com/gin-gonic/gin"
)

// Handler - зависимости сервиса
type Handler struct{}

// NewHandler - конструктор обработчиков
func NewHandler() *Handler {

	return &Handler{}
}

// createEvent - POST /create_event
func (h *Handler) createEvent(c *gin.Context) {

	c.JSON(nethttp.StatusOK, gin.H{"result": "заглушка create_event"})
}

// updateEvent - POST /update_event
func (h *Handler) updateEvent(c *gin.Context) {

	c.JSON(nethttp.StatusOK, gin.H{"result": "заглушка update_event"})
}

// deleteEvent - POST /delete_event
func (h *Handler) deleteEvent(c *gin.Context) {

	c.JSON(nethttp.StatusOK, gin.H{"result": "заглушка delete_event"})
}

// eventsForDay - GET /events_for_day
func (h *Handler) eventsForDay(c *gin.Context) {

	c.JSON(nethttp.StatusOK, gin.H{"result": "заглушка events_for_day"})
}

// eventsForWeek - GET /events_for_week
func (h *Handler) eventsForWeek(c *gin.Context) {

	c.JSON(nethttp.StatusOK, gin.H{"result": "заглушка events_for_week"})
}

// eventsForMonth - GET /events_for_month
func (h *Handler) eventsForMonth(c *gin.Context) {

	c.JSON(nethttp.StatusOK, gin.H{"result": "заглушка events_for_month"})
}
