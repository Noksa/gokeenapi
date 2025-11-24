# Requirements Document

## Introduction

This document specifies the requirements for a unified, comprehensive mock implementation of the Keenetic router API. The current testing infrastructure uses multiple separate mock servers scattered across test files, making it difficult to maintain, extend, and ensure consistency. The unified mock system will provide a single, centralized, stateful mock that simulates the complete Keenetic router API, enabling comprehensive testing while being easy to maintain and extend.

## Glossary

- **Keenetic Router**: A network router device manufactured by Keenetic (formerly Zyxel Keenetic) that provides a REST API for configuration and management
- **Mock Server**: A test double that simulates the Keenetic router's HTTP API endpoints for testing purposes
- **RCI (Router Command Interface)**: The REST API protocol used by Keenetic routers for configuration and monitoring
- **WireGuard Interface**: A VPN interface type supported by Keenetic routers using the WireGuard protocol
- **AWG (Amnezia WireGuard)**: An enhanced WireGuard configuration with additional obfuscation parameters (Jc, Jmin, Jmax, S1, S2, H1-H4)
- **Hotspot**: A connected device on the router's network, identified by MAC address, IP, and hostname
- **Static Route**: A manually configured network route that directs traffic for specific networks through specific interfaces
- **DNS Record**: A static DNS entry that maps domain names to IP addresses
- **Parse Endpoint**: The RCI endpoint (`/rci/`) that accepts CLI-style commands in JSON format for router configuration
- **Test Suite**: A collection of related tests organized using the testify suite package
- **State Management**: The ability of the mock to maintain and modify router configuration state across multiple API calls
- **Migration Path**: A documented process for transitioning existing tests from old mocks to the new unified mock

## Requirements

### Requirement 1

**User Story:** As a developer, I want a single unified mock server that simulates all Keenetic router API endpoints, so that I can test all router interactions without maintaining multiple separate mocks

#### Acceptance Criteria

1. WHEN the unified mock is initialized THEN the system SHALL provide all endpoints currently implemented across separate mocks (auth, version, interfaces, routes, DNS, hotspot, parse, running-config, system-mode)
2. WHEN a test requires router API interaction THEN the system SHALL provide a single mock instance that handles all endpoint types
3. WHEN the mock is created THEN the system SHALL initialize with sensible default state for all router resources
4. WHEN multiple tests use the mock THEN the system SHALL support independent mock instances to prevent test interference
5. THE unified mock SHALL support all HTTP methods currently used (GET, POST) for each endpoint

### Requirement 2

**User Story:** As a developer, I want the mock to maintain stateful behavior, so that I can test sequences of operations that modify router configuration

#### Acceptance Criteria

1. WHEN a route is added via the parse endpoint THEN the system SHALL include that route in subsequent route listing requests
2. WHEN a DNS record is added via the parse endpoint THEN the system SHALL include that record in subsequent DNS listing requests
3. WHEN an interface configuration is modified via the parse endpoint THEN the system SHALL reflect those changes in subsequent interface queries
4. WHEN a known host is deleted via the parse endpoint THEN the system SHALL exclude that host from subsequent hotspot queries
5. WHEN the mock state is modified THEN the system SHALL maintain consistency across all related endpoints

### Requirement 3

**User Story:** As a developer, I want to easily configure the mock's initial state, so that I can set up specific test scenarios without complex setup code

#### Acceptance Criteria

1. WHEN creating a mock instance THEN the system SHALL accept optional configuration for initial interfaces
2. WHEN creating a mock instance THEN the system SHALL accept optional configuration for initial routes
3. WHEN creating a mock instance THEN the system SHALL accept optional configuration for initial DNS records
4. WHEN creating a mock instance THEN the system SHALL accept optional configuration for initial hotspot devices
5. WHERE custom initial state is not provided THEN the system SHALL use reasonable defaults that support common test scenarios

### Requirement 4

**User Story:** As a developer, I want the mock to validate requests and return appropriate errors, so that I can test error handling in my code

#### Acceptance Criteria

1. WHEN an invalid interface ID is queried THEN the system SHALL return HTTP 404 with appropriate error response
2. WHEN invalid authentication credentials are provided THEN the system SHALL return HTTP 401 with authentication challenge
3. WHEN a malformed parse command is submitted THEN the system SHALL return a parse response with error status
4. WHEN a POST request is made to a GET-only endpoint THEN the system SHALL return HTTP 405 Method Not Allowed
5. WHEN invalid JSON is submitted to the parse endpoint THEN the system SHALL return HTTP 400 Bad Request

### Requirement 5

**User Story:** As a developer, I want comprehensive support for WireGuard and AWG interface operations, so that I can test all VPN-related functionality

#### Acceptance Criteria

1. WHEN querying interfaces THEN the system SHALL return WireGuard interfaces with all required fields (id, type, description, address, connected, link, state)
2. WHEN querying SC interfaces THEN the system SHALL return WireGuard interfaces with AWG configuration parameters (Jc, Jmin, Jmax, S1, S2, H1, H2, H3, H4)
3. WHEN AWG parameters are updated via parse endpoint THEN the system SHALL reflect the new values in subsequent SC interface queries
4. WHEN a WireGuard interface is created via parse endpoint THEN the system SHALL add it to both interface and SC interface listings
5. WHEN a WireGuard interface state is changed via parse endpoint THEN the system SHALL update the connected, link, and state fields accordingly

### Requirement 6

