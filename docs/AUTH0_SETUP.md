# Auth0 Setup Guide

Complete setup guide for configuring Auth0 authentication with the Movie Database application.

## Prerequisites

- Auth0 account (free tier is sufficient)
- Access to Auth0 Dashboard
- Movie Database project cloned locally

## Step 1: Create Auth0 API (Backend)

### 1.1 Create API
1. Go to **Auth0 Dashboard** → **APIs**  
2. Click **"Create API"**
3. Configure:
   - **Name**: `Movie Database API`
   - **Identifier**: `https://movie.sagen.app` (use your domain or a unique identifier)
   - **Signing Algorithm**: `RS256`
4. Click **"Create"**

### 1.2 Configure API Settings
1. Go to **Settings** tab
2. Configure:
   - **Token Expiration**: `86400` (24 hours)
   - **Allow Offline Access**: ❌ Disabled
   - **Allow Skipping User Consent**: ✅ Enabled (for development)
3. Click **"Save Changes"**

> **Note**: The API Identifier becomes your `AUTH0_AUDIENCE` value

## Step 2: Create Auth0 Application (Frontend)

### 2.1 Create Application
1. Go to **Applications** → **Create Application**
2. Configure:
   - **Name**: `Movie Database Web App`
   - **Type**: **Single Page Web Applications**
3. Click **"Create"**

### 2.2 Configure Application Settings
1. Go to **Settings** tab
2. Configure URLs:
   ```
   Allowed Callback URLs:
   http://localhost:3000
   
   Allowed Logout URLs:
   http://localhost:3000
   
   Allowed Web Origins:
   http://localhost:3000
   
   Allowed Origins (CORS):
   http://localhost:3000
   ```
3. Click **"Save Changes"**

### 2.3 Note Important Values
From the **Settings** tab, copy these values:
- **Domain**: `your-tenant.auth0.com`
- **Client ID**: `your-client-id-here`

## Step 3: Configure Environment Variables

### 3.1 Backend Configuration
Create/update `.env` in project root:
```bash
# Auth0 Configuration
AUTH0_DOMAIN=your-tenant.auth0.com
AUTH0_AUDIENCE=https://movie.sagen.app

# TMDB API
TMDB_API_KEY=your-tmdb-api-key

# Database & Server
DATABASE_PATH=./moviedb.db
PORT=8080
STATIC_DIR=./web/dist
ENV=development
```

### 3.2 Frontend Configuration  
Create/update `web/.env.local`:
```bash
# Auth0 Configuration
VITE_AUTH0_DOMAIN=your-tenant.auth0.com
VITE_AUTH0_CLIENT_ID=your-client-id-here
VITE_AUTH0_AUDIENCE=https://movie.sagen.app
```

> **Important**: The `AUTH0_AUDIENCE` and `VITE_AUTH0_AUDIENCE` must match exactly!

## Step 4: Test Authentication

### 4.1 Start the Application
```bash
# Install dependencies
make install

# Start development servers
make dev
```

### 4.2 Verify Setup
1. **Frontend**: Visit `http://localhost:3000`
   - Should load without Auth0 errors
   - Login/logout should work
   
2. **Backend**: Test endpoints
   ```bash
   # Public endpoint (should work)
   curl http://localhost:8080/health
   
   # Protected endpoint (should return 401)
   curl http://localhost:8080/api/me
   ```

## Step 5: Production Configuration

### 5.1 Update Auth0 Settings
When deploying to production:

1. **Add Production URLs** to Auth0 Application:
   ```
   Allowed Callback URLs:
   http://localhost:3000, https://yourdomain.com
   
   Allowed Logout URLs:
   http://localhost:3000, https://yourdomain.com
   
   Allowed Web Origins:
   http://localhost:3000, https://yourdomain.com
   
   Allowed Origins (CORS):
   http://localhost:3000, https://yourdomain.com
   ```

2. **Update Environment Variables** with production values

### 5.2 Production Build
```bash
# Update web/.env.production with production values
# Then build the production binary
make build-prod
```

## Step 6: Security Best Practices

### 6.1 Enable Security Features
In Auth0 Dashboard → **Security** → **Attack Protection**:
- ✅ **Brute Force Protection**: Enabled
- ✅ **Suspicious IP Throttling**: Enabled  
- ✅ **Breached Password Detection**: Enabled

### 6.2 Configure Login Flow
In **Actions** → **Flows** → **Login**:
- Add custom actions if needed (e.g., user profile enrichment)
- Keep default flow for standard use cases

## Troubleshooting

### Common Issues

| Issue | Solution |
|-------|----------|
| **CORS Errors** | Verify `http://localhost:3000` is in "Allowed Origins (CORS)" |
| **Invalid Audience** | Ensure `AUTH0_AUDIENCE` matches API Identifier exactly |
| **Login Redirect Fails** | Check "Allowed Callback URLs" includes your URL |
| **Token Validation Error** | Verify API uses RS256 and audience matches |
| **App Won't Load** | Check Domain and Client ID are correct |

### Debug Commands

```bash
# Test health endpoint (no auth required)
curl http://localhost:8080/health

# Test protected endpoint (should return 401)
curl http://localhost:8080/api/me

# Check if frontend can reach backend
curl http://localhost:8080/api/movies
```

### Verify Environment Variables

```bash
# Backend
cd /path/to/project
cat .env

# Frontend  
cd web
cat .env.local
```

## Complete Example Configuration

### Example `.env`:
```bash
AUTH0_DOMAIN=dev-abc123.us.auth0.com
AUTH0_AUDIENCE=https://movie.sagen.app
TMDB_API_KEY=eyJhbGciOiJIUzI1NiJ9...
DATABASE_PATH=./moviedb.db
PORT=8080
STATIC_DIR=./web/dist
ENV=development
```

### Example `web/.env.local`:
```bash
VITE_AUTH0_DOMAIN=dev-abc123.us.auth0.com
VITE_AUTH0_CLIENT_ID=abc123xyz789
VITE_AUTH0_AUDIENCE=https://movie.sagen.app
```

## What's Included

✅ **JWT Authentication** - Secure token-based auth  
✅ **User Profiles** - Auth0 user management  
✅ **Protected Routes** - Frontend route protection  
✅ **API Security** - Backend endpoint protection  
✅ **Dark Mode Support** - User preferences stored  
✅ **Mobile Responsive** - Works on all devices  

Auth0 setup is now complete! The application includes full authentication flow with login, logout, and protected routes.