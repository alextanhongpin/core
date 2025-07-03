# States

The `states` package provides powerful utilities for managing sequential workflows, state machines, and conditional logic. It's designed for applications that need to track progress through ordered steps, validate state transitions, or evaluate complex conditional scenarios.

## Features

- **Sequential Workflows**: Track progress through ordered steps with validation
- **State Machines**: Generic state transition management with listeners and validation
- **Conditional Logic**: Rich set of boolean predicates for complex condition evaluation
- **Type Safety**: Full generic support for custom state types
- **Progress Tracking**: Monitor completion status and next required steps
- **Audit Support**: State change listeners for logging and monitoring

## Quick Start

```go
package main

import (
    "fmt"
    "github.com/alextanhongpin/core/types/states"
)

func main() {
    // Sequential workflow example
    var emailVerified, profileComplete bool
    
    registration := states.NewSequence(
        states.NewStepFunc("email", func() bool { return emailVerified }),
        states.NewStepFunc("profile", func() bool { return profileComplete }),
    )
    
    fmt.Println("Status:", registration.Status()) // idle
    
    emailVerified = true
    fmt.Println("Status:", registration.Status()) // pending
    
    profileComplete = true
    fmt.Println("Status:", registration.Status()) // success
}
```

## API Reference

### Sequential Workflows

Sequential workflows ensure steps are completed in order before proceeding to the next step.

```go
// Create a sequence of steps
sequence := states.NewSequence(
    states.NewStep("step1", true),                              // Static condition
    states.NewStepFunc("step2", func() bool { return ready }), // Dynamic condition
)

// Check status
status := sequence.Status() // idle, pending, success, or failed

// Get next required step
if next, ok := sequence.Next(); ok {
    fmt.Println("Next step:", next.Name())
}

// Validation and progress
isValid := sequence.IsValid()
completed := sequence.CompletedSteps()
total := sequence.TotalSteps()
```

### State Machines

State machines manage transitions between defined states with validation and listeners.

```go
type OrderStatus string
const (
    Pending   OrderStatus = "pending"
    Paid      OrderStatus = "paid" 
    Shipped   OrderStatus = "shipped"
    Delivered OrderStatus = "delivered"
)

// Create state machine
orderSM := states.NewStateMachine(Pending,
    states.NewTransition("pay", Pending, Paid),
    states.NewTransition("ship", Paid, Shipped),
    states.NewTransition("deliver", Shipped, Delivered),
)

// Add state change listener
orderSM.AddListener(func(from, to OrderStatus, transitionName string) {
    fmt.Printf("Order %s: %s -> %s\n", transitionName, from, to)
})

// Execute transitions
err := orderSM.Execute("pay")     // pending -> paid
err = orderSM.Execute("ship")     // paid -> shipped
err = orderSM.Execute("deliver")  // shipped -> delivered

// Check valid transitions
canPay := orderSM.CanTransitionTo(Paid)
validStates := orderSM.GetValidStates()
```

### Conditional Logic

Rich boolean operations for complex condition evaluation.

```go
// Exact count requirements
exactly2 := states.ExactlyN(2, true, true, false, false) // true

// Range requirements  
atLeast2 := states.AtLeastN(2, true, true, false)        // true
atMost3 := states.AtMostN(3, true, true, true, false)    // true

// Special patterns
allOrNone := states.AllOrNone(true, true, true)          // true
majority := states.Majority(true, true, false, false)    // true
xor := states.XOR(true, false, false)                    // true (exactly one)

// Function-based predicates
isReady := func() bool { return systemReady }
hasPermission := func() bool { return userHasPermission }

canProceed := states.AllFunc(isReady, hasPermission)
```

## Real-World Examples

### User Registration Workflow

```go
type RegistrationData struct {
    EmailProvided   bool
    EmailVerified   bool
    ProfileComplete bool
    TermsAccepted   bool
}

func NewRegistrationWorkflow(data *RegistrationData) *states.Sequence {
    return states.NewSequence(
        states.NewStepFunc("email_provided", func() bool { 
            return data.EmailProvided 
        }),
        states.NewStepFunc("email_verified", func() bool { 
            return data.EmailVerified 
        }),
        states.NewStepFunc("profile_complete", func() bool { 
            return data.ProfileComplete 
        }),
        states.NewStepFunc("terms_accepted", func() bool { 
            return data.TermsAccepted 
        }),
    )
}

// Usage
regData := &RegistrationData{}
workflow := NewRegistrationWorkflow(regData)

// Guide user through registration
for workflow.Status() != states.Success {
    if next, ok := workflow.Next(); ok {
        fmt.Printf("Please complete: %s\n", next.Name())
        
        // Simulate user action based on step
        switch next.Name() {
        case "email_provided":
            regData.EmailProvided = true
        case "email_verified":
            regData.EmailVerified = true
        case "profile_complete":
            regData.ProfileComplete = true
        case "terms_accepted":
            regData.TermsAccepted = true
        }
    }
}
```

