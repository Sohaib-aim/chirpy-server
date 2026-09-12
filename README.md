# Chirpy API

A simple REST API for creating, retrieving, updating, and deleting chirps. It also includes user authentication with JWTs and a webhook for upgrading users to Chirpy Red.

## API Endpoints

### Users

#### Create User

```http
POST /api/users
```

Creates a new user.

**Request body:**

```json
{
  "email": "user@example.com",
  "password": "password123"
}
```

**Response:**

```json
{
  "id": "user-uuid",
  "created_at": "2026-01-01T00:00:00Z",
  "updated_at": "2026-01-01T00:00:00Z",
  "email": "user@example.com",
  "is_chirpy_red": false
}
```

---

#### Login

```http
POST /api/login
```

Authenticates a user and returns JWT access and refresh tokens.

**Request body:**

```json
{
  "email": "user@example.com",
  "password": "password123"
}
```

---

#### Update User

```http
PUT /api/users
```

Updates the authenticated user's email and/or password.

Requires a valid JWT:

```http
Authorization: Bearer <token>
```

**Request body:**

```json
{
  "email": "new@example.com",
  "password": "newpassword123"
}
```

---

### Chirps

#### Create Chirp

```http
POST /api/chirps
```

Creates a new chirp for the authenticated user.

Requires:

```http
Authorization: Bearer <token>
```

**Request body:**

```json
{
  "body": "Hello, Chirpy!"
}
```

---

#### Get Chirps

```http
GET /api/chirps
```

Returns a list of chirps.

**Optional query parameters:**

| Parameter   | Values        | Default     | Description                               |
| ----------- | ------------- | ----------- | ----------------------------------------- |
| `author_id` | UUID          | All authors | Only return chirps from a specific author |
| `sort`      | `asc`, `desc` | `asc`       | Sort by `created_at`                      |

**Examples:**

```http
GET /api/chirps
```

Returns all chirps in ascending order.

```http
GET /api/chirps?author_id=<user-uuid>
```

Returns chirps from a specific user.

```http
GET /api/chirps?sort=desc
```

Returns chirps from newest to oldest.

```http
GET /api/chirps?author_id=<user-uuid>&sort=desc
```

Returns a user's chirps from newest to oldest.

---

#### Get Chirp

```http
GET /api/chirps/{chirpId}
```

Returns a single chirp by its UUID.

Example:

```http
GET /api/chirps/123e4567-e89b-12d3-a456-426614174000
```

---

#### Delete Chirp

```http
DELETE /api/chirps/{chirpId}
```

Deletes a chirp.

Requires a valid JWT:

```http
Authorization: Bearer <token>
```

Only the user who owns the chirp can delete it.

---

### Tokens

#### Refresh Token

```http
POST /api/refresh
```

Generates a new access token using a valid refresh token.

Requires:

```http
Authorization: Bearer <refresh-token>
```

---

#### Revoke Refresh Token

```http
POST /api/revoke
```

Revokes the authenticated user's refresh token.

Requires:

```http
Authorization: Bearer <refresh-token>
```

---

### Chirpy Red

#### Upgrade User

```http
POST /api/polka/webhooks
```

Webhook endpoint used to upgrade a user to Chirpy Red.

Requires a valid API key:

```http
Authorization: ApiKey <api-key>
```

The webhook listens for the `user.upgraded` event.

**Example request:**

```json
{
  "event": "user.upgraded",
  "data": {
    "user_id": "user-uuid"
  }
}
```

---

## Authentication

The API uses JWT access tokens for authenticated requests.

Include the access token in the `Authorization` header:

```http
Authorization: Bearer <token>
```

Refresh tokens are used to obtain new access tokens and can be revoked when necessary.

## Response Status Codes

| Status                      | Meaning                                                 |
| --------------------------- | ------------------------------------------------------- |
| `200 OK`                    | Request successful                                      |
| `201 Created`               | Resource successfully created                           |
| `204 No Content`            | Request successful with no response body                |
| `400 Bad Request`           | Invalid request or malformed input                      |
| `401 Unauthorized`          | Missing or invalid authentication                       |
| `403 Forbidden`             | Authenticated user is not allowed to perform the action |
| `404 Not Found`             | Resource not found                                      |
| `500 Internal Server Error` | Server/database error                                   |

## Tech Stack

* Go
* `net/http`
* PostgreSQL
* sqlc
* JWT authentication
* REST API
* Webhooks
