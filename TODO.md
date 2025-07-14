# MovieDB TODO List

## High Priority (Performance & Security)
- [ ] **Database Transactions**: Wrap all sync operations in transactions with rollback
- [ ] **Token Encryption**: Encrypt Plex tokens before database storage
- [ ] **WebSocket Implementation**: Replace polling with real-time updates for sync progress
- [ ] **Database Indexes**: Add performance indexes for large-scale operations
- [ ] **Rate Limiter Race Conditions**: Improve synchronization in token bucket algorithm

## Medium Priority (Optimization)
- [ ] **Memory Management**: Implement batch processing for large libraries (>1000 items)
- [ ] **Context Cancellation**: Add ctx.Done() checks in all long-running operations
- [ ] **Error Recovery**: Implement structured error handling with retry mechanisms
- [ ] **Job Deduplication**: Add database constraints to prevent duplicate job creation
- [ ] **Structured Logging**: Replace fmt.Printf with structured logging system

## Low Priority (Features & Monitoring)
- [ ] **Sync Statistics**: Add detailed sync metrics and performance monitoring
- [ ] **Partial Sync**: Allow syncing specific libraries instead of full sync only
- [ ] **Sync Scheduling**: Add user-configurable automatic sync intervals
- [ ] **Library Filtering**: Allow users to choose which libraries to sync
- [ ] **Sync History**: Extended sync history with detailed operation logs

## Technical Debt
- [ ] **API Response Types**: Create proper response structs instead of generic maps
- [ ] **Configuration**: Move hardcoded values to configuration files
- [ ] **Unit Tests**: Add comprehensive test coverage for sync operations
- [ ] **Documentation**: Add API documentation and developer guides
- [ ] **Performance Testing**: Load testing for large libraries and multiple users

## Security Enhancements
- [ ] **API Rate Limiting**: Add rate limiting to prevent API abuse
- [ ] **Input Sanitization**: Comprehensive input validation for all endpoints
- [ ] **Audit Logging**: Track all user actions for security monitoring
- [ ] **Token Rotation**: Implement automatic Plex token refresh
- [ ] **Permission Validation**: Double-check user permissions before operations

## Future Features
- [ ] **TV Shows Support**: Extend sync to handle TV shows and episodes
- [ ] **Multi-Server Optimization**: Optimize for users with many Plex servers
- [ ] **Library Sharing**: Allow users to share library access with friends
- [ ] **Sync Conflicts**: Handle conflicts when multiple users sync same library
- [ ] **Backup & Recovery**: Implement sync data backup and recovery procedures