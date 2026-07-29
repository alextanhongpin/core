package sets_test

import (
	"fmt"
	"log"
	"sort"
	"strings"
	"testing"

	"github.com/alextanhongpin/core/types/sets"
)

// Example: User Permission Management
func ExampleSet_userPermissions() {
	fmt.Println("User Permission Management:")

	// Define permissions for different roles
	adminPerms := sets.Of("read", "write", "delete", "admin", "manage_users")
	editorPerms := sets.Of("read", "write", "edit", "publish")
	_ = sets.Of("read", "view") // viewerPerms for demonstration

	// User has multiple roles
	userPerms := sets.Union(adminPerms, editorPerms)
	fmt.Printf("User permissions: %s\n", userPerms)

	// Check specific permissions
	canDelete := userPerms.Has("delete")
	canManage := userPerms.Has("manage_users")
	fmt.Printf("Can delete: %v, Can manage users: %v\n", canDelete, canManage)

	// Find common permissions between roles
	commonPerms := sets.Intersection(adminPerms, editorPerms)
	fmt.Printf("Common permissions: %s\n", commonPerms)

	// Admin-only permissions
	adminOnlyPerms := sets.Difference(adminPerms, editorPerms)
	fmt.Printf("Admin-only permissions: %s\n", adminOnlyPerms)

	// Output:
	// User Permission Management:
	// User permissions: {admin, delete, edit, manage_users, publish, read, write}
	// Can delete: true, Can manage users: true
	// Common permissions: {read, write}
	// Admin-only permissions: {admin, delete, manage_users}
}

// Example: Tag and Category Management
func ExampleSet_tagManagement() {
	fmt.Println("Content Tag Management:")

	// Article tags
	article1Tags := sets.Of("golang", "programming", "backend", "tutorial")
	article2Tags := sets.Of("golang", "web", "frontend", "tutorial")
	article3Tags := sets.Of("python", "machine-learning", "data-science")

	// Find articles with common tags
	commonTags := sets.Intersection(article1Tags, article2Tags)
	fmt.Printf("Common tags between article 1 & 2: %s\n", commonTags)

	// All unique tags across articles
	allTags := sets.Union(sets.Union(article1Tags, article2Tags), article3Tags)
	fmt.Printf("All unique tags: %s\n", allTags)

	// Programming-related tags
	programmingTags := sets.Of("golang", "python", "programming", "backend", "frontend")

	// Check which articles are programming-related
	fmt.Printf("Article 1 programming-related: %v\n", !sets.IsDisjoint(article1Tags, programmingTags))
	fmt.Printf("Article 2 programming-related: %v\n", !sets.IsDisjoint(article2Tags, programmingTags))
	fmt.Printf("Article 3 programming-related: %v\n", !sets.IsDisjoint(article3Tags, programmingTags))

	// Output:
	// Content Tag Management:
	// Common tags between article 1 & 2: {golang, tutorial}
	// All unique tags: {backend, data-science, frontend, golang, machine-learning, programming, python, tutorial, web}
	// Article 1 programming-related: true
	// Article 2 programming-related: true
	// Article 3 programming-related: true
}

// Example: Feature Flag Management
func ExampleSet_featureFlags() {
	fmt.Println("Feature Flag Management:")

	// Define feature flags for different environments
	productionFlags := sets.Of("feature_a", "feature_b", "feature_stable")
	stagingFlags := sets.Of("feature_a", "feature_b", "feature_c", "feature_experimental")
	developmentFlags := sets.Of("feature_a", "feature_b", "feature_c", "feature_d", "feature_debug")

	// Features available in all environments
	universalFeatures := sets.Intersection(productionFlags, stagingFlags, developmentFlags)
	fmt.Printf("Universal features: %s\n", universalFeatures)

	// Development-only features
	devOnlyFeatures := sets.Difference(developmentFlags, productionFlags)
	fmt.Printf("Development-only features: %s\n", devOnlyFeatures)

	// Features that need production testing
	needsProdTesting := sets.Difference(stagingFlags, productionFlags)
	fmt.Printf("Features needing production testing: %s\n", needsProdTesting)

	// Check if staging is ready for production
	readyForProd := sets.IsSubset(stagingFlags, productionFlags)
	fmt.Printf("Staging ready for production: %v\n", readyForProd)

	// Output:
	// Feature Flag Management:
	// Universal features: {feature_a, feature_b}
	// Development-only features: {feature_c, feature_d, feature_debug}
	// Features needing production testing: {feature_c, feature_experimental}
	// Staging ready for production: false
}

