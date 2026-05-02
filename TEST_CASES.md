# Test Cases for URL Shortener with Authentication

This document provides test cases for verifying the functionality of the URL shortener application with authentication.

## Authentication Endpoints

### 1. User Signup
**POST** `/auth/signup`

**Request Body:**
```json
{
  "email": "test@example.com",
  "password": "password123"
}
```

**Expected Responses:**
- **201 Created**: User successfully created
  ```json
  {
    "userId": 1,
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
  }
  ```
- **409 Conflict**: Email already registered
  ```json
  {
    "error": "email already registered"
  }
  ```
- **400 Bad Request**: Invalid JSON or missing fields
  ```json
  {
    "error": "invalid JSON"
  }
  ```
- **500 Internal Server Error**: Server error

### 2. User Login
**POST** `/auth/login`

**Request Body:**
```json
{
  "email": "test@example.com",
  "password": "password123"
}
```

**Expected Responses:**
- **200 OK**: Login successful
  ```json
  {
    "userId": 1,
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
  }
  ```
- **401 Unauthorized**: Invalid credentials
  ```json
  {
    "error": "invalid credentials"
  }
  ```
- **400 Bad Request**: Invalid JSON or missing fields
  ```json
  {
    "error": "invalid JSON"
  }
  ```

### 3. User Logout
**POST** `/auth/logout`

**Expected Response:**
- **200 OK**: Logout successful
  ```json
  {
    "message": "logged out successfully"
  }
  ```

## URL Shortener Endpoints (Protected)

All URL endpoints require authentication via Bearer token in the Authorization header:
```
Authorization: Bearer <your_token_here>
```

### 4. Create Short URL
**POST** `/api/v1/shorten`

**Request Body:**
```json
{
  "url": "https://example.com/very/long/url",
  "alias": "customAlias",  // Optional
  "expiry": "2026-12-31T23:59:59Z"  // Optional, ISO 8601 format
}
```

**Expected Responses:**
- **200 OK**: URL shortened successfully
  ```json
  {
    "short_code": "abc123",
    "short_url": "http://localhost:8000/abc123"
  }
  ```
- **400 Bad Request**: Invalid JSON, missing URL, or expiry in past
  ```json
  {
    "error": "url is required"
  }
  ```
- **409 Conflict**: Custom alias already exists
  ```json
  {
    "error": "alias already exists"
  }
  ```
- **500 Internal Server Error**: Server error

### 5. Redirect to Original URL
**GET** `/:code`

**Note:** This endpoint is now protected and requires authentication

**Expected Responses:**
- **302 Found**: Redirect to original URL
- **404 Not Found**: URL code not found
  ```json
  {
    "error": "not found"
  }
  ```
- **401 Unauthorized**: Missing or invalid authentication

### 6. Check Alias Availability
**GET** `/api/v1/alias/check/:code`

**Expected Responses:**
- **200 OK**: Alias availability status
  ```json
  {
    "available": true
  }
  ```
  or
  ```json
  {
    "available": false
  }
  ```
- **401 Unauthorized**: Missing or invalid authentication
- **500 Internal Server Error**: Server error

### 7. Get User's URLs
**GET** `/api/v1/user/urls`

**Expected Responses:**
- **200 OK**: List of user's URL short codes
  ```json
  {
    "urls": ["abc123", "xyz789"]
  }
  ```
- **401 Unauthorized**: Missing or invalid authentication
- **500 Internal Server Error**: Server error

### 8. Delete URL
**DELETE** `api/v1/:code`

**Expected Responses:**
- **200 OK**: URL deleted successfully
  ```json
  {
    "message": "URL deleted successfully"
  }
  ```
- **400 Bad Request**: Missing URL code
  ```json
  {
    "error": "URL code is required"
  }
  ```
- **401 Unauthorized**: Missing or invalid authentication or not authorized to delete this URL
- **404 Not Found**: URL not found
- **500 Internal Server Error**: Server error

## Test Flow Example

1. **Signup a new user**
   - POST `/auth/signup` with email and password
   - Save the returned token

2. **Login with the same user**
   - POST `/auth/login` with email and password
   - Save the returned token (should be different from signup token)

3. **Create a short URL**
   - POST `/api/v1/shorten` with URL data and Authorization header
   - Save the returned short_code

4. **Check alias availability**
   - GET `/api/v1/alias/check/:code` with Authorization header

5. **Get user's URLs**
   - GET `/api/v1/user/urls` with Authorization header
   - Verify the short_code appears in the list

6. **Redirect using the short URL**
   - GET `/:code` with Authorization header
   - Should redirect to the original URL

7. **Delete the URL**
   - DELETE `/api/v1/:code` with Authorization header

8. **Verify deletion**
   - GET `/api/v1/user/urls` with Authorization header
   - The short_code should no longer appear in the list
   - GET `/:code` with Authorization header should return 404

## Security Test Cases

1. **Access protected endpoints without token** - Should return 401
2. **Access protected endpoints with invalid token** - Should return 401
3. **Try to signup with existing email** - Should return 409
4. **Try to login with wrong password** - Should return 401
5. **Try to delete another user's URL** - Should return 401 or 404 (depending on implementation)
6. **Try to create URL with expired date** - Should return 400
7. **Try to create URL without required fields** - Should return 400