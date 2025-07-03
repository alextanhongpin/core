package states_test

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/alextanhongpin/core/types/states"
	"github.com/stretchr/testify/assert"
)

// Example: User Registration Workflow
func ExampleSequence_userRegistration() {
	// Track user registration progress
	var (
		emailProvided   = false
		emailVerified   = false
		profileComplete = false
		termsAccepted   = false
	)

	// Create registration sequence
	registration := states.NewSequence(
		states.NewStepFunc("email_provided", func() bool { return emailProvided }),
		states.NewStepFunc("email_verified", func() bool { return emailVerified }),
		states.NewStepFunc("profile_complete", func() bool { return profileComplete }),
		states.NewStepFunc("terms_accepted", func() bool { return termsAccepted }),
	)

	// Initial state
	fmt.Printf("Initial status: %s\n", registration.Status())

	// User provides email
	emailProvided = true
	fmt.Printf("After email: %s\n", registration.Status())

	// User verifies email
	emailVerified = true
	fmt.Printf("After verification: %s\n", registration.Status())

	// Complete profile
	profileComplete = true
	fmt.Printf("After profile: %s\n", registration.Status())

	// Accept terms
	termsAccepted = true
	fmt.Printf("Final status: %s\n", registration.Status())

	// Output:
	// Initial status: idle
	// After email: pending
	// After verification: pending
	// After profile: pending
	// Final status: success
}

// Example: Order Processing State Machine
type OrderStatus string

const (
	OrderPending   OrderStatus = "pending"
	OrderPaid      OrderStatus = "paid"
	OrderShipped   OrderStatus = "shipped"
	OrderDelivered OrderStatus = "delivered"
	OrderCancelled OrderStatus = "cancelled"
	OrderReturned  OrderStatus = "returned"
)

func ExampleStateMachine_orderProcessing() {
	// Define valid order transitions
	orderSM := states.NewStateMachine(OrderPending,
		states.NewTransition("pay", OrderPending, OrderPaid),
		states.NewTransition("ship", OrderPaid, OrderShipped),
		states.NewTransition("deliver", OrderShipped, OrderDelivered),
		states.NewTransition("cancel", OrderPending, OrderCancelled),
		states.NewTransition("cancel", OrderPaid, OrderCancelled),
		states.NewTransition("return", OrderDelivered, OrderReturned),
	)

	// Add state change listener
	orderSM.AddListener(func(from, to OrderStatus, transitionName string) {
		fmt.Printf("Order %s: %s -> %s\n", transitionName, from, to)
	})

	// Process order
	fmt.Printf("Initial state: %s\n", orderSM.State())

	// Pay for order
	orderSM.Execute("pay")

	// Ship order
	orderSM.Execute("ship")

	// Deliver order
	orderSM.Execute("deliver")

	fmt.Printf("Final state: %s\n", orderSM.State())

	// Output:
	// Initial state: pending
	// Order pay: pending -> paid
	// Order ship: paid -> shipped
	// Order deliver: shipped -> delivered
	// Final state: delivered
}

// Example: Document Approval Workflow
func ExampleSequence_documentApproval() {
	// Document approval steps
	var (
		managerApproved = false
		hrApproved      = false
		cfoApproved     = false
	)

	approval := states.NewSequence(
		states.NewStepFunc("manager_approval", func() bool { return managerApproved }),
		states.NewStepFunc("hr_approval", func() bool { return hrApproved }),
		states.NewStepFunc("cfo_approval", func() bool { return cfoApproved }),
	)

	// Check next required step
	if next, ok := approval.Next(); ok {
		fmt.Printf("Next step required: %s\n", next.Name())
	}

	// Manager approves
	managerApproved = true
	if next, ok := approval.Next(); ok {
		fmt.Printf("Next step required: %s\n", next.Name())
	}

	// HR approves
	hrApproved = true
	if next, ok := approval.Next(); ok {
		fmt.Printf("Next step required: %s\n", next.Name())
	}

	// CFO approves
	cfoApproved = true
	fmt.Printf("Approval complete: %s\n", approval.Status())

	// Output:
	// Next step required: manager_approval
	// Next step required: hr_approval
	// Next step required: cfo_approval
	// Approval complete: success
}

