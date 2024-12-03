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

## Functional & Non-Functional Requirements
### Functional Requirements
1. Endpoint to register a new user
2. Endpoint to login
3. Endpoint to get dating profiles
    - It should accept a list of excluded profiles ID
    - It should not return a same profile which has been swiped by the user on that day
    - It should not return a profile which has been matched with the user
4. Endpoint to swipe dating profile
    - It should accept a profile ID and action (like or pass)
    - It should return a outcome of the swipe
    - If user swipe like more than 10 times, user shouldn't be able to swipe like anymore

### Non-Functional Requirements
1. User can likes and pass other users
2. User can likes up to 10 times for free, if user want to increase the limit, user need to buy the premium subscription
3. User will only see dating profile that they haven't swiped yet
4. When like a profile which like back, user will notified immediately
5. When user pass a profile which likes the user back, user will notified immediately that he missed the chance to match with that profile
6. When a user swipe like more than 10 times, user shouldn't be able to perform any like action but can still pass other users

## Stacks
- Golang
    
    The ease of use of goroutine which allows for us to utilize the distributed processing for future development, which is critical for the scalability of the service, in distributed system which needs to handle a large number of requests per seconds.

    Golang is a simple language which similar to C, which is developer friendly and easy to understand.
- Echo Router
    
    Based on benchmark from from techempower JSON Serialization benchmark ehco is one of the fastest compared to chi and golang stardard library, a hight throughput is critical for an application such as tinder, which needs to handle a lot of RPS

    Echo is also compatible with the standard go net/http library, which gives flexibility for developers

    https://www.techempower.com/benchmarks/#hw=ph&test=json&section=data-r22&l=zijocf-cn3
- PostgreSQL
    
    PostgreSQL offer extensive support for SQL and it's a battle-tested database which is used by many big companies, which ensures that the database part of the service is reliable and scalable.

    Postgres Extension such as 
        - pg_partman for partitioning the table for better performance and easier backup.
        - pg_mq for message queueing which can be used to handle the event when user liked or passed a profile. in the early phase of the development
        - pg_cron for scheduling the task to run periodically.

    furthermore supabase involvement in the Package manager [database.dev](database.dev) which will alow tons of community support for the database ecosystem.
- Redis
    
    Redis is used to cache the most frequently accessed data from the database, which will improve the performance of the service.

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
### Match Usecase
* Get Dating Profiles
```mermaid
sequenceDiagram
    participant User
    participant MatchUseCase
    participant MatchRepo
    participant DB

    User->>MatchUseCase: GetDatingProfiles(ctx, userID, excludeProfiles, limit)
    MatchUseCase->>MatchRepo: GetTodayLikedProfilesIDs(ctx, userID)
    MatchRepo-->>MatchUseCase: Return likedProfiles or error
    alt Error Occurred
        MatchUseCase-->>User: Return error
    else No Error
        MatchUseCase->>MatchRepo: GetMatchedProfilesIDs(ctx, userID)
        MatchRepo-->>MatchUseCase: Return matchedProfiles or error
        alt Error Occurred
            MatchUseCase-->>User: Return error
        else No Error
            MatchUseCase->>MatchUseCase: Append likedProfiles and matchedProfiles to excludeProfiles
            MatchUseCase->>MatchRepo: GetDatingProfilesIDs(ctx, userID, excludeProfiles, limit)
            MatchRepo-->>MatchUseCase: Return profiles or error
            alt Error Occurred
                MatchUseCase-->>User: Return error
            else No Error
                MatchUseCase-->>User: Return profiles
            end
        end
    end
```

* Swipe Dating Profile
```mermaid
sequenceDiagram
    participant User
    participant MatchUseCase
    participant MatchRepo
    participant UserRepo

    User->>MatchUseCase: SwipeDatingProfile(ctx, userID, likedToUserID, action)
    MatchUseCase->>MatchRepo: GetTodayLikesCount(ctx, userID)
    MatchRepo-->>MatchUseCase: Return likesCount or error
    alt Error Occurred
        MatchUseCase-->>User: Return error
    else No Error
        MatchUseCase->>UserRepo: GetUserByID(ctx, likedToUserID)
        UserRepo-->>MatchUseCase: Return user or error
        alt Error Occurred
            MatchUseCase-->>User: Return error
        else No Error
            alt Likes Limit Reached
                MatchUseCase-->>User: OutcomeLimitReached
            else Likes Limit Not Reached
                MatchUseCase->>MatchRepo: CreateSwipe(ctx, userID, likedToUserID, action)
                MatchRepo-->>MatchUseCase: Return Outcome or error
                alt Error Occurred
                    MatchUseCase-->>User: Return error
                else No Error
                    alt Outcome is Match
                        MatchUseCase-->>User: OutcomeMatch
                    else Outcome is Not Match
                        MatchUseCase-->>User: Return Outcome
                    end
                end
            end
        end
    end
```

## Test Cases