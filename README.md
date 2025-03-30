# Prayerloop – Go Backend

Welcome to **Prayerloop**! This repository, **prayerloop-backend**, is the primary Go (Golang) server application for Prayerloop. It provides the API endpoints, authentication, and business logic while interfacing with **PostgreSQL** via the scripts housed in the [prayerloop-psql](https://github.com/zdelcoco/prayerloop-psql) repository.

Below, you’ll find an overview of how this backend fits into the Prayerloop ecosystem, setup instructions, API usage basics, and references to other repositories in the Prayerloop family.

---

## Project Overview

The Prayerloop platform is composed of three major repositories:

1. **[prayerloop-psql](https://github.com/zdelcoco/prayerloop-psql)**  
   Holds the PostgreSQL database schema definitions, SQL scripts, and shell scripts for setup/migrations.

2. **[prayerloop-backend (this repo)](https://github.com/zdelcoco/prayerloop-backend)**  
   A Go-based server that provides RESTful API endpoints, orchestrates database interactions, and manages user authentication/authorization.

3. **[prayerloop-mobile](https://github.com/zdelcoco/prayerloop-mobile)**  
   A React Native mobile application enabling users to interact with the Prayerloop platform on iOS and Android devices.

---

## Features

- **Golang REST API**  
  Provides endpoints for user management, prayer creation, updating, retrieval, etc.
- **Database Integration**  
  Connects to PostgreSQL using the schemas and scripts from [prayerloop-psql](https://github.com/zdelcoco/prayerloop-psql).
- **Authentication & Authorization**  
  Secure handling of user credentials, session management, and role-based checks.
- **Scalable Architecture**  
  Utilizes Go’s concurrency and performance benefits for high availability.

---

## Prerequisites

- **Go** (1.18+ recommended)
- **PostgreSQL** (12+ recommended)
- **GOPATH** properly configured (if needed), or using Go modules with a recent Go version.

---

## Setup & Installation

1. **Clone the Repository**  

  ```bash
  git clone https://github.com/zdelcoco/prayerloop-backend.git cd prayerloop-backend
  ```

2. **Install Dependencies**  
Go modules should handle dependencies automatically:

  ```bash
  go mod tidy
  ```

3. **Configure Environment Variables**  
The backend typically requires environment variables to be set for database connection details. For example:

  ```bash
  DB_HOST=localhost DB_PORT=5432 DB_USER=prayerloop_user DB_PASSWORD=some_password DB_NAME=prayerloop JWT_SECRET=supersecretkey
  ```

Adjust names/secrets as needed. You might store these in a `.env` file or supply them in your deployment environment.

4. **Database Setup**  

- Ensure you have run the [prayerloop-psql](https://github.com/zdelcoco/prayerloop-psql) SQL scripts to create the schema and any seed data in your PostgreSQL instance.
- Confirm that the configured environment variables match your PostgreSQL credentials and database name.

5. **Run the Server**  

  ```bash
  go run main.go
  ```

  or

  ```bash
  go build -o prayerloop && ./prayerloop
  ```

The application will typically listen on a configured port (e.g., `:8080`). Check logs for confirmation or error messages.

---

## Usage

Once the server is running:

- **Authentication**  
Use a JWT or session token (as configured) in the `Authorization` header for restricted endpoints.

- **JWT Session Handling**: Sessions are not stored in the database. Instead, the Go middleware issues a JWT upon login containing key details (e.g., admin status, login time), which expires after 24 hours.

- **API Endpoints**  
Access the endpoints via `http://localhost:8080` (or whatever host/port you configured). The available endpoints are:

- Endpoints requiring no auth headers
- `GET /ping`  Health check endpoint.
- `POST /login`  User login.

- User endpoints
- `POST /users`  User signup.
- `GET /users/me`  Get current user profile.
- `GET /users/:user_profile_id/groups`  Get groups for a specific user.
- `GET /users/:user_profile_id/prayers`  Get prayers for a specific user.
- `POST /users/:user_profile_id/prayers`  Create a prayer for a specific user.

- Notification endpoints
- `GET /users/:user_profile_id/notifications`  Get notifications for a specific user.
- `PATCH /users/:user_profile_id/notifications/:notification_id`  Toggle notification status for a specific user.

- Group endpoints
- `GET /groups`  Get all groups.
- `POST /groups`  Create a new group.
- `GET /groups/:group_profile_id`  Get details for a specific group.
- `PUT /groups/:group_profile_id`  Update a specific group.
- `DELETE /groups/:group_profile_id`  Delete a specific group.
- `GET /groups/:group_profile_id/prayers`  Get prayers for a specific group.
- `POST /groups/:group_profile_id/prayers`  Create a prayer for a specific group.
- `GET /groups/:group_profile_id/users`  Get users in a specific group.
- `POST /groups/:group_profile_id/users/:user_profile_id`  Add a user to a specific group.
- `DELETE /groups/:group_profile_id/users/:user_profile_id`  Remove a user from a specific group.

- Invite endpoints
- `POST /groups/:group_profile_id/invite`  Create an invite code for a specific group.
- `POST /groups/:group_profile_id/join`  Join a specific group.

- Prayer endpoints
- `PUT /prayers/:prayer_id`  Update a specific prayer.
- `DELETE /prayers/:prayer_id`  Delete a specific prayer.
- `POST /prayers/:prayer_id/access`  Add access to a specific prayer.
- `DELETE /prayers/:prayer_id/access/:prayer_access_id`  Remove access from a specific prayer.

- Admin only routes (development/debugging)
- `GET /prayers`  Get all prayers.
- `GET /prayers/:prayer_id`  Get details for a specific prayer.

---

## Testing

- To run tests, if provided:

  ```bash
  go test ./...
  ```

or refer to the `Makefile` (if included) for any specialized commands.

- Testing may include unit tests for handlers, database interactions, or integration tests.

---

## Other Prayerloop Repositories

- **[prayerloop-psql](https://github.com/zdelcoco/prayerloop-psql)**  
PostgreSQL database schema scripts and migrations.
- **[prayerloop-mobile](https://github.com/zdelcoco/prayerloop-mobile)**  
React Native mobile app for iOS/Android that consumes the backend API.

---

## Contributing

Contributions are welcome! Whether you’re adding new features, fixing bugs, or improving documentation, feel free to open an issue or pull request. For major changes, please discuss them in an issue first to avoid duplication of effort.

---

## License

This project is licensed under the [MIT License](https://opensource.org/licenses/MIT).

---

## Contact

- **Issues & Support**: [Submit an issue](https://github.com/zdelcoco/prayerloop-backend/issues) or open a discussion for any questions, proposals, or feedback.
- For more information on the overall project, see also:
- [prayerloop-psql issues](https://github.com/zdelcoco/prayerloop-psql/issues)
- [prayerloop-mobile issues](https://github.com/zdelcoco/prayerloop-mobile/issues)
