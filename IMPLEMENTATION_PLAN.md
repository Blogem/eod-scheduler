# EoD Scheduler - Implementation Plan

This document provides a detailed, step-by-step implementation plan for the EoD Scheduler application.

## Development Phases Overview

The implementation is broken down into 10 logical phases, each building upon the previous ones:

1. **Initialize Project Structure** - Set up Go module and folders
2. **Set Up Database Layer** - SQLite connection and migrations  
3. **Create Data Models** - Go structs for all entities
4. **Build Repository Layer** - Database operations
5. **Develop Service Layer** - Business logic
6. **Build Controller Layer** - HTTP handlers
7. **Design HTML Templates** - User interface
8. **Configure Routing** - Chi router setup
9. **Add Styling** - CSS and polish
10. **Testing & Validation** - End-to-end testing

---

## Phase 1: Initialize Project Structure

### Goals
- Set up Go module with proper dependencies
- Create folder structure following Go conventions
- Verify basic setup works

### Tasks
1. **Initialize Go module**
   ```bash
   go mod init github.com/blogem/eod-scheduler
   go mod tidy
   ```

2. **Install dependencies**
   ```bash
   go get github.com/go-chi/chi/v5
   go get github.com/mattn/go-sqlite3
   ```

3. **Create folder structure**
   ```
   mkdir -p database/migrations
   mkdir -p models
   mkdir -p repositories  
   mkdir -p services
   mkdir -p controllers
   mkdir -p templates
   mkdir -p static/css
   mkdir -p static/js
   ```

4. **Create placeholder files**
   - `main.go` - Basic HTTP server setup
   - `database/sqlite.go` - Database connection placeholder
   - `.gitignore` - Ignore `*.db` files and other artifacts

### Deliverables
- ✅ Working Go module with dependencies
- ✅ Complete folder structure
- ✅ Basic `main.go` that compiles and runs

---

## Phase 2: Set Up Database Layer

### Goals
- Establish SQLite connection
- Create migration system
- Set up database schema

### Tasks
1. **Database connection (`database/sqlite.go`)**
   ```go
   // Functions: OpenDB(), CloseDB(), GetDB()
   // Handle SQLite connection lifecycle
   ```

2. **Migration system (`database/migrations/`)**
   - `001_initial_schema.sql` - All table definitions
   - `migrate.go` - Simple migration runner

3. **Schema files**
   - Create all 4 tables: team_members, working_hours, schedule_entries, schedule_state
   - Add indexes for performance (date lookups, foreign keys)

4. **Database initialization**
   - Auto-run migrations on startup
   - Create default working hours (Monday-Friday 9-5)
   - Initialize schedule_state table

### Deliverables
- ✅ SQLite database created and connected
- ✅ All tables created with proper schema
- ✅ Migration system working
- ✅ Default data populated

---

## Phase 3: Create Data Models

### Goals
- Define Go structs for all entities
- Add JSON tags for future API compatibility
- Include validation tags

### Tasks
1. **`models/team_member.go`**
   ```go
   type TeamMember struct {
       ID        int       `json:"id" db:"id"`
       Name      string    `json:"name" db:"name"`
       Email     string    `json:"email" db:"email"`
       Active    bool      `json:"active" db:"active"`
       DateAdded time.Time `json:"date_added" db:"date_added"`
   }
   ```

2. **`models/working_hours.go`**
   ```go
   type WorkingHours struct {
       ID        int    `json:"id" db:"id"`
       DayOfWeek int    `json:"day_of_week" db:"day_of_week"`
       StartTime string `json:"start_time" db:"start_time"`
       EndTime   string `json:"end_time" db:"end_time"`
       Active    bool   `json:"active" db:"active"`
   }
   ```

3. **`models/schedule.go`**
   ```go
   type ScheduleEntry struct {
       ID               int       `json:"id" db:"id"`
       Date             time.Time `json:"date" db:"date"`
       TeamMemberID     int       `json:"team_member_id" db:"team_member_id"`
       StartTime        string    `json:"start_time" db:"start_time"`
       EndTime          string    `json:"end_time" db:"end_time"`
       IsManualOverride bool      `json:"is_manual_override" db:"is_manual_override"`
       // Joined fields
       TeamMemberName   string    `json:"team_member_name,omitempty"`
   }

   type ScheduleState struct {
       ID                 int       `json:"id" db:"id"`
       LastGenerationDate time.Time `json:"last_generation_date" db:"last_generation_date"`
   }
   ```

### Deliverables
- ✅ All Go structs defined with proper tags
- ✅ Models compile without errors
- ✅ Clear separation of concerns

---

## Phase 4: Build Repository Layer

### Goals
- Implement CRUD operations for all entities
- Handle database transactions properly
- Provide clean interface for service layer

### Tasks
1. **`repositories/team_repository.go`**
   ```go
   type TeamRepository interface {
       GetAll() ([]models.TeamMember, error)
       GetByID(id int) (*models.TeamMember, error)
       Create(member *models.TeamMember) error
       Update(member *models.TeamMember) error
       Delete(id int) error
       GetActiveMembers() ([]models.TeamMember, error)
   }
   ```