**User Story:** As a developer, I want the mock to support route operations with proper validation, so that I can test route management functionality comprehensively

#### Acceptance Criteria

1. WHEN routes are queried for a specific interface THEN the system SHALL return only routes associated with that interface
2. WHEN a route is added via parse endpoint THEN the system SHALL validate that the target interface exists
3. WHEN a route is deleted via parse endpoint THEN the system SHALL remove it from the route list
4. WHEN duplicate routes are added THEN the system SHALL handle them according to Keenetic router behavior
5. WHEN routes are queried THEN the system SHALL return all required fields (network, host, mask, interface, auto)

### Requirement 7

**User Story:** As a developer, I want the mock to support DNS record operations, so that I can test DNS management functionality

#### Acceptance Criteria

1. WHEN DNS records are queried THEN the system SHALL return all static DNS entries in the expected format
2. WHEN a DNS record is added via parse endpoint THEN the system SHALL include it in subsequent DNS queries
3. WHEN a DNS record is deleted via parse endpoint THEN the system SHALL remove it from the DNS record list
4. WHEN duplicate DNS records are added THEN the system SHALL handle them according to Keenetic router behavior
5. WHEN DNS records are queried THEN the system SHALL return the data in the format expected by the API client (map with "static" key)

### Requirement 8

**User Story:** As a developer, I want the mock to support hotspot (known hosts) operations, so that I can test device management functionality

#### Acceptance Criteria

1. WHEN hotspot data is queried THEN the system SHALL return all connected devices with required fields (name, mac, ip, hostname)
2. WHEN a known host is deleted via parse endpoint THEN the system SHALL remove it from the hotspot list
3. WHEN hotspot data is queried THEN the system SHALL return data in the expected RciShowIpHotspot format
4. WHEN multiple devices exist THEN the system SHALL return all devices in a single response
5. WHEN the hotspot list is empty THEN the system SHALL return an empty host array

### Requirement 9

**User Story:** As a developer, I want the mock to properly simulate authentication, so that I can test authentication flows

#### Acceptance Criteria

1. WHEN a GET request is made to /auth THEN the system SHALL return HTTP 401 with x-ndm-realm, x-ndm-challenge headers and session cookie
2. WHEN a POST request is made to /auth with credentials THEN the system SHALL return HTTP 200 for valid credentials
3. WHEN a POST request is made to /auth with invalid credentials THEN the system SHALL return HTTP 401
4. WHEN authenticated requests are made THEN the system SHALL accept requests with valid session cookies
5. THE authentication mock SHALL support the MD5-based challenge-response authentication used by Keenetic routers

### Requirement 10

**User Story:** As a developer, I want clear documentation and examples for migrating existing tests, so that I can transition from old mocks to the unified mock efficiently

#### Acceptance Criteria

1. THE system SHALL provide migration documentation that identifies all existing mock implementations
2. THE system SHALL provide code examples showing how to replace each old mock pattern with the unified mock
3. THE system SHALL document the differences in API between old mocks and the unified mock
4. THE system SHALL provide a checklist for verifying successful migration of each test file
5. THE migration documentation SHALL include before/after code examples for common test scenarios

### Requirement 11

**User Story:** As a developer, I want the mock to be easily extensible, so that I can add support for new endpoints as the application grows

#### Acceptance Criteria

1. THE mock implementation SHALL use a clear, consistent pattern for adding new endpoints
2. THE mock implementation SHALL separate endpoint handlers from state management logic
3. THE mock implementation SHALL provide helper functions for common response patterns
4. THE mock implementation SHALL document the process for adding new endpoints
5. WHEN a new endpoint is added THEN the system SHALL require minimal changes to existing code

### Requirement 12

**User Story:** As a developer, I want the mock to support the parse endpoint comprehensively, so that I can test all CLI-style commands

#### Acceptance Criteria

1. WHEN parse requests are submitted THEN the system SHALL accept an array of ParseRequest objects
2. WHEN parse requests are processed THEN the system SHALL return an array of ParseResponse objects with matching length
3. WHEN a parse command modifies state THEN the system SHALL update the mock's internal state accordingly
4. WHEN a parse command is invalid THEN the system SHALL return a response with error status and descriptive message
5. THE parse endpoint SHALL support all command types currently used in tests (interface up, route add/delete, DNS add/delete, AWG configuration)

### Requirement 13

**User Story:** As a developer, I want the mock to support running-config and system-mode endpoints, so that I can test configuration export and system information queries

#### Acceptance Criteria

1. WHEN running-config is queried THEN the system SHALL return configuration lines in the expected format
2. WHEN system-mode is queried THEN the system SHALL return active and selected mode information
3. THE running-config SHALL reflect the current state of the mock (interfaces, routes, DNS records)
4. THE system-mode SHALL support common modes (router, access point, repeater)
5. WHEN the mock state changes THEN the running-config SHALL reflect those changes

### Requirement 14

**User Story:** As a developer, I want the mock to integrate seamlessly with the existing test infrastructure, so that migration requires minimal changes to test setup

#### Acceptance Criteria

1. THE unified mock SHALL work with the existing testify suite pattern used in the codebase
2. THE unified mock SHALL provide a setup function compatible with existing SetupSuite patterns
3. THE unified mock SHALL return an httptest.Server instance compatible with existing test code
4. THE unified mock SHALL work with the existing SetupTestConfig function
5. THE unified mock SHALL support the same cleanup patterns (TearDownSuite) as existing mocks
