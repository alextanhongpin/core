package number_test

import (
	"fmt"

	"github.com/alextanhongpin/core/types/number"
)

// Example: Basic clipping operations
func ExampleClip() {
	// Clip integers
	fmt.Println(number.Clip(0, 10, -5)) // 0
	fmt.Println(number.Clip(0, 10, 5))  // 5
	fmt.Println(number.Clip(0, 10, 15)) // 10

	// Clip floats
	fmt.Println(number.Clip(0.0, 1.0, -0.5)) // 0
	fmt.Println(number.Clip(0.0, 1.0, 0.5))  // 0.5
	fmt.Println(number.Clip(0.0, 1.0, 1.5))  // 1

	// Output:
	// 0
	// 5
	// 10
	// 0
	// 0.5
	// 1
}

// Example: Range validation and clipping
func ExampleInRange() {
	// Check if values are in range
	fmt.Println(number.InRange(0, 10, 5))  // true
	fmt.Println(number.InRange(0, 10, -1)) // false
	fmt.Println(number.InRange(0, 10, 11)) // false

	// Use with validation
	score := 95
	if !number.InRange(0, 100, score) {
		score = number.Clip(0, 100, score)
	}
	fmt.Println("Valid score:", score)

	// Output:
	// true
	// false
	// false
	// Valid score: 95
}

// Example: Linear interpolation and mapping
func ExampleLerp() {
	// Linear interpolation between 0 and 100
	fmt.Printf("%.1f\n", number.Lerp(0.0, 100.0, 0.0)) // 0.0
	fmt.Printf("%.1f\n", number.Lerp(0.0, 100.0, 0.5)) // 50.0
	fmt.Printf("%.1f\n", number.Lerp(0.0, 100.0, 1.0)) // 100.0

	// Animation example
	startPos := 10.0
	endPos := 90.0
	progress := 0.3 // 30% through animation
	currentPos := number.Lerp(startPos, endPos, progress)
	fmt.Printf("Current position: %.1f\n", currentPos)

	// Output:
	// 0.0
	// 50.0
	// 100.0
	// Current position: 34.0
}

// Example: Value mapping between ranges
func ExampleMap() {
	// Map mouse position (0-800) to rotation angle (0-360)
	mouseX := 400.0
	angle := number.Map(mouseX, 0, 800, 0, 360)
	fmt.Printf("Mouse at %.0f -> Angle: %.1f°\n", mouseX, angle)

	// Map sensor reading (0-1023) to voltage (0-5V)
	sensorValue := 512.0
	voltage := number.Map(sensorValue, 0, 1023, 0, 5)
	fmt.Printf("Sensor: %.0f -> Voltage: %.2fV\n", sensorValue, voltage)

	// Map temperature (0-40°C) to color hue (240-0, blue to red)
	temperature := 25.0
	hue := number.Map(temperature, 0, 40, 240, 0)
	fmt.Printf("Temperature: %.1f°C -> Hue: %.0f\n", temperature, hue)

	// Output:
	// Mouse at 400 -> Angle: 180.0°
	// Sensor: 512 -> Voltage: 2.50V
	// Temperature: 25.0°C -> Hue: 90
}

// Real-world example: Audio volume control
type VolumeControl struct {
	minDB     float64
	maxDB     float64
	currentDB float64
}

func NewVolumeControl() *VolumeControl {
	return &VolumeControl{
		minDB: -60.0, // Minimum dB (nearly silent)
		maxDB: 0.0,   // Maximum dB (full volume)
	}
}

func (vc *VolumeControl) SetVolumePercent(percent float64) {
	// Clip percentage to valid range
	percent = number.Clip(0.0, 100.0, percent)

	// Map percentage to dB (logarithmic scale)
	vc.currentDB = number.Map(percent, 0, 100, vc.minDB, vc.maxDB)
}

func (vc *VolumeControl) GetVolumePercent() float64 {
	return number.Map(vc.currentDB, vc.minDB, vc.maxDB, 0, 100)
}

func (vc *VolumeControl) AdjustVolume(deltaPercent float64) {
	currentPercent := vc.GetVolumePercent()
	newPercent := currentPercent + deltaPercent
	vc.SetVolumePercent(newPercent)
}

