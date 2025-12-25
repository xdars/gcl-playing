package google

import (
	"context"
	"fmt"
	"log"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"
)

type CalendarService struct {
	clientID     string
	clientSecret string
}

type AvailableCalendar struct {
	ProviderCalendarID string
	Name               string
	Color              *string
	IsPrimary          bool
}

type CalendarEvent struct {
	ProviderEventID string
	Title           string
	Description     string
	Location        string
	StartTime       int64
	EndTime         int64
	IsAllDay        bool
	Status          string
	Recurrence      string
	Attendees       string
	Etag            string
	RawData         string
}

type EventsResult struct {
	Events        []CalendarEvent
	NextSyncToken string
}

func NewCalendarService(clientID, clientSecret string) *CalendarService {
	return &CalendarService{
		clientID:     clientID,
		clientSecret: clientSecret,
	}
}

func (s *CalendarService) getOAuthConfig() *oauth2.Config {
	return &oauth2.Config{
		ClientID:     s.clientID,
		ClientSecret: s.clientSecret,
		Endpoint:     google.Endpoint,
		Scopes:       []string{calendar.CalendarScope},
	}
}

func (s *CalendarService) getClient(ctx context.Context, accessToken, refreshToken string) (*calendar.Service, error) {
	config := s.getOAuthConfig()

	// Set expiry to past so oauth2 will always try to refresh if refresh token is available
	token := &oauth2.Token{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
	}

	tokenSource := config.TokenSource(ctx, token)

	validToken, err := tokenSource.Token()
	if err != nil {
		log.Printf("Failed to get valid token: %v", err)
		return nil, fmt.Errorf("failed to get valid token: %w", err)
	}

	client := config.Client(ctx, validToken)

	srv, err := calendar.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, fmt.Errorf("failed to create calendar service: %w", err)
	}

	return srv, nil
}

func (s *CalendarService) ListCalendars(accessToken, refreshToken string) ([]AvailableCalendar, error) {
	ctx := context.Background()
	srv, err := s.getClient(ctx, accessToken, refreshToken)
	if err != nil {
		return nil, err
	}

	list, err := srv.CalendarList.List().Do()
	if err != nil {
		return nil, fmt.Errorf("failed to list calendars: %w", err)
	}

	var calendars []AvailableCalendar
	for _, item := range list.Items {
		cal := AvailableCalendar{
			ProviderCalendarID: item.Id,
			Name:               item.Summary,
			IsPrimary:          item.Primary,
		}
		if item.BackgroundColor != "" {
			cal.Color = &item.BackgroundColor
		}
		calendars = append(calendars, cal)
	}

	return calendars, nil
}

func (s *CalendarService) GetCalendarEvents(accessToken, refreshToken, calendarID string, syncToken *string) (*EventsResult, error) {
	ctx := context.Background()
	srv, err := s.getClient(ctx, accessToken, refreshToken)
	if err != nil {
		return nil, err
	}

	call := srv.Events.List(calendarID).
		ShowDeleted(true).
		SingleEvents(true)

	if syncToken != nil && *syncToken != "" {
		call = call.SyncToken(*syncToken)
	}

	result := &EventsResult{
		Events: []CalendarEvent{},
	}

	err = call.Pages(ctx, func(events *calendar.Events) error {
		for _, event := range events.Items {
			calEvent := s.convertEvent(event)
			result.Events = append(result.Events, calEvent)
		}
		if events.NextSyncToken != "" {
			result.NextSyncToken = events.NextSyncToken
		}
		return nil
	})

	if err != nil {
		log.Printf("Error fetching events: %v", err)
		return nil, fmt.Errorf("failed to fetch events: %w", err)
	}

	return result, nil
}

func (s *CalendarService) convertEvent(event *calendar.Event) CalendarEvent {
	calEvent := CalendarEvent{
		ProviderEventID: event.Id,
		Title:           event.Summary,
		Description:     event.Description,
		Location:        event.Location,
		Status:          event.Status,
		Etag:            event.Etag,
	}

	if event.Start != nil {
		if event.Start.DateTime != "" {
			calEvent.IsAllDay = false
		} else if event.Start.Date != "" {
			calEvent.IsAllDay = true
		}
	}

	if event.End != nil {
		if event.End.DateTime != "" {
		} else if event.End.Date != "" {
		}
	}

	if len(event.Recurrence) > 0 {
		calEvent.Recurrence = fmt.Sprintf("%v", event.Recurrence)
	}

	if len(event.Attendees) > 0 {
		calEvent.Attendees = fmt.Sprintf("%v", event.Attendees)
	}

	return calEvent
}

func (s *CalendarService) RefreshAccessToken(refreshToken string) (string, error) {
	ctx := context.Background()
	config := s.getOAuthConfig()

	token := &oauth2.Token{
		RefreshToken: refreshToken,
	}

	tokenSource := config.TokenSource(ctx, token)
	newToken, err := tokenSource.Token()
	if err != nil {
		return "", fmt.Errorf("failed to refresh token: %w", err)
	}

	return newToken.AccessToken, nil
}