// Example: Feature Flag Validation
func ExampleAllOrNone_featureFlags() {
	// Feature flags that must be all enabled or all disabled
	var (
		experimentalUI        = true
		experimentalBackend   = true
		experimentalAnalytics = false
	)

	// Check if experimental features are consistently configured
	consistent := states.AllOrNone(experimentalUI, experimentalBackend, experimentalAnalytics)
	fmt.Printf("Feature flags consistent: %v\n", consistent)

	// Fix configuration
	experimentalAnalytics = true
	consistent = states.AllOrNone(experimentalUI, experimentalBackend, experimentalAnalytics)
	fmt.Printf("After fix: %v\n", consistent)

	// Output:
	// Feature flags consistent: false
	// After fix: true
}

// Example: Voting System
func ExampleMajority_voting() {
	votes := []bool{true, true, false, true, false}

	// Check if majority voted yes
	majorityYes := states.Majority(votes...)
	fmt.Printf("Majority voted yes: %v\n", majorityYes)

	// Check specific vote counts
	exactlyThree := states.ExactlyN(3, votes...)
	atLeastTwo := states.AtLeastN(2, votes...)
	atMostFour := states.AtMostN(4, votes...)

	fmt.Printf("Exactly 3 yes votes: %v\n", exactlyThree)
	fmt.Printf("At least 2 yes votes: %v\n", atLeastTwo)
	fmt.Printf("At most 4 yes votes: %v\n", atMostFour)

	// Output:
	// Majority voted yes: true
	// Exactly 3 yes votes: true
	// At least 2 yes votes: true
	// At most 4 yes votes: true
}

// Example: Game State Machine
type GameState string

const (
	GameMenu    GameState = "menu"
	GamePlaying GameState = "playing"
	GamePaused  GameState = "paused"
	GameOver    GameState = "game_over"
)

func ExampleStateMachine_gameState() {
	// Define game state transitions
	game := states.NewStateMachine(GameMenu,
		states.NewTransition("start", GameMenu, GamePlaying),
		states.NewTransition("pause", GamePlaying, GamePaused),
		states.NewTransition("resume", GamePaused, GamePlaying),
		states.NewTransition("quit", GamePlaying, GameMenu),
		states.NewTransition("quit", GamePaused, GameMenu),
		states.NewTransition("die", GamePlaying, GameOver),
		states.NewTransition("restart", GameOver, GamePlaying),
		states.NewTransition("menu", GameOver, GameMenu),
	)

	fmt.Printf("Game state: %s\n", game.State())

	// Start game
	game.Execute("start")
	fmt.Printf("Game state: %s\n", game.State())

	// Pause game
	game.Execute("pause")
	fmt.Printf("Game state: %s\n", game.State())

	// Resume game
	game.Execute("resume")
	fmt.Printf("Game state: %s\n", game.State())

	// Game over
	game.Execute("die")
	fmt.Printf("Game state: %s\n", game.State())

	// Output:
	// Game state: menu
	// Game state: playing
	// Game state: paused
	// Game state: playing
	// Game state: game_over
}

// Example: Multi-Factor Authentication
func ExampleSequence_mfaAuthentication() {
	// MFA steps
	var (
		passwordCorrect = false
		smsVerified     = false
		totpVerified    = false
	)

	mfa := states.NewSequence(
		states.NewStepFunc("password", func() bool { return passwordCorrect }),
		states.NewStepFunc("sms_verification", func() bool { return smsVerified }),
		states.NewStepFunc("totp_verification", func() bool { return totpVerified }),
	)

	// User enters password
	passwordCorrect = true
	fmt.Printf("After password: %s (%d/%d)\n",
		mfa.Status(), mfa.CompletedSteps(), mfa.TotalSteps())

	// User verifies SMS
	smsVerified = true
	fmt.Printf("After SMS: %s (%d/%d)\n",
		mfa.Status(), mfa.CompletedSteps(), mfa.TotalSteps())

	// User verifies TOTP
	totpVerified = true
	fmt.Printf("After TOTP: %s (%d/%d)\n",
		mfa.Status(), mfa.CompletedSteps(), mfa.TotalSteps())

	// Output:
	// After password: pending (1/3)
	// After SMS: pending (2/3)
	// After TOTP: success (3/3)
}