func ExampleVolumeControl() {
	volume := NewVolumeControl()

	// Set volume to 75%
	volume.SetVolumePercent(75)
	fmt.Printf("Volume: %.1f%% (%.1fdB)\n", volume.GetVolumePercent(), volume.currentDB)

	// Increase volume by 10%
	volume.AdjustVolume(10)
	fmt.Printf("After +10%%: %.1f%% (%.1fdB)\n", volume.GetVolumePercent(), volume.currentDB)

	// Try to set volume beyond maximum
	volume.SetVolumePercent(150) // Will be clipped to 100%
	fmt.Printf("Max volume: %.1f%% (%.1fdB)\n", volume.GetVolumePercent(), volume.currentDB)

	// Output:
	// Volume: 75.0% (-15.0dB)
	// After +10%: 85.0% (-9.0dB)
	// Max volume: 100.0% (0.0dB)
}

// Real-world example: Progress bar with smooth animation
type ProgressBar struct {
	current float64
	target  float64
	speed   float64 // Animation speed (0-1)
}

func NewProgressBar() *ProgressBar {
	return &ProgressBar{
		speed: 0.1, // 10% of the distance per frame
	}
}

func (pb *ProgressBar) SetTarget(targetPercent float64) {
	pb.target = number.Clip(0, 100, targetPercent)
}

func (pb *ProgressBar) Update() {
	// Smooth animation towards target
	pb.current = number.Lerp(pb.current, pb.target, pb.speed)

	// Snap to target if very close (prevent infinite animation)
	if number.Abs(pb.target-pb.current) < 0.1 {
		pb.current = pb.target
	}
}

func (pb *ProgressBar) GetProgress() float64 {
	return pb.current
}

func (pb *ProgressBar) IsComplete() bool {
	return pb.current >= 100
}

func ExampleProgressBar() {
	progress := NewProgressBar()
	progress.SetTarget(75)

	fmt.Println("Animating progress bar:")
	for i := range 20 {
		progress.Update()
		fmt.Printf("Frame %2d: %5.1f%%\n", i, progress.GetProgress())

		// Stop when we reach the target
		if number.Abs(progress.GetProgress()-75) < 0.1 {
			break
		}
	}

	// Output:
	// Animating progress bar:
	// Frame  0:   7.5%
	// Frame  1:  14.2%
	// Frame  2:  20.3%
	// Frame  3:  25.8%
	// Frame  4:  30.7%
	// Frame  5:  35.1%
	// Frame  6:  39.1%
	// Frame  7:  42.7%
	// Frame  8:  45.9%
	// Frame  9:  48.8%
	// Frame 10:  51.5%
	// Frame 11:  53.8%
	// Frame 12:  55.9%
	// Frame 13:  57.8%
	// Frame 14:  59.6%
	// Frame 15:  61.1%
	// Frame 16:  62.5%
	// Frame 17:  63.7%
	// Frame 18:  64.9%
	// Frame 19:  65.9%
}

// Real-world example: Game health system
type HealthSystem struct {
	maxHealth     int
	currentHealth int
	armor         int
}

func NewHealthSystem(maxHealth int) *HealthSystem {
	return &HealthSystem{
		maxHealth:     maxHealth,
		currentHealth: maxHealth,
		armor:         0,
	}
}

func (hs *HealthSystem) TakeDamage(damage int) {
	// Apply armor reduction (50% damage reduction)
	if hs.armor > 0 {
		damage = damage / 2
		hs.armor = number.ClipMin(0, hs.armor-1) // Armor degrades
	}

	hs.currentHealth = number.ClipMin(0, hs.currentHealth-damage)
}

func (hs *HealthSystem) Heal(amount int) {
	hs.currentHealth = number.ClipMax(hs.maxHealth, hs.currentHealth+amount)
}

func (hs *HealthSystem) AddArmor(amount int) {
	hs.armor = number.ClipMax(10, hs.armor+amount) // Max 10 armor
}

func (hs *HealthSystem) GetHealthPercent() float64 {
	return number.Map(float64(hs.currentHealth), 0, float64(hs.maxHealth), 0, 100)
}

func (hs *HealthSystem) IsAlive() bool {
	return hs.currentHealth > 0
}

func (hs *HealthSystem) IsCritical() bool {
	return hs.GetHealthPercent() < 25 // Below 25% health
}

