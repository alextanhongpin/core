# DataLoader

A Go implementation of Facebook's DataLoader pattern for efficient batch loading and caching of data. Perfect for GraphQL servers and any application that needs to solve the N+1 query problem.

## Features

- **Batch Loading**: Groups multiple individual requests into batches to reduce database/API calls
- **Caching**: Built-in caching to avoid redundant requests within the same context
- **Concurrency Safe**: Thread-safe operations for concurrent access
- **Timeout Control**: Configurable batch timeout for optimal performance
- **Error Handling**: Graceful error handling with per-key error reporting
- **Generic Support**: Full Go generics support for type safety

## Installation

```bash
go get github.com/alextanhongpin/core/sync/dataloader
```

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "database/sql"
    
    "github.com/alextanhongpin/core/sync/dataloader"
)

func main() {
    ctx := context.Background()
    
    // Create a dataloader for loading users by ID
    userLoader := dataloader.New(ctx, &dataloader.Options[int, User]{
        BatchFn: func(ctx context.Context, userIDs []int) (map[int]User, error) {
            // This function will be called with batched IDs
            return loadUsersFromDB(ctx, userIDs)
        },
        BatchTimeout: 16 * time.Millisecond,
        BatchMaxKeys: 100,
    })
    defer userLoader.Stop()
    
    // Load individual users (these will be batched automatically)
    user1, err := userLoader.Load(1)
    if err != nil {
        log.Fatal(err)
    }
    
    user2, err := userLoader.Load(2)
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("User 1: %+v\n", user1)
    fmt.Printf("User 2: %+v\n", user2)
}

type User struct {
    ID   int
    Name string
}

func loadUsersFromDB(ctx context.Context, userIDs []int) (map[int]User, error) {
    // This would typically be a database query like:
    // SELECT id, name FROM users WHERE id IN (?, ?, ...)
    
    users := make(map[int]User)
    for _, id := range userIDs {
        users[id] = User{ID: id, Name: fmt.Sprintf("User %d", id)}
    }
    return users, nil
}
```

## API Reference

### Options

```go
type Options[K comparable, V any] struct {
    BatchFn        func(ctx context.Context, keys []K) (map[K]V, error)
    BatchMaxKeys   int           // Maximum keys per batch (default: 1000)
    BatchTimeout   time.Duration // Batch timeout (default: 16ms)
    BatchQueueSize int           // Queue size for batching
    Cache          cache[K, V]   // Custom cache implementation
}
```

### Methods

#### `Load(key K) (V, error)`
Loads a single value by key. If the key is not found, returns `ErrNoResult`.

#### `LoadMany(keys []K) ([]Result[V], error)`
Loads multiple values by keys. Returns a slice of results in the same order as the input keys.

#### `Prime(key K, value V)`
Primes the cache with a key-value pair to avoid future loads.

#### `Clear(key K)`
Removes a key from the cache.

#### `Stop()`
Stops the dataloader and cleans up resources.

## Real-World Examples

### GraphQL Server with User/Post Relationship

```go
package main

import (
    "context"
    "fmt"
    "time"
    
    "github.com/alextanhongpin/core/sync/dataloader"
)

// User represents a user in our system
type User struct {
    ID   int
    Name string
}

// Post represents a blog post
type Post struct {
    ID       int
    Title    string
    AuthorID int
    Author   *User // Will be loaded via dataloader
}

// DataLoaders holds all our dataloaders
type DataLoaders struct {
    UserLoader *dataloader.DataLoader[int, User]
    PostLoader *dataloader.DataLoader[int, Post]
}

// NewDataLoaders creates a new set of dataloaders
func NewDataLoaders(ctx context.Context, db *sql.DB) *DataLoaders {
    return &DataLoaders{
        UserLoader: dataloader.New(ctx, &dataloader.Options[int, User]{
            BatchFn: func(ctx context.Context, userIDs []int) (map[int]User, error) {
                return loadUsersFromDB(ctx, db, userIDs)
            },
            BatchTimeout: 16 * time.Millisecond,
            BatchMaxKeys: 100,
        }),
        PostLoader: dataloader.New(ctx, &dataloader.Options[int, Post]{
            BatchFn: func(ctx context.Context, postIDs []int) (map[int]Post, error) {
                return loadPostsFromDB(ctx, db, postIDs)
            },
            BatchTimeout: 16 * time.Millisecond,
            BatchMaxKeys: 100,
        }),
    }
}

// LoadUserPosts loads posts for a user and efficiently loads post authors
func (dl *DataLoaders) LoadUserPosts(ctx context.Context, userID int) ([]Post, error) {
    // Get user's posts
    posts, err := getUserPosts(ctx, userID)
    if err != nil {
        return nil, err
    }
    
    // Load all authors in a single batch
    authorIDs := make([]int, len(posts))
    for i, post := range posts {
        authorIDs[i] = post.AuthorID
    }
    
    authors, err := dl.UserLoader.LoadMany(authorIDs)
    if err != nil {
        return nil, err
    }
    
    // Attach authors to posts
    for i, author := range authors {
        if author.Err == nil {
            posts[i].Author = &author.Data
        }
    }
    
    return posts, nil
}

func loadUsersFromDB(ctx context.Context, db *sql.DB, userIDs []int) (map[int]User, error) {
    // Convert IDs to interface{} for query
    args := make([]interface{}, len(userIDs))
    placeholders := make([]string, len(userIDs))
    for i, id := range userIDs {
        args[i] = id
        placeholders[i] = "?"
    }
    
    query := fmt.Sprintf("SELECT id, name FROM users WHERE id IN (%s)", 
        strings.Join(placeholders, ","))
    
    rows, err := db.QueryContext(ctx, query, args...)
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    
    users := make(map[int]User)
    for rows.Next() {
        var user User
        if err := rows.Scan(&user.ID, &user.Name); err != nil {
            return nil, err
        }
        users[user.ID] = user
    }
    
    return users, nil
}

