# Backend Test

### Requirements

- Go

### Setup

1. Install dependencies:
  ```sh
  go mod tidy
  ```

2. Run the application:
  ```sh
  go run main.go
  ```

3. The server will start on `http://localhost:8080`

## Database Schema

The database consists of three tables: `users`, `films`, and `reviews`.

### Users Table

| Column    | Type    | Description                  |
|-----------|---------|------------------------------|
| id        | integer | Primary key, auto-increment  |
| username  | text    | Unique username              |
| password  | text    | Hashed password              |

### Films Table

| Column      | Type    | Description                  |
|-------------|---------|------------------------------|
| id          | integer | Primary key, auto-increment  |
| title       | text    | Title of the film            |
| director    | text    | Director of the film         |
| year        | integer | Release year of the film     |
| description | text    | Description of the film      |

### Reviews Table

| Column    | Type    | Description                  |
|-----------|---------|------------------------------|
| id        | integer | Primary key, auto-increment  |
| film_id   | integer | Foreign key to films table   |
| review    | text    | Review text                  |
| user_id   | integer | Foreign key to users table   |

## API Endpoints

### Register

- **URL:** `/register`
- **Method:** `POST`
- **Request Body:**
  ```json
  {
    "username": "string",
    "password": "string"
  }
  ```
- **Response:**
  - `201 Created` on success
  - `409 Conflict` if username already exists
  - `400 Bad Request` if request body is invalid

### Login

- **URL:** `/login`
- **Method:** `POST`
- **Request Body:**
  ```json
  {
    "username": "string",
    "password": "string"
  }
  ```
- **Response:**
  - `200 OK` on success
  - `401 Unauthorized` if username or password is incorrect

### Logout

- **URL:** `/logout`
- **Method:** `POST`
- **Response:**
  - `200 OK` on success

### Get Films

- **URL:** `/film`
- **Method:** `GET`
- **Response:**
  - `200 OK` with list of films
  - `401 Unauthorized` if user is not logged in

### Get Film by ID

- **URL:** `/film?id={id}`
- **Method:** `GET`
- **Response:**
  - `200 OK` with film details and reviews
  - `404 Not Found` if film is not found
  - `401 Unauthorized` if user is not logged in

### Add Review

- **URL:** `/review`
- **Method:** `POST`
- **Request Body:**
  ```json
  {
    "film_id": "integer",
    "review": "string"
  }
  ```
- **Response:**
  - `201 Created` on success
  - `401 Unauthorized` if user is not logged in

### Update Review

- **URL:** `/review`
- **Method:** `PATCH`
- **Request Body:**
  ```json
  {
    "id": "integer",
    "review": "string"
  }
  ```
- **Response:**
  - `202 Accepted` on success
  - `404 Not Found` if review is not found or user is not authorized
  - `401 Unauthorized` if user is not logged in

### Delete Review

- **URL:** `/review`
- **Method:** `DELETE`
- **Request Body:**
  ```json
  {
    "id": "integer"
  }
  ```
- **Response:**
  - `202 Accepted` on success
  - `404 Not Found` if review is not found or user is not authorized
  - `401 Unauthorized` if user is not logged in