func ExampleHealthSystem() {
	player := NewHealthSystem(100)

	fmt.Printf("Initial health: %d/%d (%.0f%%)\n",
		player.currentHealth, player.maxHealth, player.GetHealthPercent())

	// Add some armor
	player.AddArmor(3)
	fmt.Printf("Added armor: %d\n", player.armor)

	// Take damage
	player.TakeDamage(40)
	fmt.Printf("After 40 damage: %d/%d (%.0f%%) - Armor: %d\n",
		player.currentHealth, player.maxHealth, player.GetHealthPercent(), player.armor)

	// Take more damage
	player.TakeDamage(30)
	fmt.Printf("After 30 damage: %d/%d (%.0f%%) - Critical: %t\n",
		player.currentHealth, player.maxHealth, player.GetHealthPercent(), player.IsCritical())

	// Heal
	player.Heal(25)
	fmt.Printf("After healing: %d/%d (%.0f%%)\n",
		player.currentHealth, player.maxHealth, player.GetHealthPercent())

	// Output:
	// Initial health: 100/100 (100%)
	// Added armor: 3
	// After 40 damage: 80/100 (80%) - Armor: 2
	// After 30 damage: 65/100 (65%) - Critical: false
	// After healing: 90/100 (90%)
}

// Real-world example: Temperature controller
type TemperatureController struct {
	targetTemp   float64
	currentTemp  float64
	tolerance    float64
	heatingPower float64 // 0-100%
	coolingPower float64 // 0-100%
}

func NewTemperatureController(target float64) *TemperatureController {
	return &TemperatureController{
		targetTemp:  target,
		currentTemp: 20.0, // Room temperature
		tolerance:   0.5,  // ±0.5°C tolerance
	}
}

func (tc *TemperatureController) Update(deltaTime float64) {
	// Calculate temperature difference
	diff := tc.targetTemp - tc.currentTemp

	// Determine heating/cooling power based on difference
	if number.Abs(diff) <= tc.tolerance {
		// Within tolerance, no heating/cooling needed
		tc.heatingPower = 0
		tc.coolingPower = 0
	} else if diff > 0 {
		// Need heating
		tc.heatingPower = number.Clip(0, 100, number.Abs(diff)*20)
		tc.coolingPower = 0

		// Apply heating
		heatRate := tc.heatingPower / 100.0 * 5.0 // Max 5°C per second
		tc.currentTemp += heatRate * deltaTime
	} else {
		// Need cooling
		tc.coolingPower = number.Clip(0, 100, number.Abs(diff)*20)
		tc.heatingPower = 0

		// Apply cooling
		coolRate := tc.coolingPower / 100.0 * 3.0 // Max 3°C per second
		tc.currentTemp -= coolRate * deltaTime
	}

	// Ambient temperature loss
	ambientLoss := (tc.currentTemp - 20.0) * 0.1 * deltaTime
	tc.currentTemp -= ambientLoss
}

func (tc *TemperatureController) SetTarget(target float64) {
	tc.targetTemp = number.Clip(-10, 100, target) // Safe range
}

func (tc *TemperatureController) IsAtTarget() bool {
	return number.Abs(tc.targetTemp-tc.currentTemp) <= tc.tolerance
}

func ExampleTemperatureController() {
	controller := NewTemperatureController(25.0)

	fmt.Printf("Target: %.1f°C, Starting: %.1f°C\n",
		controller.targetTemp, controller.currentTemp)

	// Simulate temperature control over time
	deltaTime := 0.1 // 100ms updates
	for i := 0; i < 50; i++ {
		controller.Update(deltaTime)

		if i%10 == 0 { // Print every second
			fmt.Printf("Time %.1fs: %.1f°C (Heat: %.0f%%, Cool: %.0f%%)\n",
				float64(i)*deltaTime, controller.currentTemp,
				controller.heatingPower, controller.coolingPower)
		}

		if controller.IsAtTarget() && i > 20 {
			fmt.Printf("Target reached at time %.1fs\n", float64(i)*deltaTime)
			break
		}
	}

	// Output:
	// Target: 25.0°C, Starting: 20.0°C
	// Time 0.0s: 20.5°C (Heat: 100%, Cool: 0%)
	// Time 1.0s: 23.3°C (Heat: 38%, Cool: 0%)
	// Time 2.0s: 24.1°C (Heat: 18%, Cool: 0%)
	// Time 3.0s: 24.4°C (Heat: 12%, Cool: 0%)
	// Time 4.0s: 24.5°C (Heat: 10%, Cool: 0%)
	// Target reached at time 4.0s
}
