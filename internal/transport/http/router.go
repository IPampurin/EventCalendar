// регистрация маршрутов API календаря (CRUD + выборки по дню/неделе/месяцу)
package http

// registerRoutes - привязка путей к обработчикам
func (s *Server) registerRoutes() {

	r := s.engine
	h := s.handler

	// POST /create_event - создание события (тело JSON: CreateEventRequest)
	r.POST("/create_event", h.createEvent)

	// POST /update_event - обновление события (тело JSON: UpdateEventRequest)
	r.POST("/update_event", h.updateEvent)

	// POST /delete_event - удаление события (тело JSON: DeleteEventRequest)
	r.POST("/delete_event", h.deleteEvent)

	// GET /events_for_day - все события пользователя на календарный день (query: user_id, date)
	r.GET("/events_for_day", h.eventsForDay)

	// GET /events_for_week - события на неделю от якорной даты (query: user_id, date)
	r.GET("/events_for_week", h.eventsForWeek)

	// GET /events_for_month - события на месяц от якорной даты (query: user_id, date)
	r.GET("/events_for_month", h.eventsForMonth)
}
