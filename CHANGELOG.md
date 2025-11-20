# Changelog

All notable changes to the Prayerloop backend API will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project uses a date-based versioning scheme: `[year].[month].[sequence]`
(e.g., 2025.11.3 is the third release in November 2025).

## [Unreleased]

### Added

- Comprehensive test coverage for controllers and middleware
- Controller tests: groupController, userController, inviteController, notificationController
- Middleware tests: authMiddleware, rateLimitMiddleware
- ~75% overall test coverage
- GitHub Actions workflow for automated testing and linting
- golangci-lint configuration for code quality

## [2025.11.3] - 2025-11-19

### Added

- **Delete Account Endpoint** - `DELETE /users/:id/account`
  - Cascade deletes all user data (prayers, groups, memberships)
  - Sends confirmation email before deletion
  - Supports mobile app delete account feature
- **Prayer Reordering Endpoint** - `PATCH /users/:id/prayers/reorder`
  - Accepts array of prayer IDs in desired order
  - Updates `display_sequence` in `prayer_access` table
  - Validates complete prayer list and unique sequences
- **Group Prayer Reordering Endpoint** - `PATCH /groups/:id/prayers/reorder`
  - Reorders prayers within a group
  - Shared order across all group members
  - Validates group membership before allowing reorder
- **User Groups Reordering Endpoint** - `PATCH /users/:id/groups/reorder`
  - Reorders user's group list
  - Updates `group_display_sequence` in `user_group` table
  - Per-user ordering (each user can have their own order)

### Changed

- **Versioning Convention** - Switched from semantic versioning to date-based versioning
  - Format: `[year].[month].[sequence]`
  - Aligns with mobile app versioning

### Fixed

- Rate limiting improvements for delete account endpoint
- Validation improvements for reorder endpoints

## [0.0.1] - 2025-11-16

### Added

- Environment variable configuration for production API URL
- CORS configuration for production domain

- **Core API Endpoints**
  - User authentication (`POST /login`)
  - User signup (`POST /users`, `POST /public/signup`)
  - JWT-based authentication middleware
  - Rate limiting middleware (different limits for different endpoint groups)

- **Prayer Management**
  - `GET /users/:id/prayers` - Get user's prayers
  - `POST /prayers` - Create prayer
  - `PATCH /prayers/:id` - Update prayer
  - `DELETE /prayers/:id` - Delete prayer
  - `PATCH /prayers/:id/answer` - Mark prayer as answered

- **Group Management**
  - `POST /groups` - Create group
  - `GET /users/:id/groups` - Get user's groups
  - `GET /groups/:id` - Get group details
  - `GET /groups/:id/prayers` - Get group prayers
  - `POST /groups/:id/prayers` - Add prayer to group
  - `DELETE /groups/:id` - Delete group (creator only)
  - `DELETE /groups/:id/users/:userId` - Remove user from group (creator only)
  - `POST /groups/:id/leave` - Leave group

- **Group Invitations**
  - `POST /groups/:id/invite` - Invite user to group
  - `POST /groups/:id/join` - Join group via invitation

- **User Profile**
  - `GET /users/:id` - Get user profile
  - `PATCH /users/:id` - Update user profile
  - `PATCH /users/:id/password` - Change password

- **Password Reset**
  - `POST /password-reset/request` - Request reset code
  - `POST /password-reset/verify` - Verify reset code
  - `POST /password-reset/reset` - Reset password with code

- **Push Notifications**
  - `POST /users/:id/register-push-token` - Register FCM token
  - `POST /notifications/send` - Send push notification (admin)
  - Firebase Admin SDK integration
  - APNs configuration for iOS notifications

- **User Preferences**
  - `GET /users/:id/preferences` - Get user preferences
  - `PATCH /users/:id/preferences` - Update preferences

- **Email Notifications** (via Resend)
  - Welcome email on signup
  - Password reset codes
  - Group invitation emails
  - Group management notifications (leave, delete, remove)

### Fixed

- Production deployment configuration
- Environment-specific configuration loading

### Security

- JWT token authentication (24-hour expiration)
- Rate limiting on all endpoints
- Password hashing with bcrypt
- Input validation and sanitization
- CORS configuration

### Database

- PostgreSQL with direct SQL queries (no ORM)
- Connection pooling
- Prepared statements for security

## Version History

- **2025.11.3** - Current release with delete account and reordering
- **0.0.1** - Initial MVP release

---

## Migration Notes

### Upgrading to 2025.11.3 from 0.0.1

- Database migration required for `display_sequence` columns
- Run migration: `002_add_display_sequence_to_prayer_access.sql`
- Run migration: `002_add_group_display_sequence_to_user_group.sql`

### API Breaking Changes

None - all new endpoints are additive
