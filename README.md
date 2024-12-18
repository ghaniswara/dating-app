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

## Running the Test
For the test we're using Ory/Dockertest which allows us to run integration test with dockerized database
Simply run `go test ./test/*` which will run all the test cases
If an error eccountered due to port collision, you need to run the test separately for the Auth and Match Cases this is due to TestMain in both test cases is running on the same port

- Auth Test
    ```
    `$ go test ./test/auth`
    ```
- Match Test
    ```
    `$ go test ./test/match`
    ```

     >⚠️ Will currently fail for the Match Test, due to `TestNoSameProfile` is implemented without parallel test in mind, which will cause data to be inconsistent,


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

* GetTodayLikedProfilesIDs
```mermaid
sequenceDiagram
    participant MatchRepo as matchRepo
    participant Redis
    participant Database

    matchRepo->>Redis: SMembers(profilesKey)
    alt Profiles found in Redis
        Redis-->>matchRepo: return profiles
        matchRepo-->>User: return profiles
    else Profiles not found (redis.Nil)
        Redis-->>matchRepo: return redis.Nil
        matchRepo->>Database: getLikedProfilesIDs(userID, now)
        alt Database query fails
            Database-->>matchRepo: return error
            matchRepo-->>User: return error
        else Database query succeeds
            Database-->>matchRepo: return profiles
            matchRepo->>Redis: SAdd(profilesKey, profiles)
            matchRepo->>Redis: Expire(profilesKey, TTL)
            matchRepo-->>User: return profiles
        end
    end
``` 

* GetTodayLikesCount
```mermaid
sequenceDiagram
    participant MatchRepo as m
    participant Redis
    participant Database

    m->>Redis: Get(countKey)
    alt Count found in Redis
        Redis-->>m: return count
        m-->>User: return count
    else Count not found (redis.Nil)
        Redis-->>m: return redis.Nil
        m->>Database: getLikesCount(userID, now)
        alt Database query fails
            Database-->>m: return error
            m-->>User: return error
        else Database query succeeds
            Database-->>m: return count
            m->>Redis: Set(countKey, count, TTL)
            m-->>User: return count
        end
    end
```

### Match Usecase
* Get Dating Profiles
```mermaid
sequenceDiagram
    participant User
    participant MatchUseCase
    participant MatchRepo

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
        MatchUseCase->>UserRepo: GetUserByID(ctx, userID)
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

### Auth Usecase
* Signup User
```mermaid
sequenceDiagram
    participant User
    participant AuthUseCase as authUseCase
    participant UserRepo as userRepo

    User->>AuthUseCase: SignupUser(authData)
    AuthUseCase->>AuthUseCase: Generate hashedPassword
    alt Password generation fails
        AuthUseCase-->>User: return error
    else Password generation succeeds
        AuthUseCase->>UserRepo: CreateUser(user)
        UserRepo-->>AuthUseCase: return created user
        AuthUseCase-->>User: return created user
    end
```

* Signin User
```mermaid
sequenceDiagram
    participant User
    participant AuthUseCase as authUseCase
    participant UserRepo as userRepo
    participant JWT

    User->>AuthUseCase: SignIn(email, username, password)
    AuthUseCase->>UserRepo: GetUserByUnameOrEmail(email, username)
    alt User not found
        UserRepo-->>AuthUseCase: return error
        AuthUseCase-->>User: return error
    else User found
        UserRepo-->>AuthUseCase: return user
        AuthUseCase->>AuthUseCase: Compare password
        alt Password mismatch
            AuthUseCase-->>User: return error
        else Password match
            AuthUseCase->>JWT: CreateToken(user.ID, user.Username)
            alt Token creation fails
                JWT-->>AuthUseCase: return error
                AuthUseCase-->>User: return error
            else Token creation succeeds
                JWT-->>AuthUseCase: return token
                AuthUseCase-->>User: return token
            end
        end
    end
```
## Test Cases

### Auth Case
1. Should Success on valid sign up form
2. Should Return error on invalid sign up form
    - Email is empty
    - Email is invalid
    - Username is empty
    - Password is empty
3. Should Success on valid sign in form
4. Should Return error on invalid sign in form
    - Email/Username is empty
    - Email is invalid
    - Password is empty

### Match Usecase

1. Should Success on get dating profiles
2. Should not return same profile which has been swiped by the user on that day
3. Should not return a profile which has been matched with the user
4. Should Success on swipe dating profile
    - Return OutcomeMatch when user swipe like and the profile likes the user back
    - Return OutcomeNoLike when user swipe like and the profile doesn't like the user back
    - Return OutcomeMissed when user swipe pass
5. Should Return OutcomeLimitReached when non premium user swipe like more than 10 times 
6. Should not be able to like nonexistent user

