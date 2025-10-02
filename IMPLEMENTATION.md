# EoD Scheduler

The EoD Scheduler is a simple tool to schedule the Engineer on Duty roster for a team.

## Functional Requirements

### Core Functionality
- Add team members (name, email)
- Define working hours per day of the week (different hours per day)
- Generate a rotating schedule for next 3 months automatically
- Allow manual overrides for specific days (doesn't affect rotation)
- Team changes only regenerate schedule from change date forward

### User Interface
- **Dashboard** (`/`) - Current week + next 2 weeks view
- **Team Management** (`/team`) - Add/remove/edit team members  
- **Working Hours** (`/hours`) - Configure hours per day of week
- **Schedule** (`/schedule`) - View/edit full 3-month schedule
- Simple navigation with top menu bar

### Scheduling Logic
- Round-robin rotation of active team members
- Generate 3 months ahead whenever app runs
- Only schedule days that have working hours defined
- Manual overrides don't change rotation order
- Track rotation state in database

## Technical Requirements

### Architecture
- Build in Go
- Follows controller-service-repository pattern
- Runs on local machine
- Uses SQLite as database engine
- Frontend uses Go built-in html/template package
- Traditional HTML forms with POST/GET requests (no JavaScript)

### Data Models
- **Team Member**: ID, Name, Email, Active status, DateAdded
- **Working Hours**: Day of week, Start time, End time, Active status
- **Schedule Entry**: Date, Team member ID, Start/End time, Manual override flag
- **Schedule State**: Next person index, Last generation date

### Database Schema
```sql
CREATE TABLE team_members (
    id INTEGER PRIMARY KEY,
    name TEXT NOT NULL,
    email TEXT UNIQUE,
    active BOOLEAN DEFAULT 1,
    date_added DATE DEFAULT CURRENT_DATE
);

CREATE TABLE working_hours (
    id INTEGER PRIMARY KEY,
    day_of_week INTEGER NOT NULL, -- 0=Monday, 6=Sunday
    start_time TEXT NOT NULL,     -- "09:00"
    end_time TEXT NOT NULL,       -- "17:00"
    active BOOLEAN DEFAULT 1
);

CREATE TABLE schedule_entries (
    id INTEGER PRIMARY KEY,
    date DATE NOT NULL,
    team_member_id INTEGER,
    start_time TEXT NOT NULL,
    end_time TEXT NOT NULL,
    is_manual_override BOOLEAN DEFAULT 0,
    FOREIGN KEY (team_member_id) REFERENCES team_members(id)
);

CREATE TABLE schedule_state (
    id INTEGER PRIMARY KEY,
    next_person_index INTEGER DEFAULT 0,
    last_generation_date DATE
);
```

### Project Structure
```
eod-scheduler/
├── main.go
├── go.mod
├── database/
│   ├── migrations/
│   └── sqlite.go
├── models/
│   ├── team_member.go
│   ├── working_hours.go
│   └── schedule.go
├── repositories/
│   ├── team_repository.go
│   ├── schedule_repository.go
│   └── working_hours_repository.go
├── services/
│   ├── team_service.go
│   ├── schedule_service.go
│   └── working_hours_service.go
├── controllers/
│   ├── team_controller.go
│   ├── schedule_controller.go
│   └── working_hours_controller.go
├── templates/
│   ├── layout.html
│   ├── dashboard.html
│   ├── team.html
│   ├── hours.html
│   └── schedule.html
└── static/
    ├── css/
    └── js/
```

### Routes
```
# Pages (render HTML templates)
GET    /                      - Dashboard page
GET    /team                  - Team management page  
GET    /hours                 - Working hours configuration page
GET    /schedule              - Schedule view page

# Form Actions (handle form submissions)
POST   /team                  - Create new team member
POST   /team/{id}/edit        - Update team member
POST   /team/{id}/delete      - Delete team member

POST   /hours                 - Update working hours
POST   /hours/{day}           - Update working hours for specific day

POST   /schedule/generate     - Trigger schedule generation
POST   /schedule/{id}/edit    - Update schedule entry (manual override)
POST   /schedule/{id}/delete  - Remove manual override
```

## Implementation Notes

### Dependencies
- `github.com/mattn/go-sqlite3` - SQLite driver
- `github.com/go-chi/chi/v5` - HTTP router
- `net/http` - Built-in HTTP server  
- `html/template` - Built-in templating
- `database/sql` - Built-in database interface

### Key Features  
- Auto-generate schedule on app startup if needed
- Traditional HTML forms with POST redirects
- Server-side rendering with html/template
- No authentication (local tool)
- Basic CSS for clean presentation
- No JavaScript required