func loadPostsFromDB(ctx context.Context, db *sql.DB, postIDs []int) (map[int]Post, error) {
    // Similar implementation for posts
    args := make([]interface{}, len(postIDs))
    placeholders := make([]string, len(postIDs))
    for i, id := range postIDs {
        args[i] = id
        placeholders[i] = "?"
    }
    
    query := fmt.Sprintf("SELECT id, title, author_id FROM posts WHERE id IN (%s)", 
        strings.Join(placeholders, ","))
    
    rows, err := db.QueryContext(ctx, query, args...)
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    
    posts := make(map[int]Post)
    for rows.Next() {
        var post Post
        if err := rows.Scan(&post.ID, &post.Title, &post.AuthorID); err != nil {
            return nil, err
        }
        posts[post.ID] = post
    }
    
    return posts, nil
}
```

### REST API with Caching

```go
package main

import (
    "context"
    "encoding/json"
    "fmt"
    "net/http"
    "time"
    
    "github.com/alextanhongpin/core/sync/dataloader"
)

type APIClient struct {
    client *http.Client
    loader *dataloader.DataLoader[string, APIResponse]
}

type APIResponse struct {
    ID   string `json:"id"`
    Data string `json:"data"`
}

func NewAPIClient(ctx context.Context) *APIClient {
    client := &APIClient{
        client: &http.Client{Timeout: 30 * time.Second},
    }
    
    client.loader = dataloader.New(ctx, &dataloader.Options[string, APIResponse]{
        BatchFn: client.batchLoad,
        BatchTimeout: 50 * time.Millisecond,
        BatchMaxKeys: 10,
    })
    
    return client
}

func (c *APIClient) batchLoad(ctx context.Context, ids []string) (map[string]APIResponse, error) {
    // Make a single API call for multiple IDs
    url := fmt.Sprintf("https://api.example.com/batch?ids=%s", strings.Join(ids, ","))
    
    req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
    if err != nil {
        return nil, err
    }
    
    resp, err := c.client.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    
    var responses []APIResponse
    if err := json.NewDecoder(resp.Body).Decode(&responses); err != nil {
        return nil, err
    }
    
    result := make(map[string]APIResponse)
    for _, response := range responses {
        result[response.ID] = response
    }
    
    return result, nil
}

func (c *APIClient) Get(ctx context.Context, id string) (APIResponse, error) {
    return c.loader.Load(id)
}

func (c *APIClient) GetMany(ctx context.Context, ids []string) ([]APIResponse, error) {
    results, err := c.loader.LoadMany(ids)
    if err != nil {
        return nil, err
    }
    
    responses := make([]APIResponse, 0, len(results))
    for _, result := range results {
        if result.Err == nil {
            responses = append(responses, result.Data)
        }
    }
    
    return responses, nil
}
```

## Error Handling

The dataloader provides detailed error information:

```go
results, err := loader.LoadMany([]int{1, 2, 999})
if err != nil {
    log.Fatal(err)
}

for i, result := range results {
    if result.Err != nil {
        if errors.Is(result.Err, dataloader.ErrNoResult) {
            fmt.Printf("Key %d not found\n", i)
        } else {
            fmt.Printf("Error loading key %d: %v\n", i, result.Err)
        }
    } else {
        fmt.Printf("Key %d: %+v\n", i, result.Data)
    }
}
```

## Performance Considerations

1. **Batch Timeout**: Lower timeouts reduce latency but may result in smaller batches
2. **Batch Size**: Larger batches are more efficient but use more memory
3. **Cache Management**: Consider cache TTL and size limits for long-running applications
4. **Context Propagation**: Always use context for proper request lifecycle management

## Testing

```go
func TestUserLoader(t *testing.T) {
    ctx := context.Background()
    
    // Mock database
    mockDB := map[int]User{
        1: {ID: 1, Name: "Alice"},
        2: {ID: 2, Name: "Bob"},
    }
    
    loader := dataloader.New(ctx, &dataloader.Options[int, User]{
        BatchFn: func(ctx context.Context, ids []int) (map[int]User, error) {
            result := make(map[int]User)
            for _, id := range ids {
                if user, exists := mockDB[id]; exists {
                    result[id] = user
                }
            }
            return result, nil
        },
    })
    defer loader.Stop()
    
    // Test single load
    user, err := loader.Load(1)
    assert.NoError(t, err)
    assert.Equal(t, "Alice", user.Name)
    
    // Test load many
    results, err := loader.LoadMany([]int{1, 2, 999})
    assert.NoError(t, err)
    assert.Len(t, results, 3)
    assert.NoError(t, results[0].Err)
    assert.NoError(t, results[1].Err)
    assert.Error(t, results[2].Err)
}
```

## Best Practices

1. **Use Short Batch Timeouts**: 16ms is often optimal for web applications
2. **Implement Proper Error Handling**: Always check for `ErrNoResult` and handle gracefully
3. **Prime Cache When Possible**: Use `Prime()` to avoid redundant loads
4. **Context Propagation**: Pass context through all operations for proper cancellation
5. **Resource Cleanup**: Always call `Stop()` when done with the dataloader
6. **Monitor Performance**: Track batch sizes and hit rates in production

## License

MIT License. See [LICENSE](../../LICENSE) for details.
