# Auth0 Setup Guide

This guide will walk you through setting up Auth0 for the Movie Database application with your development tenant.

## Prerequisites

- Auth0 development tenant created
- Access to your Auth0 Dashboard

## Step 1: Create the API (Backend Authentication)

1. **Navigate to APIs**:
   - Go to your Auth0 Dashboard
   - Click on "APIs" in the left sidebar
   - Click "Create API"

2. **Configure the API**:
   - **Name**: `Movie Database API`
   - **Identifier**: `https://moviedb-api` (this will be your audience)
   - **Signing Algorithm**: `RS256`
   - Click "Create"

3. **Configure API Settings**:
   - In your new API, go to the "Settings" tab
   - **Token Expiration**: Set to `86400` (24 hours)
   - **Allow Offline Access**: Keep disabled for now
   - **Allow Skipping User Consent**: Enable this (for development)
   - Click "Save Changes"

4. **Configure Scopes** (Optional for now):
   - Go to the "Scopes" tab
   - You can add scopes later as needed (e.g., `read:movies`, `write:movies`)

## Step 2: Create the Single Page Application (Frontend)

1. **Navigate to Applications**:
   - Go to "Applications" in the left sidebar
   - Click "Create Application"

2. **Configure the Application**:
   - **Name**: `Movie Database Web App`
   - **Application Type**: Select "Single Page Web Applications"
   - Click "Create"

3. **Configure Application Settings**:
   - Go to the "Settings" tab of your new application
   - **Allowed Callback URLs**: 
     ```
     http://localhost:3000, http://localhost:3000/callback
     ```
   - **Allowed Logout URLs**:
     ```
     http://localhost:3000
     ```
   - **Allowed Web Origins**:
     ```
     http://localhost:3000
     ```
   - **Allowed Origins (CORS)**:
     ```
     http://localhost:3000
     ```
   - Click "Save Changes"

4. **Note Important Values**:
   - **Domain**: `your-tenant.auth0.com`
   - **Client ID**: Copy this value
   - **Client Secret**: Not needed for SPA

## Step 3: Configure Environment Variables

1. **Backend Environment** (`.env`):
   ```bash
   AUTH0_DOMAIN=your-tenant.auth0.com
   AUTH0_AUDIENCE=https://moviedb-api
   TMDB_API_KEY=your-tmdb-key-here
   DATABASE_PATH=./moviedb.db
   PORT=8080
   STATIC_DIR=./web/dist
   ENV=development
   ```

2. **Frontend Environment** (`web/.env.local`):
   ```bash
   VITE_AUTH0_DOMAIN=your-tenant.auth0.com
   VITE_AUTH0_CLIENT_ID=your-spa-client-id-here
   VITE_AUTH0_AUDIENCE=https://moviedb-api
   VITE_API_BASE_URL=http://localhost:8080/api
   ```

## Step 4: Test the Setup

1. **Start the Application**:
   ```bash
   make install
   make dev
   ```

2. **Test Authentication**:
   - Visit `http://localhost:3000`
   - The app should load without Auth0 errors
   - Authentication flow will be implemented in Phase 2

## Step 5: Configure User Profile

1. **Go to User Management > Users**:
   - This is where you'll see registered users
   - You can manually create test users here if needed

2. **Configure User Profile**:
   - Go to "Actions" > "Flows" > "Login"
   - You can add custom actions later to enrich user profiles

## Step 6: Security Best Practices

1. **Rate Limiting**:
   - Go to "Security" > "Attack Protection"
   - Enable "Brute Force Protection"
   - Enable "Suspicious IP Throttling"

2. **Anomaly Detection**:
   - Enable "Breached Password Detection"

## Production Setup (Later)

When you're ready to deploy:

1. **Update Callback URLs** to include your production domain:
   ```
   https://yourdomain.com, https://yourdomain.com/callback
   ```

2. **Update Environment Variables** with production values

3. **Consider Custom Domain** (Auth0 paid plans)

## Troubleshooting

### Common Issues:

1. **CORS Errors**:
   - Ensure `http://localhost:3000` is in "Allowed Origins (CORS)"
   - Check that frontend is running on port 3000

2. **Invalid Audience**:
   - Verify `AUTH0_AUDIENCE` in backend matches API identifier
   - Verify `VITE_AUTH0_AUDIENCE` in frontend matches API identifier

3. **Login Redirect Issues**:
   - Check "Allowed Callback URLs" includes your callback URL
   - Verify domain and client ID are correct

4. **Token Validation Errors**:
   - Ensure API uses RS256 signing algorithm
   - Check that audience matches between frontend and backend

### Testing Auth0 Setup

You can test your Auth0 configuration before implementing the full authentication flow:

1. **Test API Configuration**:
   ```bash
   curl -X GET http://localhost:8080/health
   ```
   This should return "OK" (no auth required)

2. **Test Protected Endpoint**:
   ```bash
   curl -X GET http://localhost:8080/api/me
   ```
   This should return 401 Unauthorized (auth required)

## Next Steps

Once Auth0 is configured:

1. Implement login/logout buttons in React
2. Add protected routes
3. Implement user registration flow
4. Add user profile management
5. Test the complete authentication flow

Your Auth0 setup is now ready for development. The actual authentication implementation will be added in Phase 2 of the project.