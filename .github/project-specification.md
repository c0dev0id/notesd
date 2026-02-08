# Notes & Todo Application Specification

## 1. Overview

A cross-platform notes and todo application with offline-first architecture and opportunistic synchronization. The application supports rich text editing, embedded todos, calendar integration, and multi-user collaboration through note sharing.

## 2. Core Concepts

### 2.1 Data Model

**Everything is a Note**
- Notes are the primary data structure
- Todos can exist standalone or be embedded within notes
- Notes can be converted to todo lists where each line becomes a todo

**Note Types**
- `note`: Standard rich text note
- `todo_list`: Note where each line is automatically a todo

**Todo Integration**
- Arbitrary text in notes can be marked as todo via:
  - Markup/tag syntax
  - Context menu: Select text → "Use as todo"
- Once converted, additional metadata can be added (due date, completion status)
- Standalone todos exist without note reference

### 2.2 Sync Strategy

**Opportunistic Sync**
- Clients work fully offline
- Sync occurs when connection is available
- Each object tracks modification timestamp and device ID

**Conflict Resolution**
- Last-Write-Wins (LWW) based on modification timestamp
- Timestamp format: UTC with millisecond precision
- Soft deletes using tombstones (`deleted_at` field)

### 2.3 User Accounts & Sharing

**Authentication**
- User registration and login
- JWT-based authentication with refresh tokens
- Device identification for sync tracking

**Sharing**
- Notes can be shared with other users
- Permission levels: `read`, `write`
- Shared notes sync to all permitted users

## 3. Functional Requirements

### 3.1 Notes

**Rich Text Features**
- Text styling (bold, italic, underline, strikethrough)
- Headers (H1-H6)
- Lists (ordered, unordered)
- Code blocks with syntax highlighting
- Block quotes
- Links
- Embedded images

**Image Handling**
- Upload from device/file system
- Crop functionality
- Resize functionality
- Rotate functionality
- Images stored on server, referenced in note content

**Note Management**
- Create, read, update, delete notes
- Search notes by title and content
- Tag/categorize notes (optional enhancement)

### 3.2 Todos

**Todo Properties**
- Content text
- Completion status (boolean)
- Due date (optional)
- Parent note reference (optional)
- Line reference within note (optional)

**Todo Creation Methods**
1. Standalone todo creation
2. Convert note text to todo via selection + context menu
3. Automatic todo per line in `todo_list` type notes
4. Markup/tag syntax for inline todo creation

**Todo Metadata**
- After conversion, user can add:
  - Due date
  - Priority (optional enhancement)
  - Tags/categories (optional enhancement)

### 3.3 Calendar

**Calendar View**
- Display todos by due date
- Standard calendar grid (month/week/day views)
- Today view shows:
  - Todos due today
  - Overdue todos (past due date)

**Todo Visibility**
- Only todos with due dates appear in calendar
- Todos without due dates excluded from calendar views
- Overdue todos always appear in "today" section

### 3.4 Synchronization

**Sync Operations**
- Pull changes from server since last sync timestamp
- Push local changes to server
- Resolve conflicts using LWW strategy
- Background sync when app is running and connected

**Conflict Resolution Algorithm**
1. Client requests changes since last sync timestamp
2. Server responds with changes and detects conflicts
3. For each conflict:
   - Compare `modified_at` timestamps
   - Winner: most recent timestamp
   - If timestamps equal: use `modified_by_device` as tiebreaker
4. Client applies resolved state to local database
5. Client confirms sync completion to server

**Tombstone Handling**
- Deleted items marked with `deleted_at` timestamp
- Tombstones synced to all clients
- Tombstones periodically cleaned after all devices sync

### 3.5 Offline Support

**Local Storage**
- Full note and todo database stored locally
- All CRUD operations work offline
- Changes queued for sync when online

**Sync Indicators**
- Visual indicator of sync status (synced/syncing/offline)
- Last sync timestamp displayed
- Pending changes count (optional)

## 4. Technical Architecture

### 4.1 Data Models

