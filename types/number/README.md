# Number - Mathematical Utilities

[![Go Reference](https://pkg.go.dev/badge/github.com/alextanhongpin/core/types/number.svg)](https://pkg.go.dev/github.com/alextanhongpin/core/types/number)

Package `number` provides mathematical utilities and operations for numeric types. It includes functions for clipping values to ranges, linear interpolation, value mapping, and other numeric utilities that work across different numeric types using Go generics.

## Features

- **Range Clipping**: Constrain values within specified bounds
- **Linear Interpolation**: Smooth transitions between values  
- **Value Mapping**: Map values between different ranges
- **Normalization**: Convert values to and from normalized ranges
- **Mathematical Operations**: Absolute values, sign detection, rounding
- **Type Safety**: Generic functions that work with all numeric types
- **Game Development**: Utilities commonly needed in games and simulations
- **Zero Dependencies**: Pure Go implementation

## Installation

```bash
go get github.com/alextanhongpin/core/types/number
```

## Quick Start

```go
package main

import (
    "fmt"
    "github.com/alextanhongpin/core/types/number"
)

func main() {
    // Clip values to valid ranges
    score := number.Clip(0, 100, 150)  // 100
    
    // Linear interpolation
    pos := number.Lerp(0.0, 100.0, 0.5)  // 50.0
    
    // Map values between ranges
    angle := number.Map(mouseX, 0, 800, 0, 360)
    
    // Check ranges
    valid := number.InRange(0, 100, score)  // true
}
```

## API Reference

### Range Operations

#### `Clip[T Number](lo, hi, v T) T`
Constrains a value to be within the specified range [lo, hi].

```go
score := number.Clip(0, 100, 150)    // 100
temp := number.Clip(-10.0, 50.0, 75.0)  // 50.0
```

#### `ClipMin[T Number](minVal, v T) T`
Constrains a value to be at least the minimum value.

```go
health := number.ClipMin(0, health - damage)  // Never goes below 0
```

#### `ClipMax[T Number](maxVal, v T) T`
Constrains a value to be at most the maximum value.

```go
volume := number.ClipMax(100, volume + 10)  // Never exceeds 100
```

#### `InRange[T Number](lo, hi, v T) bool`
Checks if a value is within the specified range (inclusive).

```go
valid := number.InRange(18, 65, age)  // Check if age is valid
```

### Interpolation and Mapping

#### `Lerp[T constraints.Float](a, b, t T) T`
Performs linear interpolation between two values. `t` should be between 0 and 1.

```go
// Animation: move from position 10 to 90 over time
currentPos := number.Lerp(10.0, 90.0, animationProgress)
```

#### `Map[T constraints.Float](value, inMin, inMax, outMin, outMax T) T`
Maps a value from one range to another.

```go
// Map mouse position (0-800) to rotation angle (0-360)
angle := number.Map(mouseX, 0, 800, 0, 360)

// Map sensor reading (0-1023) to voltage (0-5V)
voltage := number.Map(sensorValue, 0, 1023, 0, 5)
```

#### `Normalize[T constraints.Float](value, min, max T) T`
Maps a value from range [min, max] to range [0, 1].

```go
normalized := number.Normalize(temperature, -10, 40)  // 0.0 to 1.0
```

#### `Denormalize[T constraints.Float](normalizedValue, min, max T) T`
Maps a value from range [0, 1] to range [min, max].

```go
temperature := number.Denormalize(0.5, -10, 40)  // 15.0
```

### Mathematical Operations

#### `Abs[T Number](v T) T`
Returns the absolute value of a number.

```go
distance := number.Abs(targetPos - currentPos)
```

#### `Sign[T Number](v T) int`
Returns the sign of a number (-1, 0, or 1).

```go
direction := number.Sign(velocity)  // -1, 0, or 1
```

#### `Round[T constraints.Float](v T) T`
Rounds a float to the nearest integer.

```go
rounded := number.Round(3.7)  // 4.0
```

## Real-World Examples

### Audio Volume Control

```go
type VolumeControl struct {
    minDB     float64
    maxDB     float64
    currentDB float64
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
```

### Game Health System

```go
type HealthSystem struct {
    maxHealth     int
    currentHealth int
    armor         int
}

func (hs *HealthSystem) TakeDamage(damage int) {
    // Apply armor reduction
    if hs.armor > 0 {
        damage = damage / 2
        hs.armor = number.ClipMin(0, hs.armor-1)
    }
    
    hs.currentHealth = number.ClipMin(0, hs.currentHealth-damage)
}

func (hs *HealthSystem) Heal(amount int) {
    hs.currentHealth = number.ClipMax(hs.maxHealth, hs.currentHealth+amount)
}

func (hs *HealthSystem) GetHealthPercent() float64 {
    return number.Map(float64(hs.currentHealth), 0, float64(hs.maxHealth), 0, 100)
}
```

### Smooth Progress Bar Animation

```go
type ProgressBar struct {
    current float64
    target  float64
    speed   float64
}

func (pb *ProgressBar) Update() {
    // Smooth animation towards target
    pb.current = number.Lerp(pb.current, pb.target, pb.speed)
    
    // Snap to target if very close
    if number.Abs(pb.target-pb.current) < 0.1 {
        pb.current = pb.target
    }
}
```

### Temperature Controller

```go
type TemperatureController struct {
    targetTemp    float64
    currentTemp   float64
    tolerance     float64
    heatingPower  float64
}

func (tc *TemperatureController) Update(deltaTime float64) {
    diff := tc.targetTemp - tc.currentTemp
    
    if number.Abs(diff) <= tc.tolerance {
        tc.heatingPower = 0
    } else if diff > 0 {
        tc.heatingPower = number.Clip(0, 100, number.Abs(diff)*20)
        heatRate := tc.heatingPower / 100.0 * 5.0
        tc.currentTemp += heatRate * deltaTime
    }
}
```

### UI Component - Slider Control

```go
type Slider struct {
    min, max   float64
    value      float64
    position   float64  // 0-1 normalized position
}

func (s *Slider) SetPosition(pos float64) {
    s.position = number.Clip(0, 1, pos)
    s.value = number.Denormalize(s.position, s.min, s.max)
}

func (s *Slider) SetValue(val float64) {
    s.value = number.Clip(s.min, s.max, val)
    s.position = number.Normalize(s.value, s.min, s.max)
}

func (s *Slider) GetDisplayValue() string {
    return fmt.Sprintf("%.1f", s.value)
}
```

### Game Camera System

```go
type Camera struct {
    position   Vector2
    target     Vector2
    followSpeed float64
    bounds     Rectangle
}

func (c *Camera) Update(deltaTime float64) {
    // Smooth camera following
    c.position.X = number.Lerp(c.position.X, c.target.X, c.followSpeed * deltaTime)
    c.position.Y = number.Lerp(c.position.Y, c.target.Y, c.followSpeed * deltaTime)
    
    // Keep camera within bounds
    c.position.X = number.Clip(c.bounds.Left, c.bounds.Right, c.position.X)
    c.position.Y = number.Clip(c.bounds.Top, c.bounds.Bottom, c.position.Y)
}

func (c *Camera) SetTarget(target Vector2) {
    c.target = target
}
```

## Common Use Cases

### Gaming
- **Health/Mana Systems**: Clip values to valid ranges
- **Animation**: Smooth interpolation between keyframes
- **Camera Controls**: Smooth following and boundary constraints
- **UI Elements**: Progress bars, sliders, meters
- **Physics**: Velocity limiting, force application

### Data Visualization
- **Chart Scaling**: Map data values to pixel coordinates
- **Color Mapping**: Map values to color gradients
- **Animation**: Smooth transitions between states
- **Normalization**: Convert data to standard ranges

### Control Systems
- **PID Controllers**: Value clamping and normalization
- **Sensor Processing**: Map raw readings to meaningful values
- **Motor Control**: Speed and position limiting
- **Temperature Control**: Heating/cooling power calculation

### UI/UX
- **Smooth Animations**: Easing and interpolation
- **Input Validation**: Range checking and clamping
- **Responsive Design**: Scale factors and breakpoints
- **Progress Indicators**: Value mapping and animation

## Performance Considerations

- **Zero Allocation**: All functions operate on values, no heap allocations
- **Generic Implementation**: Type-safe operations without boxing
- **Compile-Time Optimization**: Simple math operations optimize well
- **CPU Cache Friendly**: Small functions suitable for inlining

```go
// These operations are very fast and suitable for real-time use
for i := 0; i < 1000000; i++ {
    value := number.Clip(0, 100, values[i])
    normalized := number.Normalize(value, 0, 100)
    mapped := number.Map(normalized, 0, 1, -1, 1)
}
```

## Best Practices

### 1. Validate Input Ranges
```go
// Always validate min/max relationships
func CreateSlider(min, max, initial float64) *Slider {
    if min > max {
        min, max = max, min  // Swap if needed
    }
    return &Slider{
        min:   min,
        max:   max,
        value: number.Clip(min, max, initial),
    }
}
```

### 2. Use Appropriate Precision
```go
// For UI, lower precision is often sufficient
progress := number.Round(number.Map(completed, 0, total, 0, 100))

// For physics, higher precision may be needed
velocity := number.Lerp(currentVelocity, targetVelocity, deltaTime)
```

### 3. Handle Edge Cases
```go
func SafeMap(value, inMin, inMax, outMin, outMax float64) float64 {
    if inMax == inMin {
        return outMin  // Avoid division by zero
    }
    return number.Map(value, inMin, inMax, outMin, outMax)
}
```

### 4. Cache Expensive Calculations
```go
type ColorMapper struct {
    minHue, maxHue float64
    // Cache the range for repeated use
    hueRange float64
}

func (cm *ColorMapper) MapToHue(value, min, max float64) float64 {
    normalized := number.Normalize(value, min, max)
    return cm.minHue + normalized * cm.hueRange
}
```

## Integration with Other Packages

### With Animation Systems
```go
import "github.com/alextanhongpin/core/types/number"

type Animator struct {
    startValue, endValue float64
    duration, elapsed    time.Duration
}

func (a *Animator) Update(deltaTime time.Duration) float64 {
    a.elapsed += deltaTime
    progress := number.Clip(0, 1, float64(a.elapsed)/float64(a.duration))
    return number.Lerp(a.startValue, a.endValue, progress)
}
```

### With Game Engines
```go
// Integrate with game loop
func (game *Game) Update(deltaTime float64) {
    // Update player health with clamping
    game.player.health = number.ClipMin(0, game.player.health)
    
    // Smooth camera following
    game.camera.x = number.Lerp(game.camera.x, game.player.x, 0.1)
    
    // Map joystick input to movement speed
    speed := number.Map(joystickMagnitude, 0, 1, 0, game.player.maxSpeed)
}
```

## Testing

```go
func TestClipping(t *testing.T) {
    tests := []struct {
        min, max, input, expected int
    }{
        {0, 10, -5, 0},
        {0, 10, 5, 5},
        {0, 10, 15, 10},
    }
    
    for _, test := range tests {
        result := number.Clip(test.min, test.max, test.input)
        assert.Equal(t, test.expected, result)
    }
}
```

## License

MIT License - see LICENSE file for details.