// Example: Form Validation with Predicates
func ExampleExactlyN_formValidation() {
	// Form validation predicates
	hasName := func() bool { return true }
	hasEmail := func() bool { return true }
	hasPhone := func() bool { return false }
	hasAddress := func() bool { return false }

	// Require exactly 2 contact methods
	exactlyTwo := states.ExactlyNFunc(2, hasName, hasEmail, hasPhone, hasAddress)
	fmt.Printf("Exactly 2 fields filled: %v\n", exactlyTwo)

	// Check if at least one contact method provided
	hasContact := states.AnyFunc(hasEmail, hasPhone)
	fmt.Printf("Has contact method: %v\n", hasContact)

	// Output:
	// Exactly 2 fields filled: true
	// Has contact method: true
}

// Example: Transaction Processing with State Machine
type TransactionState string

const (
	TxCreated   TransactionState = "created"
	TxPending   TransactionState = "pending"
	TxCompleted TransactionState = "completed"
	TxFailed    TransactionState = "failed"
	TxRefunded  TransactionState = "refunded"
)

func ExampleStateMachine_transactionProcessing() {
	// Create transaction state machine
	tx := states.NewStateMachine(TxCreated,
		states.NewTransition("submit", TxCreated, TxPending),
		states.NewTransition("complete", TxPending, TxCompleted),
		states.NewTransition("fail", TxPending, TxFailed),
		states.NewTransition("refund", TxCompleted, TxRefunded),
	)

	// Add audit logging
	tx.AddListener(func(from, to TransactionState, transitionName string) {
		fmt.Printf("[AUDIT] Transaction %s: %s -> %s at %s\n",
			transitionName, from, to, time.Now().Format("15:04:05"))
	})

	// Process transaction
	tx.Execute("submit")

	// Simulate processing with error handling
	err := tx.TransitionWithFunc(TxCompleted, "complete", func() error {
		// Simulate payment processing
		fmt.Println("Processing payment...")
		return nil // Success
	})

	if err != nil {
		fmt.Printf("Transaction failed: %v\n", err)
	}

	// Output:
	// [AUDIT] Transaction submit: created -> pending at 15:04:05
	// Processing payment...
	// [AUDIT] Transaction complete: pending -> completed at 15:04:05
}

// Unit Tests
func TestSequenceValidation(t *testing.T) {
	t.Run("valid sequence progression", func(t *testing.T) {
		assert := assert.New(t)

		var step1, step2, step3 bool
		seq := states.NewSequence(
			states.NewStepFunc("step1", func() bool { return step1 }),
			states.NewStepFunc("step2", func() bool { return step2 }),
			states.NewStepFunc("step3", func() bool { return step3 }),
		)

		// Initial state
		assert.Equal(states.Idle, seq.Status())
		assert.True(seq.IsValid())

		// Complete step 1
		step1 = true
		assert.Equal(states.Pending, seq.Status())
		assert.True(seq.IsValid())

		// Complete step 2
		step2 = true
		assert.Equal(states.Pending, seq.Status())
		assert.True(seq.IsValid())

		// Complete step 3
		step3 = true
		assert.Equal(states.Success, seq.Status())
		assert.True(seq.IsValid())
	})

	t.Run("invalid sequence (skipped step)", func(t *testing.T) {
		assert := assert.New(t)

		var step1, step2, step3 bool
		seq := states.NewSequence(
			states.NewStepFunc("step1", func() bool { return step1 }),
			states.NewStepFunc("step2", func() bool { return step2 }),
			states.NewStepFunc("step3", func() bool { return step3 }),
		)

		// Skip step 1, complete step 2
		step2 = true
		assert.Equal(states.Failed, seq.Status())
		assert.False(seq.IsValid())
	})
}