```go
type Note struct {
    ID          string     `json:"id"`
    UserID      string     `json:"user_id"`
    Title       string     `json:"title"`
    Content     string     `json:"content"`      // Rich text JSON
    Type        string     `json:"type"`         // "note" | "todo_list"
    ModifiedAt  time.Time  `json:"modified_at"`
    ModifiedBy  string     `json:"modified_by_device"`
    DeletedAt   *time.Time `json:"deleted_at,omitempty"`
    CreatedAt   time.Time  `json:"created_at"`
}

type Todo struct {
    ID          string     `json:"id"`
    UserID      string     `json:"user_id"`
    NoteID      *string    `json:"note_id,omitempty"`
    LineRef     *string    `json:"line_ref,omitempty"`
    Content     string     `json:"content"`
    DueDate     *time.Time `json:"due_date,omitempty"`
    Completed   bool       `json:"completed"`
    ModifiedAt  time.Time  `json:"modified_at"`
    ModifiedBy  string     `json:"modified_by_device"`
    DeletedAt   *time.Time `json:"deleted_at,omitempty"`
    CreatedAt   time.Time  `json:"created_at"`
}

type Share struct {
    ID          string    `json:"id"`
    NoteID      string    `json:"note_id"`
    OwnerID     string    `json:"owner_id"`
    SharedWith  string    `json:"shared_with_user_id"`
    Permission  string    `json:"permission"`  // "read" | "write"
    CreatedAt   time.Time `json:"created_at"`
}

type User struct {
    ID           string    `json:"id"`
    Email        string    `json:"email"`
    PasswordHash string    `json:"-"`
    DisplayName  string    `json:"display_name"`
    CreatedAt    time.Time `json:"created_at"`
}

type Image struct {
    ID         string    `json:"id"`
    NoteID     string    `json:"note_id"`
    UserID     string    `json:"user_id"`
    Filename   string    `json:"filename"`
    MimeType   string    `json:"mime_type"`
    Size       int64     `json:"size"`
    URL        string    `json:"url"`
    CreatedAt  time.Time `json:"created_at"`
}
```

### 4.2 REST API Specification

**Authentication**
```
POST   /api/v1/auth/register
POST   /api/v1/auth/login
POST   /api/v1/auth/refresh
POST   /api/v1/auth/logout
```

**Sync**
```
GET    /api/v1/sync/changes?since=<unix_timestamp_ms>
POST   /api/v1/sync/push
```

**Notes**
```
GET    /api/v1/notes
GET    /api/v1/notes/:id
POST   /api/v1/notes
PUT    /api/v1/notes/:id
DELETE /api/v1/notes/:id
GET    /api/v1/notes/search?q=<query>
```

**Todos**
```
GET    /api/v1/todos
GET    /api/v1/todos/:id
POST   /api/v1/todos
PUT    /api/v1/todos/:id
DELETE /api/v1/todos/:id
GET    /api/v1/todos/overdue
```

**Sharing**
```
GET    /api/v1/shares
POST   /api/v1/shares
DELETE /api/v1/shares/:id
GET    /api/v1/notes/:id/shares
```

**Calendar**
```
GET    /api/v1/calendar?start=<date>&end=<date>
GET    /api/v1/calendar/today
```

**Images**
```
POST   /api/v1/images
GET    /api/v1/images/:id
DELETE /api/v1/images/:id
```

### 4.3 Authentication Flow

**JWT Token Structure**
- Access token: 15 minute expiry
- Refresh token: 30 day expiry
- Token claims include: `user_id`, `device_id`, `exp`, `iat`

**Token Storage**
- Web: localStorage for refresh token, memory for access token
- iOS: Keychain
- Android: KeyStore
- CLI: Encrypted local file

**Token Refresh**
- Client automatically refreshes access token using refresh token
- If refresh token expired, user must re-authenticate

## 5. Technology Stack

### 5.1 Server: notesd

**Language & Framework**
- Go 1.21+
- Standard library `net/http` or Chi router

**Database**
- SQLite (primary)
- Migration path to PostgreSQL for scaling

**Libraries**
- `github.com/golang-jwt/jwt/v5` - JWT authentication
- `github.com/mattn/go-sqlite3` - SQLite driver
- `golang.org/x/crypto/bcrypt` - Password hashing
- `github.com/go-chi/chi/v5` - HTTP router (optional)

**Deployment**
- Single binary daemon
- Systemd service
- Configuration via environment variables or config file

### 5.2 Web Client

**Framework**
- SvelteKit 2.0+

**Rich Text Editor**
- Tiptap (ProseMirror-based)
- Custom extensions for todo conversion

**Offline Storage**
- IndexedDB via Dexie.js
- Service Worker for offline support
- Background Sync API for opportunistic sync

**Additional Libraries**
- Tailwind CSS for styling
- date-fns for date manipulation
- Cropper.js for image manipulation

**Build & Deploy**
- Vite bundler
- PWA support via Vite PWA plugin
- Static site deployment or SvelteKit adapter

### 5.3 iOS Client

**Language & Framework**
- Swift 5.9+
- SwiftUI

**Local Storage**
- Core Data with CloudKit integration path