// Example: Data Processing and Deduplication
func ExampleSet_dataDeduplication() {
	fmt.Println("Data Deduplication:")

	// Simulate data from different sources
	source1IDs := []int{1, 2, 3, 4, 5, 2, 3} // has duplicates
	source2IDs := []int{3, 4, 5, 6, 7, 8}
	source3IDs := []int{5, 6, 7, 8, 9, 10}

	// Convert to sets (automatically removes duplicates)
	set1 := sets.From(source1IDs)
	set2 := sets.From(source2IDs)
	set3 := sets.From(source3IDs)

	fmt.Printf("Source 1 (deduplicated): %s\n", set1)
	fmt.Printf("Source 2: %s\n", set2)
	fmt.Printf("Source 3: %s\n", set3)

	// All unique IDs across sources
	allUniqueIDs := sets.Union(sets.Union(set1, set2), set3)
	fmt.Printf("All unique IDs: %s\n", allUniqueIDs)

	// IDs present in all sources
	commonIDs := sets.Intersection(sets.Intersection(set1, set2), set3)
	fmt.Printf("IDs in all sources: %s\n", commonIDs)

	// IDs unique to each source
	unique1 := sets.Difference(set1, sets.Union(set2, set3))
	unique2 := sets.Difference(set2, sets.Union(set1, set3))
	unique3 := sets.Difference(set3, sets.Union(set1, set2))

	fmt.Printf("Unique to source 1: %s\n", unique1)
	fmt.Printf("Unique to source 2: %s\n", unique2)
	fmt.Printf("Unique to source 3: %s\n", unique3)

	// Output:
	// Data Deduplication:
	// Source 1 (deduplicated): {1, 2, 3, 4, 5}
	// Source 2: {3, 4, 5, 6, 7, 8}
	// Source 3: {5, 6, 7, 8, 9, 10}
	// All unique IDs: {1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	// IDs in all sources: {5}
	// Unique to source 1: {1, 2}
	// Unique to source 2: {}
	// Unique to source 3: {9, 10}
}

// Example: Access Control and Security Groups
func ExampleSet_accessControl() {
	fmt.Println("Access Control Management:")

	// Define security groups
	adminGroup := sets.Of("alice", "bob")
	developersGroup := sets.Of("charlie", "diana", "eve")
	qaGroup := sets.Of("frank", "grace")
	allEmployees := sets.Of("alice", "bob", "charlie", "diana", "eve", "frank", "grace", "henry")

	// Resource access permissions
	sensitiveResourceUsers := sets.Union(adminGroup, sets.Of("diana")) // senior developer
	_ = allEmployees                                                   // publicResourceUsers for demonstration

	// Check access permissions
	checkAccess := func(user string, resource string, allowedUsers *sets.Set[string]) {
		hasAccess := allowedUsers.Has(user)
		fmt.Printf("User '%s' access to %s: %v\n", user, resource, hasAccess)
	}

	checkAccess("alice", "sensitive resource", sensitiveResourceUsers)
	checkAccess("diana", "sensitive resource", sensitiveResourceUsers)
	checkAccess("charlie", "sensitive resource", sensitiveResourceUsers)

	// Find users without any group membership
	usersInGroups := sets.Union(adminGroup, developersGroup, qaGroup)
	ungroupedUsers := sets.Difference(allEmployees, usersInGroups)
	fmt.Printf("Users without group membership: %s\n", ungroupedUsers)

	// Check if all developers have access to development resources
	devResourceUsers := sets.Union(developersGroup, adminGroup) // admins have dev access too
	allDevsHaveAccess := sets.IsSubset(developersGroup, devResourceUsers)
	fmt.Printf("All developers have dev resource access: %v\n", allDevsHaveAccess)

	// Output:
	// Access Control Management:
	// User 'alice' access to sensitive resource: true
	// User 'diana' access to sensitive resource: true
	// User 'charlie' access to sensitive resource: false
	// Users without group membership: {henry}
	// All developers have dev resource access: true
}

