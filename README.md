# EOD Scheduler 🕐

A Go-based web application for managing engineer on duty (EOD) team scheduling with automatic rotation and manual override capabilities.

## Overview

The EOD Scheduler automates the process of assigning team members to handle engineer on duy responsibilities. It features scheduling based on working hours, supports manual overrides for special circumstances, and provides a clean web interface for team management.

## Features

### 🎯 Core Functionality
- **Automatic Schedule Generation**: Assignment of team members to days based on round robin
- **Manual Override System**: Easy rescheduling and takeovers for special circumstances  
- **Working Hours Management**: Configure team working hours by day of the week
- **Team Member Management**: Add, edit, and manage team members ~~with Slack integration~~ _Slack integration is coming soon_
- **Dashboard Overview**: Real-time view of current and upcoming schedules

### 🛠️ Technical Features
- **Clean Architecture**: Repository pattern with service layer abstraction
- **SQLite Database**: Lightweight, file-based database with migrations
- **Responsive UI**: Modern web interface with theme switching
- **RESTful API**: Clean HTTP endpoints for all operations
- **Comprehensive Testing**: Unit tests with mocks for reliable code

## Quick Start

### Prerequisites
- Go 1.24.2 or later
- SQLite3 (included with Go sqlite3 driver)

### Installation

1. **Clone the repository**
   ```bash
   git clone https://github.com/blogem/eod-scheduler.git
   cd eod-scheduler
   ```

2. **Install dependencies**
   ```bash
   go mod download
   ```

3. **Build and run**
   ```bash
   go build -o eod-scheduler main.go
   ./eod-scheduler
   ```

4. **Access the application**
   ```
   http://localhost:8080
   ```

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8080` | Server port |

## Project Structure

```
eod-scheduler/
├── main.go                 # Application entry point
├── go.mod                  # Go module definition
├── eod_scheduler.db        # SQLite database (auto-created)
├── controllers/            # HTTP request handlers
│   ├── controllers.go      # Controller registry
│   ├── dashboard_controller.go
│   ├── hours_controller.go
│   ├── schedule_controller.go
│   └── team_controller.go
├── database/               # Database layer
│   ├── sqlite.go          # Database connection
│   ├── migrate.go         # Migration runner
│   └── migrations/        # SQL migration files
├── models/                 # Data structures
│   ├── schedule.go        # Schedule entities
│   ├── team_member.go     # Team member entities
│   └── working_hours.go   # Working hours entities
├── repositories/           # Data access layer
│   ├── repositories.go    # Repository registry
│   ├── schedule_repository.go
│   ├── team_repository.go
│   ├── working_hours_repository.go
│   └── mocks/             # Test mocks
├── services/               # Business logic layer
│   ├── services.go        # Service registry
│   ├── schedule_service.go
│   ├── team_service.go
│   └── working_hours_service.go
├── static/                 # Static assets
│   ├── css/main.css       # Stylesheets
│   └── js/main.js         # JavaScript
└── templates/              # HTML templates
    ├── layout.html        # Base layout
    ├── dashboard.html     # Dashboard view
    ├── schedule.html      # Schedule management
    ├── team.html          # Team management
    └── hours.html         # Working hours config
```

## API Endpoints

### Dashboard
- `GET /` - Main dashboard view

### Team Management
- `GET /team` - Team members list
- `GET /team/edit/{id}` - Edit team member form
- `POST /team/save` - Save team member
- `POST /team/delete/{id}` - Delete team member

### Schedule Management
- `GET /schedule` - Schedule view
- `GET /schedule/edit/{date}` - Edit schedule for date
- `POST /schedule/save` - Save schedule changes
- `POST /schedule/generate` - Generate new schedules
- `POST /schedule/takeover` - Request schedule takeover

### Working Hours
- `GET /hours` - Working hours configuration
- `POST /hours/save` - Save working hours

### Static Assets
- `GET /static/*` - CSS, JavaScript, and other static files

## Data Models

### Team Member
```go
type TeamMember struct {
    ID          int    `json:"id"`
    Name        string `json:"name"`
    SlackHandle string `json:"slack_handle"`
    Active      bool   `json:"active"`
    DateAdded   string `json:"date_added"`
}
```

### Schedule Entry
```go
type ScheduleEntry struct {
    ID                   int       `json:"id"`
    Date                 time.Time `json:"date"`
    TeamMemberID         int       `json:"team_member_id"`
    StartTime            string    `json:"start_time"`
    EndTime              string    `json:"end_time"`
    IsManualOverride     bool      `json:"is_manual_override"`
    OriginalTeamMemberID *int      `json:"original_team_member_id,omitempty"`
}
```

### Working Hours
```go
type WorkingHours struct {
    ID        int    `json:"id"`
    DayOfWeek int    `json:"day_of_week"` // 0=Monday, 6=Sunday
    StartTime string `json:"start_time"`  // "09:00" format
    EndTime   string `json:"end_time"`    // "17:00" format
    Active    bool   `json:"active"`
}
```

## Development

### Running Tests
```bash
# Run all tests
go test ./...

# Run tests with verbose output
go test -v ./...

# Run tests with coverage
go test -cover ./...
```

### Database Migrations

The application automatically runs migrations on startup. Migration files are located in `database/migrations/` and follow the naming convention:
- `001_initial_schema.sql`
- `002_add_feature.sql`
- etc.

### Adding New Features

1. **Model**: Define data structures in `models/`
2. **Repository**: Add database operations in `repositories/`
3. **Service**: Implement business logic in `services/`
4. **Controller**: Create HTTP handlers in `controllers/`
5. **Templates**: Add HTML views in `templates/`
6. **Routes**: Register routes in `main.go`

## Architecture

The application follows a clean architecture pattern:

```
┌─────────────────┐
│   Controllers   │ ← HTTP handlers, request/response
├─────────────────┤
│    Services     │ ← Business logic, validation
├─────────────────┤
│  Repositories   │ ← Data access, SQL queries
├─────────────────┤
│    Database     │ ← SQLite storage
└─────────────────┘
```

### Design Principles
- **Separation of Concerns**: Each layer has a specific responsibility
- **Dependency Injection**: Services and repositories are injected via constructors
- **Interface-Based**: All dependencies use interfaces for testability
- **Repository Pattern**: Abstracts database operations behind interfaces

## Configuration

### Working Hours
Configure team working hours through the web interface at `/hours`. Default configuration:
- Monday-Friday: 09:00-17:00
- Weekend: No working hours

### Schedule Generation
The system automatically generates schedules based on:
1. Team member availability (active status)
2. Working hours configuration
3. Fair rotation algorithm
4. Previous schedule history

## Troubleshooting

### Common Issues

**Database locked error**
```bash
# Stop any running instances
pkill -f eod-scheduler
# Remove database file and restart
rm eod_scheduler.db
./eod-scheduler
```

**Port already in use**
```bash
# Use different port
PORT=8081 ./eod-scheduler
```

**Missing dependencies**
```bash
go mod download
go mod tidy
```

## Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Make your changes
4. Add tests for new functionality
5. Ensure all tests pass (`go test ./...`)
6. Commit your changes (`git commit -m 'Add amazing feature'`)
7. Push to the branch (`git push origin feature/amazing-feature`)
8. Open a Pull Request

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Support

For questions, issues, or contributions, please:
- Open an issue on GitHub
- Contact the developer
- Check the [Implementation Plan](IMPLEMENTATION_PLAN.md) for detailed technical documentation