### Order Processing System

```go
type OrderStatus string

const (
    OrderCreated   OrderStatus = "created"
    OrderPaid      OrderStatus = "paid"
    OrderShipped   OrderStatus = "shipped"
    OrderDelivered OrderStatus = "delivered"
    OrderCancelled OrderStatus = "cancelled"
    OrderReturned  OrderStatus = "returned"
)

type Order struct {
    ID     string
    Status OrderStatus
    SM     *states.StateMachine[OrderStatus]
}

func NewOrder(id string) *Order {
    sm := states.NewStateMachine(OrderCreated,
        // Payment transitions
        states.NewTransition("pay", OrderCreated, OrderPaid),
        
        // Fulfillment transitions  
        states.NewTransition("ship", OrderPaid, OrderShipped),
        states.NewTransition("deliver", OrderShipped, OrderDelivered),
        
        // Cancellation transitions
        states.NewTransition("cancel", OrderCreated, OrderCancelled),
        states.NewTransition("cancel", OrderPaid, OrderCancelled),
        
        // Return transitions
        states.NewTransition("return", OrderDelivered, OrderReturned),
    )
    
    order := &Order{ID: id, SM: sm}
    
    // Add audit logging
    sm.AddListener(func(from, to OrderStatus, transitionName string) {
        order.Status = to
        fmt.Printf("[ORDER %s] %s: %s -> %s\n", 
            order.ID, transitionName, from, to)
    })
    
    return order
}

func (o *Order) Pay() error {
    return o.SM.TransitionWithFunc(OrderPaid, "pay", func() error {
        // Process payment
        return processPayment(o.ID)
    })
}

func (o *Order) Ship() error {
    return o.SM.Execute("ship")
}

func (o *Order) Cancel() error {
    return o.SM.Execute("cancel")
}
```

### Document Approval Workflow

```go
type ApprovalLevel struct {
    Name     string
    Required bool
    Approved bool
}

type DocumentApproval struct {
    Document string
    Levels   []ApprovalLevel
    Sequence *states.Sequence
}

func NewDocumentApproval(document string, levels []ApprovalLevel) *DocumentApproval {
    var steps []states.Step
    
    for _, level := range levels {
        if level.Required {
            steps = append(steps, states.NewStepFunc(level.Name, func() bool {
                return level.Approved
            }))
        }
    }
    
    return &DocumentApproval{
        Document: document,
        Levels:   levels,
        Sequence: states.NewSequence(steps...),
    }
}

func (da *DocumentApproval) Approve(levelName string) error {
    for i := range da.Levels {
        if da.Levels[i].Name == levelName {
            da.Levels[i].Approved = true
            break
        }
    }
    
    if da.Sequence.Status() == states.Success {
        fmt.Printf("Document %s fully approved!\n", da.Document)
    }
    
    return nil
}

func (da *DocumentApproval) GetNextApprover() (string, bool) {
    if next, ok := da.Sequence.Next(); ok {
        return next.Name(), true
    }
    return "", false
}
```

### Feature Flag Validation

```go
type FeatureConfig struct {
    ExperimentalUI       bool
    ExperimentalBackend  bool
    ExperimentalAnalytics bool
    BetaFeatures        bool
    DebugMode           bool
}

func (fc *FeatureConfig) Validate() error {
    // Experimental features must be all on or all off
    if !states.AllOrNone(fc.ExperimentalUI, fc.ExperimentalBackend, fc.ExperimentalAnalytics) {
        return errors.New("experimental features must be consistently enabled/disabled")
    }
    
    // Debug mode requires at least one experimental feature in development
    if fc.DebugMode {
        hasExperimental := states.AnyFunc(
            func() bool { return fc.ExperimentalUI },
            func() bool { return fc.ExperimentalBackend },
            func() bool { return fc.ExperimentalAnalytics },
        )
        if !hasExperimental {
            return errors.New("debug mode requires experimental features")
        }
    }
    
    return nil
}

func (fc *FeatureConfig) CanEnableBeta() bool {
    // Beta features require majority of experimental features
    return states.Majority(fc.ExperimentalUI, fc.ExperimentalBackend, fc.ExperimentalAnalytics)
}
```

### Multi-Factor Authentication