// Example: A/B Testing and Experiment Groups
func ExampleSet_abTesting() {
	fmt.Println("A/B Testing Groups:")

	// Define experiment groups
	controlGroup := sets.Of("user1", "user3", "user5", "user7", "user9")
	treatmentGroupA := sets.Of("user2", "user4", "user6", "user8")
	treatmentGroupB := sets.Of("user10", "user11", "user12", "user13")

	// All experiment participants
	allParticipants := sets.Union(controlGroup, treatmentGroupA, treatmentGroupB)
	fmt.Printf("Total participants: %d\n", allParticipants.Len())

	// Ensure no overlap between groups (proper A/B test design)
	controlVsA := sets.IsDisjoint(controlGroup, treatmentGroupA)
	controlVsB := sets.IsDisjoint(controlGroup, treatmentGroupB)
	aVsB := sets.IsDisjoint(treatmentGroupA, treatmentGroupB)

	fmt.Printf("Groups are properly isolated: %v\n", controlVsA && controlVsB && aVsB)

	// Simulate user actions
	purchasedUsers := sets.Of("user2", "user4", "user7", "user9", "user11")

	// Calculate conversion rates by group
	controlPurchases := sets.Intersection(controlGroup, purchasedUsers)
	treatmentAPurchases := sets.Intersection(treatmentGroupA, purchasedUsers)
	treatmentBPurchases := sets.Intersection(treatmentGroupB, purchasedUsers)

	fmt.Printf("Control group conversions: %d/%d\n", controlPurchases.Len(), controlGroup.Len())
	fmt.Printf("Treatment A conversions: %d/%d\n", treatmentAPurchases.Len(), treatmentGroupA.Len())
	fmt.Printf("Treatment B conversions: %d/%d\n", treatmentBPurchases.Len(), treatmentGroupB.Len())

	// Output:
	// A/B Testing Groups:
	// Total participants: 13
	// Groups are properly isolated: true
	// Control group conversions: 2/5
	// Treatment A conversions: 2/4
	// Treatment B conversions: 1/4
}

// Example: Social Network Analysis
func ExampleSet_socialNetwork() {
	fmt.Println("Social Network Analysis:")

	// User connections (followers)
	aliceFollowers := sets.Of("bob", "charlie", "diana", "eve")
	bobFollowers := sets.Of("alice", "charlie", "frank")
	charlieFollowers := sets.Of("alice", "bob", "diana", "grace")
	dianaFollowers := sets.Of("alice", "charlie", "eve")

	// Find mutual followers
	aliceBobMutual := sets.Intersection(aliceFollowers, bobFollowers)
	fmt.Printf("Alice & Bob mutual followers: %s\n", aliceBobMutual)

	// Find influencers (users who follow each other)
	aliceCharlieMutual := sets.Intersection(aliceFollowers, charlieFollowers)
	fmt.Printf("Alice & Charlie mutual followers: %s\n", aliceCharlieMutual)

	// Users who follow Alice but not Bob
	aliceExclusive := sets.Difference(aliceFollowers, bobFollowers)
	fmt.Printf("Follow Alice but not Bob: %s\n", aliceExclusive)

	// Total unique users in the network
	allUsers := sets.Union(aliceFollowers, bobFollowers, charlieFollowers, dianaFollowers)
	fmt.Printf("Total unique users: %d - %s\n", allUsers.Len(), allUsers)

	// Find users who are followed by everyone
	followedByAll := sets.Intersection(aliceFollowers, bobFollowers, charlieFollowers, dianaFollowers)
	fmt.Printf("Followed by everyone: %s\n", followedByAll)

	// Output:
	// Social Network Analysis:
	// Alice & Bob mutual followers: {charlie}
	// Alice & Charlie mutual followers: {bob, diana}
	// Follow Alice but not Bob: {bob, diana, eve}
	// Total unique users: 7 - {alice, bob, charlie, diana, eve, frank, grace}
	// Followed by everyone: {}
}