2. **`repositories/working_hours_repository.go`**
   ```go
   type WorkingHoursRepository interface {
       GetAll() ([]models.WorkingHours, error)
       GetByDay(day int) (*models.WorkingHours, error)
       Update(hours *models.WorkingHours) error
       GetActiveDays() ([]models.WorkingHours, error)
   }
   ```

3. **`repositories/schedule_repository.go`**
   ```go
   type ScheduleRepository interface {
       GetByDateRange(from, to time.Time) ([]models.ScheduleEntry, error)
       Create(entry *models.ScheduleEntry) error
       Update(entry *models.ScheduleEntry) error
       Delete(id int) error
       DeleteByDateRange(from, to time.Time) error
       GetState() (*models.ScheduleState, error)
       UpdateState(state *models.ScheduleState) error
   }
   ```

4. **Error handling**
   - Define custom error types
   - Handle database constraint violations
   - Proper transaction management

### Deliverables
- ✅ All repository interfaces and implementations
- ✅ CRUD operations working correctly
- ✅ Proper error handling and transactions

---

## Phase 5: Develop Service Layer

### Goals
- Implement all business logic
- Create schedule generation algorithm
- Handle complex operations and validation

### Tasks
1. **`services/team_service.go`**
   ```go
   type TeamService interface {
       GetAllMembers() ([]models.TeamMember, error)
       CreateMember(name, email string) error
       UpdateMember(id int, name, email string) error
       DeleteMember(id int) error
       // Business logic: validate email, handle active/inactive
   }
   ```

2. **`services/working_hours_service.go`**
   ```go
   type WorkingHoursService interface {
       GetWorkingHours() ([]models.WorkingHours, error)
       UpdateWorkingHours(day int, startTime, endTime string) error
       // Business logic: validate time formats, handle day ranges
   }
   ```

3. **`services/schedule_service.go`** (Most Complex)
   ```go
   type ScheduleService interface {
       GetSchedule(from, to time.Time) ([]models.ScheduleEntry, error)
       GenerateSchedule() error
       ManualOverride(entryID int, newMemberID int) error
       RemoveOverride(entryID int) error
   }
   ```

4. **Schedule Generation Algorithm**
   - Get active team members and working hours
   - Generate entries for next 3 months
   - Implement round-robin rotation
   - Handle team member changes (regenerate from change date)
   - Skip days without working hours

5. **Business Rules**
   - Email validation
   - Time format validation (HH:MM)
   - Prevent deletion of members with future assignments
   - Handle edge cases (no team members, no working hours)

### Deliverables
- ✅ All service interfaces and implementations
- ✅ Schedule generation algorithm working
- ✅ Business validation rules implemented
- ✅ Comprehensive error handling

---

## Phase 6: Build Controller Layer

### Goals
- Create HTTP handlers for all routes
- Handle form processing and validation
- Implement proper redirects and error pages

### Tasks
1. **`controllers/team_controller.go`**
   ```go
   func (c *TeamController) ShowTeamPage(w http.ResponseWriter, r *http.Request)
   func (c *TeamController) CreateMember(w http.ResponseWriter, r *http.Request)
   func (c *TeamController) UpdateMember(w http.ResponseWriter, r *http.Request)
   func (c *TeamController) DeleteMember(w http.ResponseWriter, r *http.Request)
   ```

2. **`controllers/working_hours_controller.go`**
   ```go
   func (c *WorkingHoursController) ShowHoursPage(w http.ResponseWriter, r *http.Request)
   func (c *WorkingHoursController) UpdateHours(w http.ResponseWriter, r *http.Request)
   ```

3. **`controllers/schedule_controller.go`**
   ```go
   func (c *ScheduleController) ShowDashboard(w http.ResponseWriter, r *http.Request)
   func (c *ScheduleController) ShowSchedulePage(w http.ResponseWriter, r *http.Request)
   func (c *ScheduleController) GenerateSchedule(w http.ResponseWriter, r *http.Request)
   func (c *ScheduleController) UpdateEntry(w http.ResponseWriter, r *http.Request)
   func (c *ScheduleController) DeleteOverride(w http.ResponseWriter, r *http.Request)
   ```

4. **Form Processing**
   - Parse form data
   - Validate inputs
   - Call service layer
   - Handle success/error responses
   - Implement POST-redirect-GET pattern

5. **Template Rendering**
   - Pass data to templates
   - Handle template errors
   - Support flash messages for user feedback

### Deliverables
- ✅ All HTTP handlers implemented
- ✅ Form processing working correctly
- ✅ Proper error handling and user feedback
- ✅ Clean separation from service layer

---

## Phase 7: Design HTML Templates

### Goals
- Create clean, functional HTML templates
- Implement consistent layout and navigation
- Build forms for all user interactions

### Tasks
1. **`templates/layout.html`**
   - Base template with navigation
   - Include CSS and common elements
   - Flash message support
   - Responsive meta tags

