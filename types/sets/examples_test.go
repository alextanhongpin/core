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
	adminPerms := sets.New("read", "write", "delete", "admin", "manage_users")
	editorPerms := sets.New("read", "write", "edit", "publish")
	_ = sets.New("read", "view") // viewerPerms for demonstration

	// User has multiple roles
	userPerms := adminPerms.Union(editorPerms)
	fmt.Printf("User permissions: %s\n", userPerms)

	// Check specific permissions
	canDelete := userPerms.Has("delete")
	canManage := userPerms.Has("manage_users")
	fmt.Printf("Can delete: %v, Can manage users: %v\n", canDelete, canManage)

	// Find common permissions between roles
	commonPerms := adminPerms.Intersect(editorPerms)
	fmt.Printf("Common permissions: %s\n", commonPerms)

	// Admin-only permissions
	adminOnlyPerms := adminPerms.Difference(editorPerms)
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
	article1Tags := sets.New("golang", "programming", "backend", "tutorial")
	article2Tags := sets.New("golang", "web", "frontend", "tutorial")
	article3Tags := sets.New("python", "machine-learning", "data-science")

	// Find articles with common tags
	commonTags := article1Tags.Intersect(article2Tags)
	fmt.Printf("Common tags between article 1 & 2: %s\n", commonTags)

	// All unique tags across articles
	allTags := article1Tags.Union(article2Tags).Union(article3Tags)
	fmt.Printf("All unique tags: %s\n", allTags)

	// Programming-related tags
	programmingTags := sets.New("golang", "python", "programming", "backend", "frontend")

	// Check which articles are programming-related
	fmt.Printf("Article 1 programming-related: %v\n", !article1Tags.IsDisjoint(programmingTags))
	fmt.Printf("Article 2 programming-related: %v\n", !article2Tags.IsDisjoint(programmingTags))
	fmt.Printf("Article 3 programming-related: %v\n", !article3Tags.IsDisjoint(programmingTags))

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
	productionFlags := sets.New("feature_a", "feature_b", "feature_stable")
	stagingFlags := sets.New("feature_a", "feature_b", "feature_c", "feature_experimental")
	developmentFlags := sets.New("feature_a", "feature_b", "feature_c", "feature_d", "feature_debug")

	// Features available in all environments
	universalFeatures := productionFlags.Intersect(stagingFlags).Intersect(developmentFlags)
	fmt.Printf("Universal features: %s\n", universalFeatures)

	// Development-only features
	devOnlyFeatures := developmentFlags.Difference(productionFlags)
	fmt.Printf("Development-only features: %s\n", devOnlyFeatures)

	// Features that need production testing
	needsProdTesting := stagingFlags.Difference(productionFlags)
	fmt.Printf("Features needing production testing: %s\n", needsProdTesting)

	// Check if staging is ready for production
	readyForProd := stagingFlags.IsSubset(productionFlags)
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
	set1 := sets.FromSlice(source1IDs)
	set2 := sets.FromSlice(source2IDs)
	set3 := sets.FromSlice(source3IDs)

	fmt.Printf("Source 1 (deduplicated): %s\n", set1)
	fmt.Printf("Source 2: %s\n", set2)
	fmt.Printf("Source 3: %s\n", set3)

	// All unique IDs across sources
	allUniqueIDs := set1.Union(set2).Union(set3)
	fmt.Printf("All unique IDs: %s\n", allUniqueIDs)

	// IDs present in all sources
	commonIDs := set1.Intersect(set2).Intersect(set3)
	fmt.Printf("IDs in all sources: %s\n", commonIDs)

	// IDs unique to each source
	unique1 := set1.Difference(set2.Union(set3))
	unique2 := set2.Difference(set1.Union(set3))
	unique3 := set3.Difference(set1.Union(set2))

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
	adminGroup := sets.New("alice", "bob")
	developersGroup := sets.New("charlie", "diana", "eve")
	qaGroup := sets.New("frank", "grace")
	allEmployees := sets.New("alice", "bob", "charlie", "diana", "eve", "frank", "grace", "henry")

	// Resource access permissions
	sensitiveResourceUsers := adminGroup.Union(sets.New("diana")) // senior developer
	_ = allEmployees                                              // publicResourceUsers for demonstration

	// Check access permissions
	checkAccess := func(user string, resource string, allowedUsers *sets.Set[string]) {
		hasAccess := allowedUsers.Has(user)
		fmt.Printf("User '%s' access to %s: %v\n", user, resource, hasAccess)
	}

	checkAccess("alice", "sensitive resource", sensitiveResourceUsers)
	checkAccess("diana", "sensitive resource", sensitiveResourceUsers)
	checkAccess("charlie", "sensitive resource", sensitiveResourceUsers)

	// Find users without any group membership
	usersInGroups := adminGroup.Union(developersGroup).Union(qaGroup)
	ungroupedUsers := allEmployees.Difference(usersInGroups)
	fmt.Printf("Users without group membership: %s\n", ungroupedUsers)

	// Check if all developers have access to development resources
	devResourceUsers := developersGroup.Union(adminGroup) // admins have dev access too
	allDevsHaveAccess := developersGroup.IsSubset(devResourceUsers)
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
	controlGroup := sets.New("user1", "user3", "user5", "user7", "user9")
	treatmentGroupA := sets.New("user2", "user4", "user6", "user8")
	treatmentGroupB := sets.New("user10", "user11", "user12", "user13")

	// All experiment participants
	allParticipants := controlGroup.Union(treatmentGroupA).Union(treatmentGroupB)
	fmt.Printf("Total participants: %d\n", allParticipants.Len())

	// Ensure no overlap between groups (proper A/B test design)
	controlVsA := controlGroup.IsDisjoint(treatmentGroupA)
	controlVsB := controlGroup.IsDisjoint(treatmentGroupB)
	aVsB := treatmentGroupA.IsDisjoint(treatmentGroupB)

	fmt.Printf("Groups are properly isolated: %v\n", controlVsA && controlVsB && aVsB)

	// Simulate user actions
	purchasedUsers := sets.New("user2", "user4", "user7", "user9", "user11")

	// Calculate conversion rates by group
	controlPurchases := controlGroup.Intersect(purchasedUsers)
	treatmentAPurchases := treatmentGroupA.Intersect(purchasedUsers)
	treatmentBPurchases := treatmentGroupB.Intersect(purchasedUsers)

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
	aliceFollowers := sets.New("bob", "charlie", "diana", "eve")
	bobFollowers := sets.New("alice", "charlie", "frank")
	charlieFollowers := sets.New("alice", "bob", "diana", "grace")
	dianaFollowers := sets.New("alice", "charlie", "eve")

	// Find mutual followers
	aliceBobMutual := aliceFollowers.Intersect(bobFollowers)
	fmt.Printf("Alice & Bob mutual followers: %s\n", aliceBobMutual)

	// Find influencers (users who follow each other)
	aliceCharlieMutual := aliceFollowers.Intersect(charlieFollowers)
	fmt.Printf("Alice & Charlie mutual followers: %s\n", aliceCharlieMutual)

	// Users who follow Alice but not Bob
	aliceExclusive := aliceFollowers.Difference(bobFollowers)
	fmt.Printf("Follow Alice but not Bob: %s\n", aliceExclusive)

	// Total unique users in the network
	allUsers := aliceFollowers.Union(bobFollowers).Union(charlieFollowers).Union(dianaFollowers)
	fmt.Printf("Total unique users: %d - %s\n", allUsers.Len(), allUsers)

	// Find users who are followed by everyone
	followedByAll := aliceFollowers.Intersect(bobFollowers).Intersect(charlieFollowers).Intersect(dianaFollowers)
	fmt.Printf("Followed by everyone: %s\n", followedByAll)

	// Output:
	// Social Network Analysis:
	// Alice & Bob mutual followers: {charlie}
	// Alice & Charlie mutual followers: {alice, bob, diana}
	// Follow Alice but not Bob: {diana, eve}
	// Total unique users: 7 - {alice, bob, charlie, diana, eve, frank, grace}
	// Followed by everyone: {}
}