// Example: Inventory and Stock Management
func ExampleSet_inventoryManagement() {
	fmt.Println("Inventory Management:")

	// Available products in different warehouses
	warehouse1 := sets.Of("laptop", "mouse", "keyboard", "monitor")
	warehouse2 := sets.Of("laptop", "printer", "scanner", "keyboard")
	warehouse3 := sets.Of("mouse", "monitor", "printer", "webcam")

	// Products available in all warehouses
	universalStock := sets.Intersection(warehouse1, warehouse2, warehouse3)
	fmt.Printf("Available in all warehouses: %s\n", universalStock)

	// All unique products across warehouses
	allProducts := sets.Union(warehouse1, warehouse2, warehouse3)
	fmt.Printf("All products: %s\n", allProducts)

	// Products exclusive to each warehouse
	exclusive1 := sets.Difference(warehouse1, sets.Union(warehouse2, warehouse3))
	exclusive2 := sets.Difference(warehouse2, sets.Union(warehouse1, warehouse3))
	exclusive3 := sets.Difference(warehouse3, sets.Union(warehouse1, warehouse2))

	fmt.Printf("Exclusive to warehouse 1: %s\n", exclusive1)
	fmt.Printf("Exclusive to warehouse 2: %s\n", exclusive2)
	fmt.Printf("Exclusive to warehouse 3: %s\n", exclusive3)

	// Customer order checking
	customerOrder := sets.Of("laptop", "mouse", "keyboard")

	canFulfillFrom1 := sets.IsSubset(customerOrder, warehouse1)
	canFulfillFrom2 := sets.IsSubset(customerOrder, warehouse2)
	canFulfillFrom3 := sets.IsSubset(customerOrder, warehouse3)

	fmt.Printf("Can fulfill order from warehouse 1: %v\n", canFulfillFrom1)
	fmt.Printf("Can fulfill order from warehouse 2: %v\n", canFulfillFrom2)
	fmt.Printf("Can fulfill order from warehouse 3: %v\n", canFulfillFrom3)

	// Output:
	// Inventory Management:
	// Available in all warehouses: {}
	// All products: {keyboard, laptop, monitor, mouse, printer, scanner, webcam}
	// Exclusive to warehouse 1: {}
	// Exclusive to warehouse 2: {scanner}
	// Exclusive to warehouse 3: {webcam}
	// Can fulfill order from warehouse 1: true
	// Can fulfill order from warehouse 2: false
	// Can fulfill order from warehouse 3: false
}

// Example: Skills Matching for Job Recruitment
func ExampleSet_skillsMatching() {
	fmt.Println("Skills-based Job Matching:")

	// Job requirements
	backendJobSkills := sets.Of("golang", "sql", "docker", "kubernetes", "api-design")
	frontendJobSkills := sets.Of("javascript", "react", "css", "html", "typescript")
	fullstackJobSkills := sets.Of("golang", "javascript", "react", "sql", "docker")

	// Candidate skills
	candidate1Skills := sets.Of("golang", "sql", "docker", "python")
	candidate2Skills := sets.Of("javascript", "react", "css", "html", "vue")
	candidate3Skills := sets.Of("golang", "javascript", "react", "sql", "docker", "kubernetes")

	// Calculate skill match percentages
	calculateMatch := func(candidateSkills, jobSkills *sets.Set[string]) float64 {
		requiredSkills := jobSkills.Len()
		matchedSkills := sets.Intersection(candidateSkills, jobSkills).Len()
		return float64(matchedSkills) / float64(requiredSkills) * 100
	}

	fmt.Printf("Candidate 1 matches:\n")
	fmt.Printf("  Backend: %.1f%%\n", calculateMatch(candidate1Skills, backendJobSkills))
	fmt.Printf("  Frontend: %.1f%%\n", calculateMatch(candidate1Skills, frontendJobSkills))
	fmt.Printf("  Fullstack: %.1f%%\n", calculateMatch(candidate1Skills, fullstackJobSkills))

	fmt.Printf("Candidate 2 matches:\n")
	fmt.Printf("  Backend: %.1f%%\n", calculateMatch(candidate2Skills, backendJobSkills))
	fmt.Printf("  Frontend: %.1f%%\n", calculateMatch(candidate2Skills, frontendJobSkills))
	fmt.Printf("  Fullstack: %.1f%%\n", calculateMatch(candidate2Skills, fullstackJobSkills))

	fmt.Printf("Candidate 3 matches:\n")
	fmt.Printf("  Backend: %.1f%%\n", calculateMatch(candidate3Skills, backendJobSkills))
	fmt.Printf("  Frontend: %.1f%%\n", calculateMatch(candidate3Skills, frontendJobSkills))
	fmt.Printf("  Fullstack: %.1f%%\n", calculateMatch(candidate3Skills, fullstackJobSkills))

	// Missing skills analysis
	candidate1Missing := sets.Difference(backendJobSkills, candidate1Skills)
	fmt.Printf("Candidate 1 missing skills for backend: %s\n", candidate1Missing)

	// Output:
	// Skills-based Job Matching:
	// Candidate 1 matches:
	//   Backend: 60.0%
	//   Frontend: 0.0%
	//   Fullstack: 60.0%
	// Candidate 2 matches:
	//   Backend: 0.0%
	//   Frontend: 80.0%
	//   Fullstack: 40.0%
	// Candidate 3 matches:
	//   Backend: 80.0%
	//   Frontend: 40.0%
	//   Fullstack: 100.0%
	// Candidate 1 missing skills for backend: {api-design, kubernetes}
}

