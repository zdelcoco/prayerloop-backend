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

- **API Endpoints**  
Access the endpoints via `http://localhost:8080` (or whatever host/port you configured).  
Typical routes might include:
- `POST /api/v1/login`
- `POST /api/v1/register`
- `GET /api/v1/prayers`
- `POST /api/v1/prayers`
- `PUT /api/v1/prayers/:id`
- `DELETE /api/v1/prayers/:id`

(Actual route definitions may vary based on project implementation. Check the code or documentation comments for specifics.)

- **Authentication**  
Use a JWT or session token (as configured) in the `Authorization` header for restricted endpoints.

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

This project is licensed under the [MIT License](https://opensource.org/licenses/MIT). See the [LICENSE](LICENSE) file for details (if provided).

---

## Contact

- **Issues & Support**: [Submit an issue](https://github.com/zdelcoco/prayerloop-backend/issues) or open a discussion for any questions, proposals, or feedback.
- For more information on the overall project, see also:
- [prayerloop-psql issues](https://github.com/zdelcoco/prayerloop-psql/issues)
- [prayerloop-mobile issues](https://github.com/zdelcoco/prayerloop-mobile/issues)