// Example: Inventory and Stock Management
func ExampleSet_inventoryManagement() {
	fmt.Println("Inventory Management:")

	// Available products in different warehouses
	warehouse1 := sets.New("laptop", "mouse", "keyboard", "monitor")
	warehouse2 := sets.New("laptop", "printer", "scanner", "keyboard")
	warehouse3 := sets.New("mouse", "monitor", "printer", "webcam")

	// Products available in all warehouses
	universalStock := warehouse1.Intersect(warehouse2).Intersect(warehouse3)
	fmt.Printf("Available in all warehouses: %s\n", universalStock)

	// All unique products across warehouses
	allProducts := warehouse1.Union(warehouse2).Union(warehouse3)
	fmt.Printf("All products: %s\n", allProducts)

	// Products exclusive to each warehouse
	exclusive1 := warehouse1.Difference(warehouse2.Union(warehouse3))
	exclusive2 := warehouse2.Difference(warehouse1.Union(warehouse3))
	exclusive3 := warehouse3.Difference(warehouse1.Union(warehouse2))

	fmt.Printf("Exclusive to warehouse 1: %s\n", exclusive1)
	fmt.Printf("Exclusive to warehouse 2: %s\n", exclusive2)
	fmt.Printf("Exclusive to warehouse 3: %s\n", exclusive3)

	// Customer order checking
	customerOrder := sets.New("laptop", "mouse", "keyboard")

	canFulfillFrom1 := customerOrder.IsSubset(warehouse1)
	canFulfillFrom2 := customerOrder.IsSubset(warehouse2)
	canFulfillFrom3 := customerOrder.IsSubset(warehouse3)

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
	backendJobSkills := sets.New("golang", "sql", "docker", "kubernetes", "api-design")
	frontendJobSkills := sets.New("javascript", "react", "css", "html", "typescript")
	fullstackJobSkills := sets.New("golang", "javascript", "react", "sql", "docker")

	// Candidate skills
	candidate1Skills := sets.New("golang", "sql", "docker", "python")
	candidate2Skills := sets.New("javascript", "react", "css", "html", "vue")
	candidate3Skills := sets.New("golang", "javascript", "react", "sql", "docker", "kubernetes")

	// Calculate skill match percentages
	calculateMatch := func(candidateSkills, jobSkills *sets.Set[string]) float64 {
		requiredSkills := jobSkills.Len()
		matchedSkills := candidateSkills.Intersect(jobSkills).Len()
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
	candidate1Missing := backendJobSkills.Difference(candidate1Skills)
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
	//   Fullstack: 20.0%
	// Candidate 3 matches:
	//   Backend: 80.0%
	//   Frontend: 20.0%
	//   Fullstack: 100.0%
	// Candidate 1 missing skills for backend: {api-design, kubernetes}
}

// Example: Configuration and Environment Management
func ExampleSet_configManagement() {
	fmt.Println("Configuration Management:")

	// Required configurations for different environments
	devConfig := sets.New("debug", "hot-reload", "mock-api", "test-db", "dev-cors")
	stagingConfig := sets.New("logging", "staging-db", "ssl", "monitoring", "backup")
	prodConfig := sets.New("logging", "prod-db", "ssl", "monitoring", "backup", "cdn", "cache")

	// Current environment configuration
	currentConfig := sets.New("debug", "logging", "ssl", "monitoring", "test-db")

	// Check which environment this matches
	fmt.Printf("Current config: %s\n", currentConfig)

	devMatch := currentConfig.Intersect(devConfig).Len()
	stagingMatch := currentConfig.Intersect(stagingConfig).Len()
	prodMatch := currentConfig.Intersect(prodConfig).Len()

	fmt.Printf("Dev environment match: %d/%d configs\n", devMatch, devConfig.Len())
	fmt.Printf("Staging environment match: %d/%d configs\n", stagingMatch, stagingConfig.Len())
	fmt.Printf("Production environment match: %d/%d configs\n", prodMatch, prodConfig.Len())

	// Missing configurations for production
	missingForProd := prodConfig.Difference(currentConfig)
	fmt.Printf("Missing for production: %s\n", missingForProd)

	// Configurations that shouldn't be in production
	invalidForProd := currentConfig.Difference(prodConfig)
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
		s := sets.New(1, 2, 3, 4, 5)

		if s.Len() != 5 {
			t.Errorf("Expected length 5, got %d", s.Len())
		}

		if !s.Has(3) {
			t.Error("Expected set to contain 3")
		}

		if s.Has(6) {
			t.Error("Expected set not to contain 6")
		}

		s.Add(6, 7)
		if s.Len() != 7 {
			t.Errorf("Expected length 7 after adding, got %d", s.Len())
		}

		s.Delete(1, 2)
		if s.Len() != 5 {
			t.Errorf("Expected length 5 after deleting, got %d", s.Len())
		}
	})

	t.Run("Set Operations", func(t *testing.T) {
		a := sets.New(1, 2, 3, 4)
		b := sets.New(3, 4, 5, 6)

		// Union
		union := a.Union(b)
		expected := []int{1, 2, 3, 4, 5, 6}
		if !sliceEqual(union.All(), expected) {
			t.Errorf("Union failed: got %v, want %v", union.All(), expected)
		}

		// Intersection
		intersect := a.Intersect(b)
		expected = []int{3, 4}
		if !sliceEqual(intersect.All(), expected) {
			t.Errorf("Intersection failed: got %v, want %v", intersect.All(), expected)
		}

		// Difference
		diff := a.Difference(b)
		expected = []int{1, 2}
		if !sliceEqual(diff.All(), expected) {
			t.Errorf("Difference failed: got %v, want %v", diff.All(), expected)
		}

		// Symmetric Difference
		symDiff := a.SymmetricDifference(b)
		expected = []int{1, 2, 5, 6}
		if !sliceEqual(symDiff.All(), expected) {
			t.Errorf("Symmetric difference failed: got %v, want %v", symDiff.All(), expected)
		}
	})

	t.Run("Subset/Superset Operations", func(t *testing.T) {
		a := sets.New(1, 2)
		b := sets.New(1, 2, 3, 4)
		c := sets.New(1, 2)

		if !a.IsSubset(b) {
			t.Error("Expected a to be subset of b")
		}

		if !b.IsSuperset(a) {
			t.Error("Expected b to be superset of a")
		}

		if !a.Equal(c) {
			t.Error("Expected a to equal c")
		}

		if !a.IsProperSubset(b) {
			t.Error("Expected a to be proper subset of b")
		}

		if a.IsProperSubset(c) {
			t.Error("Expected a not to be proper subset of c")
		}
	})

	t.Run("Predicate Operations", func(t *testing.T) {
		s := sets.New(2, 4, 6, 8, 10)

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
	empty := sets.New[int]()
	if empty.String() != "{}" {
		t.Errorf("Empty set string: got %s, want {}", empty.String())
	}

	// Single element
	single := sets.New(42)
	if single.String() != "{42}" {
		t.Errorf("Single element set string: got %s, want {42}", single.String())
	}

	// Multiple elements
	multiple := sets.New(3, 1, 2)
	if multiple.String() != "{1, 2, 3}" {
		t.Errorf("Multiple element set string: got %s, want {1, 2, 3}", multiple.String())
	}
}

// Benchmark set operations
func BenchmarkSetOperations(b *testing.B) {
	// Create test sets
	size := 1000
	a := sets.New[int]()
	bSlice := make([]int, size)
	for i := 0; i < size; i++ {
		a.Add(i)
		bSlice[i] = i + size/2 // 50% overlap
	}
	setB := sets.FromSlice(bSlice)

	b.Run("Add", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			s := sets.New[int]()
			s.Add(i)
		}
	})

	b.Run("Has", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = a.Has(i % size)
		}
	})

	b.Run("Union", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = a.Union(setB)
		}
	})

	b.Run("Intersect", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = a.Intersect(setB)
		}
	})

	b.Run("Difference", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = a.Difference(setB)
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
	words := sets.New("apple", "banana", "cherry", "date", "elderberry", "fig", "grape")

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
	words.ForEach(func(word string) {
		totalChars += len(word)
	})
	fmt.Printf("Total characters: %d\n", totalChars)

	// Output:
	// Complex filtering example:
	// Long words: {banana, cherry, elderberry}
	// Any word starts with 'a': true
	// All words lowercase: true
	// Total characters: 41
}

func init() {
	// Don't log during tests
	log.SetOutput(nil)
}