// Example: Configuration and Environment Management
func ExampleSet_configManagement() {
	fmt.Println("Configuration Management:")

	// Required configurations for different environments
	devConfig := sets.Of("debug", "hot-reload", "mock-api", "test-db", "dev-cors")
	stagingConfig := sets.Of("logging", "staging-db", "ssl", "monitoring", "backup")
	prodConfig := sets.Of("logging", "prod-db", "ssl", "monitoring", "backup", "cdn", "cache")

	// Current environment configuration
	currentConfig := sets.Of("debug", "logging", "ssl", "monitoring", "test-db")

	// Check which environment this matches
	fmt.Printf("Current config: %s\n", currentConfig)

	devMatch := sets.Intersection(currentConfig, devConfig).Len()
	stagingMatch := sets.Intersection(currentConfig, stagingConfig).Len()
	prodMatch := sets.Intersection(currentConfig, prodConfig).Len()

	fmt.Printf("Dev environment match: %d/%d configs\n", devMatch, devConfig.Len())
	fmt.Printf("Staging environment match: %d/%d configs\n", stagingMatch, stagingConfig.Len())
	fmt.Printf("Production environment match: %d/%d configs\n", prodMatch, prodConfig.Len())

	// Missing configurations for production
	missingForProd := sets.Difference(prodConfig, currentConfig)
	fmt.Printf("Missing for production: %s\n", missingForProd)

	// Configurations that shouldn't be in production
	invalidForProd := sets.Difference(currentConfig, prodConfig)
	fmt.Printf("Invalid for production: %s\n", invalidForProd)

	// Output:
	// Configuration Management:
	// Current config: {debug, logging, monitoring, ssl, test-db}
	// Dev environment match: 2/5 configs
	// Staging environment match: 3/5 configs
	// Production environment match: 3/7 configs
	// Missing for production: {backup, cache, cdn, prod-db}
	// Invalid for production: {debug, test-db}
}

// Test set operations comprehensively
func TestSetOperations(t *testing.T) {
	t.Run("Basic Operations", func(t *testing.T) {
		s := sets.Of(1, 2, 3, 4, 5)

		if s.Len() != 5 {
			t.Errorf("Expected length 5, got %d", s.Len())
		}

		if !s.Has(3) {
			t.Error("Expected set to contain 3")
		}

		if s.Has(6) {
			t.Error("Expected set not to contain 6")
		}

		s.AddMany(6, 7)
		if s.Len() != 7 {
			t.Errorf("Expected length 7 after adding, got %d", s.Len())
		}

		s.RemoveMany(1, 2)
		if s.Len() != 5 {
			t.Errorf("Expected length 5 after deleting, got %d", s.Len())
		}
	})

	t.Run("Set Operations", func(t *testing.T) {
		a := sets.Of(1, 2, 3, 4)
		b := sets.Of(3, 4, 5, 6)

		// Union
		union := sets.Union(a, b)
		expected := []int{1, 2, 3, 4, 5, 6}
		if !sliceEqual(union.All(), expected) {
			t.Errorf("Union failed: got %v, want %v", union.All(), expected)
		}

		// Intersection
		intersect := sets.Intersection(a, b)
		expected = []int{3, 4}
		if !sliceEqual(intersect.All(), expected) {
			t.Errorf("Intersection failed: got %v, want %v", intersect.All(), expected)
		}

		// Difference
		diff := sets.Difference(a, b)
		expected = []int{1, 2}
		if !sliceEqual(diff.All(), expected) {
			t.Errorf("Difference failed: got %v, want %v", diff.All(), expected)
		}

		// Symmetric Difference
		symDiff := sets.SymmetricDifference(a, b)
		expected = []int{1, 2, 5, 6}
		if !sliceEqual(symDiff.All(), expected) {
			t.Errorf("Symmetric difference failed: got %v, want %v", symDiff.All(), expected)
		}
	})

	t.Run("Subset/Superset Operations", func(t *testing.T) {
		a := sets.Of(1, 2)
		b := sets.Of(1, 2, 3, 4)
		c := sets.Of(1, 2)

		if !sets.IsSubset(a, b) {
			t.Error("Expected a to be subset of b")
		}

		if !sets.IsSuperset(b, a) {
			t.Error("Expected b to be superset of a")
		}

		if !sets.Equal(a, c) {
			t.Error("Expected a to equal c")
		}

		if !sets.IsProperSubset(a, b) {
			t.Error("Expected a to be proper subset of b")
		}

		if sets.IsProperSubset(a, c) {
			t.Error("Expected a not to be proper subset of c")
		}
	})

	t.Run("Predicate Operations", func(t *testing.T) {
		s := sets.Of(2, 4, 6, 8, 10)

		// All even
		allEven := s.Every(func(x int) bool { return x%2 == 0 })
		if !allEven {
			t.Error("Expected all elements to be even")
		}

		// Any greater than 8
		anyGreaterThan8 := s.Any(func(x int) bool { return x > 8 })
		if !anyGreaterThan8 {
			t.Error("Expected some elements to be greater than 8")
		}

		// Filter greater than 5
		filtered := s.Filter(func(x int) bool { return x > 5 })
		expected := []int{6, 8, 10}
		if !sliceEqual(filtered.All(), expected) {
			t.Errorf("Filter failed: got %v, want %v", filtered.All(), expected)
		}
	})
}

