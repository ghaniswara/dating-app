# dating-app
A dating app backend service using golang, utilize Echo for routing, GORM for ORM, PostgreSQL as the database, and Redis for caching.

Repository URL : https://github.com/ghaniswara/dating-app

## Server Structure
- /internal/config : Configuration Loader for the Server
- /internal/routes : Routes for the Server
  - /v1/auth : Authentication Routes
  - /v1/match : Match Routes
- /internal/usecase
  - /auth : Authentication Usecases
  - /match : Match Usecases
- /internal/middleware : Middleware for the Server
- /internal/repository : Repositories for the Server
- /internal/entity : which consist of following entities
   - repository
   - request
   - response
- /pkg/http_util : HTTP Utility for the Server
- /pkg/jwt : JWT Utility for the Server
- /pkg/path : Utility for searching path used by the Config Loader & Test Helper
- /test/auth : Authentication Test
- /test/helper : Test Helper
- /test/match : Match Test

## Instruction to Run the Service
1. Clone the repository
2. Install the dependencies using `go mod tidy`
3. Setup the environment variables using `cp .env.example .env`
4. Install [golang-migrate](https://github.com/golang-migrate/migrate) by following the instruction on the website
4. Run migration using [golang-migrate](https://github.com/golang-migrate/migrate) 
    ```
    `$ migrate -source ./migrations/ -database postgres://localhost:<YOUR-PORT>/database up 2`
    ```
4. Run the server using `go run . dev`

## ER Diagram
```mermaid
erDiagram
    USERS {
        BIGSERIAL id PK
        VARCHAR name
        VARCHAR email
        VARCHAR username
        VARCHAR password
        BOOLEAN is_premium
        TIMESTAMP created_at
        TIMESTAMP updated_at
    }

    SWIPE_TRANSACTIONS {
        SERIAL id PK
        BIGINT user_id FK
        BIGINT to_id FK
        DATE date
        SMALLINT action
        TIMESTAMP timestamp
        BOOLEAN is_matched
    }

    USERS ||--o{ SWIPE_TRANSACTIONS : "makes"
    USERS ||--o{ SWIPE_TRANSACTIONS : "receives"
```

## Sequence Diagram

### Match Repository
* Create Swipe Transaction
```mermaid
sequenceDiagram
    participant User
    participant MatchRepo
    participant DB
    participant Redis

    User->>MatchRepo: CreateSwipe(ctx, userID, likedToUserID, action)
    MatchRepo->>DB: Check if liked profile exists (likedToUserID)
    DB-->>MatchRepo: Return likedProfileRes
    alt Profile Not Found
        MatchRepo-->>User: OutcomeNotFound
    else Profile Found
        alt Action is Like or SuperLike
            MatchRepo->>Redis: Increment liked count cache
            MatchRepo->>Redis: Append liked profiles cache
            MatchRepo->>DB: Check if both profiles like each other
            DB-->>MatchRepo: Return resPair
            alt Pair Found
                MatchRepo->>DB: Create SwipeTransaction
                DB-->>MatchRepo: Return res
                MatchRepo->>DB: Update isMatched for pair
                DB-->>MatchRepo: Return update result
                MatchRepo->>Redis: Append match profiles cache
                MatchRepo-->>User: OutcomeMatch
            else Pair Not Found
                MatchRepo->>DB: Create SwipeTransaction
                DB-->>MatchRepo: Return res
                MatchRepo-->>User: OutcomeNoLike
            end
        else Action is Pass
            MatchRepo-->>User: OutcomeMissed
        end
    end
```

* Get Matched Profiles IDs
```mermaid
sequenceDiagram
    participant User
    participant MatchRepo
    participant DB
    participant Redis

    User->>MatchRepo: GetMatchedProfilesIDs(ctx, userID)
    MatchRepo->>Redis: Check for matched profiles (profilesKey)
    Redis-->>MatchRepo: Return profiles or redis.Nil
    alt Profiles Found
        MatchRepo-->>User: Return profiles
    else No Profiles Found
        MatchRepo->>DB: Query SwipeTransaction for matched profiles
        DB-->>MatchRepo: Return profiles
        MatchRepo->>Redis: Add matched profiles to Redis
        Redis-->>MatchRepo: Confirm addition
        MatchRepo->>Redis: Set expiration for profilesKey
        MatchRepo-->>User: Return profiles
    end
```