**Networking**
- URLSession for REST API
- Codable for JSON serialization

**Image Processing**
- Core Image for crop/rotate/resize

**Background Sync**
- Background Tasks framework

### 5.4 Android Client

**Language & Framework**
- Kotlin 1.9+
- Jetpack Compose

**Local Storage**
- Room Database

**Networking**
- Retrofit 2 + OkHttp
- Kotlin Serialization

**Image Processing**
- Android Image Cropper library

**Background Sync**
- WorkManager for periodic sync

**Additional Libraries**
- Coil for image loading
- Material 3 components

### 5.5 CLI Client

**Language & Framework**
- Go 1.21+ (matches server)

**CLI Framework**
- Cobra for command structure
- Bubble Tea for interactive TUI (optional)

**Configuration**
- Local config file (~/.notesd/config.yaml)
- Encrypted credential storage

**Features**
- Full CRUD operations via commands
- Interactive mode for note editing
- Sync command for manual synchronization

## 6. User Interface Requirements

### 6.1 Web Client

**Main Views**
- Notes list (sidebar or grid)
- Note editor (rich text)
- Todo list view
- Calendar view
- Search interface

**Note Editor**
- Toolbar with formatting options
- Inline todo conversion
- Image upload with preview
- Auto-save indicator

**Calendar**
- Month/week/day view toggle
- Today view with overdue section
- Drag-and-drop todo reschedule (enhancement)

### 6.2 Mobile Clients (iOS/Android)

**Navigation**
- Bottom tab bar: Notes, Todos, Calendar, Settings
- Note list with search
- Swipe actions (delete, share)

**Note Editor**
- Keyboard toolbar for formatting
- Context menu for todo conversion
- Image picker with crop/rotate interface

**Calendar**
- Native calendar component
- Todo detail modal
- Pull-to-refresh for sync

### 6.3 CLI Client

**Commands**
```
notesd login
notesd logout
notesd sync
notesd notes list
notesd notes create
notesd notes edit <id>
notesd notes delete <id>
notesd todos list [--overdue]
notesd todos create
notesd todos complete <id>
notesd calendar [--date=<date>]
notesd search <query>
```

**Interactive Mode**
```
notesd interactive
> Multi-line note editing
> Tab completion
> Vim-style keybindings (optional)
```

## 7. Security Requirements

### 7.1 Authentication
- Passwords hashed with bcrypt (cost factor 12)
- JWT tokens signed with RS256
- Refresh token rotation on use
- Rate limiting on auth endpoints

### 7.2 Authorization
- User can only access own notes and shared notes
- Permission checks on all endpoints
- Share permission validation (read/write)

### 7.3 Data Protection
- HTTPS/TLS 1.3 for all API communication
- No sensitive data in logs
- Image access control via signed URLs or permission check

### 7.4 Input Validation
- Request size limits
- Content sanitization for rich text
- File type validation for images
- SQL injection prevention via parameterized queries

## 8. Performance Requirements

### 8.1 Server
- Support 100+ concurrent connections
- API response time < 200ms (p95)
- Sync operation < 1s for typical dataset

### 8.2 Clients
- Note list load < 500ms
- Note editor startup < 200ms
- Offline operation with zero latency
- Sync background operation (non-blocking UI)

### 8.3 Database
- Index on user_id, modified_at, deleted_at
- Efficient query plans for sync operations
- Pagination for large datasets

## 9. Development Phases

### Phase 1: MVP
- Server with auth, notes, todos CRUD
- Web client with basic editor
- Offline storage and sync
- CLI client for testing

### Phase 2: Mobile Clients
- iOS app with core features
- Android app with core features
- Image upload and editing

### Phase 3: Enhancement
- Calendar integration
- Sharing functionality
- Advanced search
- Performance optimization

### Phase 4: Polish
- UI/UX improvements
- Background sync optimization
- Export/import features
- Multi-device testing

## 10. Open Questions

1. Maximum image size and storage strategy
2. Note/todo limit per user
3. Retention policy for deleted items (tombstone cleanup)
4. Full-text search implementation (simple LIKE vs FTS)
5. Real-time collaboration (future) vs current sync model
6. Attachment support beyond images
7. Integration with system calendars (iOS/Google Calendar)
8. Export formats (Markdown, PDF, HTML)

## 11. Future Enhancements

- End-to-end encryption
- Note templates
- Tags and folders
- Recurring todos
- Todo dependencies
- Note versioning/history
- Collaborative editing (CRDT-based)
- Desktop clients (Electron or native)
- Browser extensions
- API webhooks for integrations