2. **`templates/dashboard.html`**
   - Current week + next 2 weeks view
   - Schedule table with team member names
   - Quick actions (generate schedule)

3. **`templates/team.html`**
   - List all team members
   - Add new member form
   - Edit/delete actions for each member
   - Form validation feedback

4. **`templates/hours.html`**
   - Working hours configuration
   - Day-by-day time inputs
   - Enable/disable days
   - Time format help text

5. **`templates/schedule.html`**
   - Full 3-month schedule view
   - Manual override forms
   - Visual indicators for overrides
   - Date navigation

6. **Template Features**
   - Form validation styling
   - Loading states for actions
   - Confirmation dialogs (using HTML dialog)
   - Accessible forms and navigation

### Deliverables
- ✅ All HTML templates created
- ✅ Consistent layout and navigation
- ✅ Forms working with proper validation
- ✅ Clean, accessible user interface

---

## Phase 8: Configure Routing and Server

### Goals
- Set up chi router with all routes
- Connect controllers to routes
- Configure static file serving
- Implement middleware

### Tasks
1. **Main router setup (`main.go`)**
   ```go
   r := chi.NewRouter()
   
   // Middleware
   r.Use(middleware.Logger)
   r.Use(middleware.Recoverer)
   
   // Static files
   r.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.Dir("static/"))))
   
   // Routes
   r.Get("/", scheduleController.ShowDashboard)
   r.Get("/team", teamController.ShowTeamPage)
   // ... all other routes
   ```

2. **Route organization**
   - Group related routes
   - Use proper HTTP methods
   - Implement route parameters (`{id}`)

3. **Middleware**
   - Request logging
   - Panic recovery  
   - Static file serving
   - CORS headers (if needed later)

4. **Server configuration**
   - Configurable port (default 8080)
   - Graceful shutdown
   - Static asset serving

5. **Dependency injection**
   - Initialize all repositories and services
   - Pass dependencies to controllers
   - Clean startup sequence

### Deliverables
- ✅ Chi router fully configured
- ✅ All routes working correctly
- ✅ Static files served properly
- ✅ Clean server startup and shutdown

---

## Phase 9: Add Styling and Polish

### Goals
- Create clean, professional styling
- Ensure responsive design
- Add user experience improvements

### Tasks
1. **`static/css/main.css`**
   - Clean, modern CSS styling
   - Responsive grid layout
   - Form styling and validation states
   - Table styling for schedules

2. **Visual Design**
   - Consistent color scheme
   - Proper typography
   - Button and form styling
   - Loading and success states

3. **User Experience**
   - Form validation feedback
   - Confirmation messages
   - Error page styling
   - Mobile-friendly design

4. **Accessibility**
   - Proper form labels
   - Keyboard navigation
   - Screen reader support
   - High contrast support

### Deliverables
- ✅ Professional, clean styling
- ✅ Responsive design working
- ✅ Good user experience
- ✅ Accessible interface

---

## Phase 10: Testing and Validation

### Goals
- Test all features end-to-end
- Verify schedule generation logic
- Fix bugs and edge cases
- Prepare for deployment

### Tasks
1. **Feature Testing**
   - Test all CRUD operations
   - Verify schedule generation works correctly
   - Test manual overrides
   - Test form validation

2. **Edge Case Testing**
   - No team members
   - No working hours defined
   - Invalid form inputs
   - Database errors

3. **Schedule Algorithm Testing**
   - Verify round-robin rotation
   - Test team member additions/removals
   - Verify 3-month generation
   - Test manual override behavior

4. **User Experience Testing**
   - Test all user workflows
   - Verify error messages are helpful
   - Test responsive design on different screens
   - Verify all forms work correctly

5. **Performance Testing**
   - Test with larger datasets
   - Verify database queries are efficient
   - Test schedule generation performance

### Deliverables
- ✅ All features tested and working
- ✅ Schedule generation algorithm verified
- ✅ Edge cases handled properly
- ✅ Application ready for use

---

## Development Guidelines

### Code Quality
- Follow Go conventions and best practices
- Use meaningful variable and function names
- Add comments for complex business logic
- Handle errors properly throughout

### Testing Strategy
- Test each layer independently during development
- Use database transactions for testing (rollback after tests)
- Create sample data for testing different scenarios

### Git Workflow
- Commit after each phase completion
- Use descriptive commit messages
- Create branches for experimental features

### Documentation
- Document any deviations from this plan
- Update README with setup and usage instructions
- Document the schedule generation algorithm

---

## Estimated Timeline

- **Phase 1-2**: 1 day (Setup and database)
- **Phase 3-4**: 1 day (Models and repositories)
- **Phase 5**: 2 days (Services and business logic)
- **Phase 6-7**: 2 days (Controllers and templates)
- **Phase 8-9**: 1 day (Routing and styling)
- **Phase 10**: 1 day (Testing and polish)

**Total: ~8 days** for a complete, production-ready application.

---

This implementation plan provides a clear roadmap from empty project to fully functional EoD Scheduler application, with each phase building logically on the previous ones.