```go
type MFAConfig struct {
    PasswordRequired bool
    SMSRequired      bool
    TOTPRequired     bool
    BiometricEnabled bool
}

type MFASession struct {
    Config   MFAConfig
    Progress *states.Sequence
}

func NewMFASession(config MFAConfig) *MFASession {
    var steps []states.Step
    
    // Always require password
    steps = append(steps, states.NewStep("password", false))
    
    // Add additional factors based on config
    if config.SMSRequired {
        steps = append(steps, states.NewStep("sms", false))
    }
    if config.TOTPRequired {
        steps = append(steps, states.NewStep("totp", false))
    }
    if config.BiometricEnabled {
        steps = append(steps, states.NewStep("biometric", false))
    }
    
    return &MFASession{
        Config:   config,
        Progress: states.NewSequence(steps...),
    }
}

func (mfa *MFASession) CompleteStep(stepName string) error {
    // Find and update the step
    for _, step := range mfa.Progress.AllSteps() {
        if step.Name() == stepName {
            // In a real implementation, you'd update the underlying condition
            break
        }
    }
    
    if mfa.Progress.Status() == states.Success {
        fmt.Println("MFA authentication complete!")
    }
    
    return nil
}

func (mfa *MFASession) GetRequiredFactors() int {
    return mfa.Progress.TotalSteps()
}

func (mfa *MFASession) IsComplete() bool {
    return mfa.Progress.Status() == states.Success
}
```

### Voting and Consensus Systems

```go
type Vote struct {
    VoterID string
    Choice  bool
    Cast    bool
}

type VotingSession struct {
    Proposal    string
    Votes       []Vote
    QuorumSize  int
    MajorityReq bool
}

func (vs *VotingSession) HasQuorum() bool {
    castVotes := vs.getCastVotes()
    return len(castVotes) >= vs.QuorumSize
}

func (vs *VotingSession) HasPassed() bool {
    if !vs.HasQuorum() {
        return false
    }
    
    castVotes := vs.getCastVotes()
    choices := make([]bool, len(castVotes))
    for i, vote := range castVotes {
        choices[i] = vote.Choice
    }
    
    if vs.MajorityReq {
        return states.Majority(choices...)
    }
    
    // Unanimous requirement
    return states.AllFunc(func() bool {
        for _, choice := range choices {
            if !choice {
                return false
            }
        }
        return true
    })
}

func (vs *VotingSession) getCastVotes() []Vote {
    var cast []Vote
    for _, vote := range vs.Votes {
        if vote.Cast {
            cast = append(cast, vote)
        }
    }
    return cast
}

func (vs *VotingSession) GetStatus() string {
    if !vs.HasQuorum() {
        return fmt.Sprintf("Waiting for quorum (%d/%d votes)", 
            len(vs.getCastVotes()), vs.QuorumSize)
    }
    
    if vs.HasPassed() {
        return "Proposal passed"
    }
    
    return "Proposal failed"
}
```

## Best Practices

1. **Step Design**: Keep step conditions simple and side-effect free
2. **State Validation**: Always validate state transitions before performing actions
3. **Listener Usage**: Use state change listeners for audit logging and notifications
4. **Error Handling**: Use `TransitionWithFunc` for operations that can fail
5. **Immutability**: Consider making state machines immutable after creation
6. **Testing**: Test both valid and invalid state sequences thoroughly

## Performance Characteristics

- **Sequence Validation**: O(n) where n is the number of steps
- **State Transitions**: O(n) where n is the number of defined transitions
- **Conditional Logic**: O(n) where n is the number of conditions
- **Memory Usage**: O(n) for transitions and listeners

## Thread Safety

The states package is not thread-safe by default. For concurrent access, wrap operations with appropriate synchronization:

```go
import "sync"

type SafeStateMachine[T comparable] struct {
    sm *states.StateMachine[T]
    mu sync.RWMutex
}

func (ssm *SafeStateMachine[T]) Execute(name string) error {
    ssm.mu.Lock()
    defer ssm.mu.Unlock()
    return ssm.sm.Execute(name)
}

func (ssm *SafeStateMachine[T]) State() T {
    ssm.mu.RLock()
    defer ssm.mu.RUnlock()
    return ssm.sm.State()
}
```

## Common Patterns

### Workflow Builder Pattern

```go
type WorkflowBuilder struct {
    steps []states.Step
}

func NewWorkflowBuilder() *WorkflowBuilder {
    return &WorkflowBuilder{}
}

func (wb *WorkflowBuilder) AddStep(name string, condition func() bool) *WorkflowBuilder {
    wb.steps = append(wb.steps, states.NewStepFunc(name, condition))
    return wb
}

func (wb *WorkflowBuilder) Build() *states.Sequence {
    return states.NewSequence(wb.steps...)
}

// Usage
workflow := NewWorkflowBuilder().
    AddStep("validate", func() bool { return dataValid }).
    AddStep("process", func() bool { return processComplete }).
    AddStep("notify", func() bool { return notificationSent }).
    Build()
```

### State Machine Factory

```go
func NewOrderStateMachine() *states.StateMachine[OrderStatus] {
    return states.NewStateMachine(OrderCreated,
        states.NewTransition("pay", OrderCreated, OrderPaid),
        states.NewTransition("ship", OrderPaid, OrderShipped),
        states.NewTransition("deliver", OrderShipped, OrderDelivered),
        states.NewTransition("cancel", OrderCreated, OrderCancelled),
        states.NewTransition("return", OrderDelivered, OrderReturned),
    )
}
```