func TestStateMachineTransitions(t *testing.T) {
	t.Run("valid transitions", func(t *testing.T) {
		assert := assert.New(t)

		sm := states.NewStateMachine("A",
			states.NewTransition("to_b", "A", "B"),
			states.NewTransition("to_c", "B", "C"),
		)

		assert.Equal("A", sm.State())

		err := sm.Execute("to_b")
		assert.NoError(err)
		assert.Equal("B", sm.State())

		err = sm.Execute("to_c")
		assert.NoError(err)
		assert.Equal("C", sm.State())
	})

	t.Run("invalid transition", func(t *testing.T) {
		assert := assert.New(t)

		sm := states.NewStateMachine("A",
			states.NewTransition("to_b", "A", "B"),
		)

		err := sm.TransitionTo("C", "invalid")
		assert.Error(err)
		assert.Equal("A", sm.State()) // State unchanged
	})

	t.Run("transition with function", func(t *testing.T) {
		assert := assert.New(t)

		sm := states.NewStateMachine("A",
			states.NewTransition("to_b", "A", "B"),
		)

		// Successful function
		err := sm.TransitionWithFunc("B", "to_b", func() error {
			return nil
		})
		assert.NoError(err)
		assert.Equal("B", sm.State())

		// Failed function
		sm = states.NewStateMachine("A",
			states.NewTransition("to_b", "A", "B"),
		)

		err = sm.TransitionWithFunc("B", "to_b", func() error {
			return errors.New("function failed")
		})
		assert.Error(err)
		assert.Equal("A", sm.State()) // State unchanged
	})
}

func TestConditionalLogic(t *testing.T) {
	t.Run("ExactlyN", func(t *testing.T) {
		assert := assert.New(t)

		assert.True(states.ExactlyN(2, true, true, false, false))
		assert.False(states.ExactlyN(2, true, false, false, false))
		assert.False(states.ExactlyN(2, true, true, true, false))
	})

	t.Run("AllOrNone", func(t *testing.T) {
		assert := assert.New(t)

		assert.True(states.AllOrNone(true, true, true))
		assert.True(states.AllOrNone(false, false, false))
		assert.False(states.AllOrNone(true, false, true))
	})

	t.Run("Majority", func(t *testing.T) {
		assert := assert.New(t)

		assert.True(states.Majority(true, true, false))
		assert.False(states.Majority(true, false, false))
		assert.True(states.Majority(true, true, true, false, false))
	})

	t.Run("AtLeastN", func(t *testing.T) {
		assert := assert.New(t)

		assert.True(states.AtLeastN(2, true, true, false))
		assert.True(states.AtLeastN(2, true, true, true))
		assert.False(states.AtLeastN(2, true, false, false))
	})

	t.Run("AtMostN", func(t *testing.T) {
		assert := assert.New(t)

		assert.True(states.AtMostN(2, true, true, false))
		assert.True(states.AtMostN(2, true, false, false))
		assert.False(states.AtMostN(2, true, true, true))
	})
}

// Benchmarks
func BenchmarkSequenceValidation(b *testing.B) {
	var step1, step2, step3 = true, true, false
	seq := states.NewSequence(
		states.NewStepFunc("step1", func() bool { return step1 }),
		states.NewStepFunc("step2", func() bool { return step2 }),
		states.NewStepFunc("step3", func() bool { return step3 }),
	)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = seq.IsValid()
	}
}

func BenchmarkStateMachineTransition(b *testing.B) {
	sm := states.NewStateMachine("A",
		states.NewTransition("to_b", "A", "B"),
		states.NewTransition("to_a", "B", "A"),
	)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if i%2 == 0 {
			sm.Execute("to_b")
		} else {
			sm.Execute("to_a")
		}
	}
}