// Test string representation
func TestSetString(t *testing.T) {
	// Empty set
	empty := sets.Of[int]()
	if empty.String() != "{}" {
		t.Errorf("Empty set string: got %s, want {}", empty.String())
	}

	// Single element
	single := sets.Of(42)
	if single.String() != "{42}" {
		t.Errorf("Single element set string: got %s, want {42}", single.String())
	}

	// Multiple elements
	multiple := sets.Of(3, 1, 2)
	if multiple.String() != "{1, 2, 3}" {
		t.Errorf("Multiple element set string: got %s, want {1, 2, 3}", multiple.String())
	}
}

// Benchmark set operations
func BenchmarkSetOperations(b *testing.B) {
	// Create test sets
	size := 1000
	a := sets.Of[int]()
	bSlice := make([]int, size)
	for i := range size {
		a.Add(i)
		bSlice[i] = i + size/2 // 50% overlap
	}
	setB := sets.From(bSlice)

	b.Run("Add", func(b *testing.B) {
		for i := range b.N {
			s := sets.Of[int]()
			s.Add(i)
		}
	})

	b.Run("Has", func(b *testing.B) {
		for i := range b.N {
			_ = a.Has(i % size)
		}
	})

	b.Run("Union", func(b *testing.B) {
		for b.Loop() {
			_ = sets.Union(a, setB)
		}
	})

	b.Run("Intersect", func(b *testing.B) {
		for b.Loop() {
			_ = sets.Intersection(a, setB)
		}
	})

	b.Run("Difference", func(b *testing.B) {
		for b.Loop() {
			_ = sets.Difference(a, setB)
		}
	})
}

// Helper function to compare slices
func sliceEqual(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}

	// Sort both slices for comparison
	aCopy := make([]int, len(a))
	bCopy := make([]int, len(b))
	copy(aCopy, a)
	copy(bCopy, b)
	sort.Ints(aCopy)
	sort.Ints(bCopy)

	for i := range aCopy {
		if aCopy[i] != bCopy[i] {
			return false
		}
	}
	return true
}

// Example of working with string sets and complex filtering
func ExampleSet_complexFiltering() {
	fmt.Println("Complex filtering example:")

	// Create a set of words
	words := sets.Of("apple", "banana", "cherry", "date", "elderberry", "fig", "grape")

	// Filter words with more than 5 characters
	longWords := words.Filter(func(word string) bool {
		return len(word) > 5
	})
	fmt.Printf("Long words: %s\n", longWords)

	// Check if any word starts with 'a'
	startsWithA := words.Any(func(word string) bool {
		return strings.HasPrefix(word, "a")
	})
	fmt.Printf("Any word starts with 'a': %v\n", startsWithA)

	// Check if all words are lowercase
	allLowercase := words.Every(func(word string) bool {
		return strings.ToLower(word) == word
	})
	fmt.Printf("All words lowercase: %v\n", allLowercase)

	// Count characters in all words
	totalChars := 0
	words.Range(func(word string) {
		totalChars += len(word)
	})
	fmt.Printf("Total characters: %d\n", totalChars)

	// Output:
	// Complex filtering example:
	// Long words: {banana, cherry, elderberry}
	// Any word starts with 'a': true
	// All words lowercase: true
	// Total characters: 39
}

func init() {
	// Don't log during tests
	log.SetOutput(nil)
}
