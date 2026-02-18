# Authentication API

Register and authenticate users. These endpoints are **public** (no JWT required).

## Endpoints

### `POST /api/auth/register`

Create a new user account and receive a JWT token.

**Request Body:**

```json
{
  "email": "user@example.com",
  "password": "securepass123"
}
```

| Field | Type | Validation | Required |
|-------|------|------------|----------|
| `email` | string | Valid email format | Yes |
| `password` | string | Minimum 8 characters | Yes |

**Response (201 Created):**

```json
{
  "token": "eyJhbGciOiJIUzI1NiIs...",
  "user": {
    "id": 1,
    "email": "user@example.com",
    "created_at": "2026-02-19T00:00:00Z",
    "updated_at": "2026-02-19T00:00:00Z"
  }
}
```

**Error Responses:**

| Status | Body | Condition |
|--------|------|-----------|
| 400 | `{"error": "...validation details..."}` | Invalid email format or password < 8 chars |
| 409 | `{"error": "email already registered"}` | Email already exists |
| 500 | `{"error": "failed to create user"}` | Database error |

---

### `POST /api/auth/login`

Authenticate with email and password to receive a JWT token.

**Request Body:**

```json
{
  "email": "user@example.com",
  "password": "securepass123"
}
```

| Field | Type | Validation | Required |
|-------|------|------------|----------|
| `email` | string | Valid email format | Yes |
| `password` | string | Non-empty | Yes |

**Response (200 OK):**

```json
{
  "token": "eyJhbGciOiJIUzI1NiIs...",
  "user": {
    "id": 1,
    "email": "user@example.com",
    "created_at": "2026-02-19T00:00:00Z",
    "updated_at": "2026-02-19T00:00:00Z"
  }
}
```

**Error Responses:**

| Status | Body | Condition |
|--------|------|-----------|
| 400 | `{"error": "...validation details..."}` | Missing or invalid fields |
| 401 | `{"error": "invalid credentials"}` | Wrong email or password |

## Authentication

After obtaining a token, include it in all subsequent requests:

```
Authorization: Bearer <token>
```

Tokens expire after the configured `JWT_EXPIRY_HOURS` (default: 72 hours). After expiry, re-authenticate with `/api/auth/login`.
