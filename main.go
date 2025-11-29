package main

import (
	"math"
	"math/rand"

	. "github.com/gen2brain/raylib-go/raylib"
	rl "github.com/gen2brain/raylib-go/raylib"
)

const (
	screenWidth              = 1920
	screenHeight             = 1000
	mapWidth                 = 64
	mapHeight                = 64
	mouseSensitivity         = 0.002
	fogDistance      float32 = 20.0
	playerRadius     float32 = 0.1
	bobFrequnecy     float32 = 2.0
	bobAmplitude     float32 = 6.0
	//===============
	// Torch lighting
	torchIntensity float32 = 0.4
	torchRadius    float32 = 5.0
	torchFlicker   float32 = 0.05
	//===============
	// Flickering lights
	lightIntensity float32 = 0.5
	lightRadius    float32 = 4.0
	lightFlicker   float32 = 0.1
	//===============
	// Procedural gen params
	fillPercent  int = 45
	caIterations int = 5
	wallCutoff   int = 4
	floorCutoff  int = 5
)

var (
	mapData = [mapWidth * mapHeight]int{}

	// Player state
	pos   = Vector2{X: 2.5, Y: 2.5}
	dir   = Vector2{X: 1, Y: 0}
	plane = Vector2{X: 0, Y: 0.66}

	// Colors
	ceilingColor = NewColor(2, 2, 20, 255)
	floorColor   = NewColor(12, 12, 12, 255)

	// Textures
	stoneWall Texture2D
	mossyWall Texture2D

	// Head bob
	bobPhase float32 = 0.0

	// Lighting
	lightSources      []lightSource
	torchFlickerPhase float32
)

type lightSource struct {
	pos          Vector2
	phaseOffset  float32
	flickerSpeed float32
}

func initMap() {
	for i := range mapData {
		if rand.Intn(90) < fillPercent {
			mapData[i] = 1 // Wall
		} else {
			mapData[i] = 0 // Floor
		}
	}

	// Cellular automata iterations
	tempMap := [mapWidth * mapHeight]int{}
	for range caIterations {
		copy(tempMap[:], mapData[:])

		for y := 1; y < mapHeight-1; y++ {
			for x := 1; x < mapWidth-1; x++ {
				idx := y*mapWidth + x
				wallCount := countWalls(tempMap[:], mapWidth, x, y)

				if tempMap[idx] == 1 && wallCount < wallCutoff {
					mapData[idx] = 0
				} else if tempMap[idx] == 0 && wallCount > floorCutoff {
					mapData[idx] = 1
				}
			}
		}
	}

	// Ensure borders are walls
	for x := range mapWidth {
		mapData[0*mapWidth+x] = 1
		mapData[(mapHeight-1)*mapWidth+x] = 1
	}
	for y := range mapHeight {
		mapData[y*mapWidth+0] = 1
		mapData[y*mapWidth+(mapWidth-1)] = 1
	}

	// Place player in valid spot
	for {
		pos.X = 1 + rand.Float32()*(float32(mapWidth)-3)
		pos.Y = 1 + rand.Float32()*(float32(mapHeight)-3)
		if isValidPosition(pos) {
			break
		}
	}

	// Init Lighting at specified map spots
	lightSources = make([]lightSource, 0)
	for y := range mapHeight {
		for x := range mapWidth {
			if mapData[y*mapWidth+x] == 1 && rand.Float32() < 0.1 { // 10% chance
				lightSources = append(lightSources, lightSource{
					pos:          Vector2{X: float32(x) + 0.5, Y: float32(y) + 0.5},
					phaseOffset:  rand.Float32() * 2 * math.Pi,
					flickerSpeed: 1.0 + rand.Float32()*0.5,
				})
			}
		}
	}
}

func countWalls(data []int, w, x, y int) int {
	count := 0
	for dy := -1; dy <= 1; dy++ {
		for dx := -1; dx <= 1; dx++ {
			if dx == 0 && dy == 0 {
				continue
			}
			nx, ny := x+dx, y+dy
			if nx >= 0 && nx < w && ny >= 0 && ny < mapHeight {
				if data[ny*w+nx] == 1 {
					count++
				}
			}
		}
	}

	return count
}

func main() {
	InitWindow(screenWidth, screenHeight, "~ GOCASTER ~")
	defer CloseWindow()

	SetTargetFPS(120)
	DisableCursor()

	stoneWall = LoadTexture("Assets/wall.png")
	mossyWall = LoadTexture("Assets/mossyStone.png")

	initMap()

	// Main loop
	for !WindowShouldClose() {
		dt := GetFrameTime()

		torchFlickerPhase += dt * 2.0
		if torchFlickerPhase > 2*math.Pi {
			torchFlickerPhase -= 2 * math.Pi
		}

		// Handle Input
		moveSpeed := 4.0 * dt // Units per second
		rotSpeed := 2.0 * dt  // Radians per second

		rotationAngle := 0.0
		if IsKeyDown(KeyLeft) {
			rotationAngle -= float64(rotSpeed)
		}
		if IsKeyDown(KeyRight) {
			rotationAngle += float64(rotSpeed)
		}
		mouseDelta := GetMouseDelta()
		rotationAngle += float64(mouseDelta.X) * mouseSensitivity
		if rotationAngle != 0 {
			rotate(float32(rotationAngle))
		}

		moveDir := Vector2Zero()
		if IsKeyDown(KeyW) {
			moveDir = Vector2Add(moveDir, dir)
		}
		if IsKeyDown(KeyS) {
			moveDir = Vector2Add(moveDir, rl.Vector2Scale(dir, -1))
		}
		if IsKeyDown(KeyD) {
			strafeDir := Vector2Normalize(plane)
			moveDir = Vector2Add(moveDir, strafeDir)
		}
		if IsKeyDown(KeyA) {
			strafeDir := Vector2Normalize(plane)
			moveDir = Vector2Add(moveDir, Vector2Scale(strafeDir, -1))
		}

		moveLength := Vector2Length(moveDir)
		if moveLength > 0 {
			// Normalize and scale by moveSpeed
			moveDir = Vector2Scale(Vector2Normalize(moveDir), moveSpeed)

			// Attempt full movement
			newPos := Vector2Add(pos, moveDir)
			if isValidPosition(newPos) {
				pos = newPos
			} else {
				newPosX := Vector2{X: pos.X + moveDir.X, Y: pos.Y}
				if isValidPosition(newPosX) {
					pos = newPosX
				} else {
					newPosY := Vector2{X: pos.X, Y: pos.Y + moveDir.Y}
					if isValidPosition(newPosY) {
						pos = newPosY
					}
				}
			}

			bobPhase += moveSpeed * bobFrequnecy
			if bobPhase > 2*math.Pi {
				bobPhase -= 2 * math.Pi
			}
		} else {
			bobPhase = 0.0
		}

		BeginDrawing()
		ClearBackground(Black)

		// Ceiling and floor rendering
		DrawRectangle(0, 0, screenWidth, screenHeight/2, ceilingColor)
		DrawRectangle(0, screenHeight/2, screenWidth, screenHeight/2, floorColor)

		bobOffset := bobAmplitude * float32(math.Sin(float64(bobPhase)))

		var sourceRec, destRec Rectangle

		// Raycasting procedure
		for x := range screenWidth {
			cameraX := 2*float32(x)/float32(screenWidth) - 1
			rayDir := Vector2Add(dir, Vector2Scale(plane, cameraX))

			// Cast ray
			wallType, side, distance, wallX := castRay(mapData[:], mapWidth, pos, rayDir)

			// Calculate wall height and drawing range
			lineHeightFloat := float32(screenHeight) / distance
			wallTop := (float32(screenHeight)-lineHeightFloat)/2.0 + bobOffset

			// Texture selection
			var texture Texture2D
			switch wallType {
			case 1:
				texture = stoneWall
			case 2:
				texture = mossyWall
			}

			texX := int(wallX * float32(texture.Width))
			sourceRec.X = float32(texX)
			sourceRec.Y = 0
			sourceRec.Width = 1
			sourceRec.Height = float32(texture.Height)
			destRec.X = float32(x)
			destRec.Y = wallTop
			destRec.Width = 1
			destRec.Height = lineHeightFloat

			brightness := float32(0.0)

			// Distance-based dynamic lighting
			brightnessDist := 1.0 - distance/fogDistance
			if brightnessDist < 0 {
				brightness += brightnessDist * 0.5
			}

			// Torch lighting
			hitPoint := Vector2Add(pos, Vector2Scale(rayDir, distance))
			torchDist := Vector2Distance(pos, hitPoint)
			if torchDist < torchRadius {
				torchBrightness := torchIntensity + (1.0 - torchDist/torchRadius)
				torchFlickerFactor := 1.0 + torchFlicker*float32(math.Sin(float64(torchFlickerPhase)))
				brightness += torchBrightness * torchFlickerFactor
			}

			// Static light sources
			for _, light := range lightSources {
				lightDist := Vector2Distance(light.pos, hitPoint)
				if lightDist < lightRadius {
					lightBrightness := lightIntensity * (1.0 - lightDist/lightRadius) * (1.0 - lightDist/lightRadius)
					flickerFactor := 1.0 + lightFlicker*float32(math.Sin(float64(light.flickerSpeed*torchFlickerPhase+light.phaseOffset)))
					brightness += lightBrightness * flickerFactor
				}
			}

			var sideFactor float32 = 1.0
			if side == 1 {
				sideFactor = 0.75
			}
			if brightness > 1.0 {
				brightness = 1.0
			} else if brightness < 0.0 {
				brightness = 0.0
			}
			finalBrightness := brightness * sideFactor
			tint := NewColor(
				uint8(255*finalBrightness),
				uint8(255*finalBrightness),
				uint8(255*finalBrightness),
				255,
			)

			DrawTexturePro(texture, sourceRec, destRec, Vector2Zero(), 0, tint)
		}
		// Draw Crosshair
		DrawLine(screenWidth/2-10, screenHeight/2, screenWidth/2+10, screenHeight/2, RayWhite)
		DrawLine(screenWidth/2, screenHeight/2-10, screenWidth/2, screenHeight/2+10, RayWhite)

		DrawFPS(10, 10)
		EndDrawing()
	}

	// Unload textures to prevent memory leaks
	UnloadTexture(stoneWall)
	UnloadTexture(mossyWall)
}

func isValidPosition(p Vector2) bool {
	mapX := int(p.X)
	mapY := int(p.Y)

	var wallRect Rectangle
	wallRect.Width = 1
	wallRect.Height = 1

	// Check 3x3 grid around position
	for dx := -1; dx <= 1; dx++ {
		for dy := -1; dy <= 1; dy++ {
			checkX := mapX + dx
			checkY := mapY + dy
			if checkX < 0 || checkX >= mapWidth || checkY < 0 || checkY >= mapHeight {
				continue
			}
			if mapData[checkY*mapWidth+checkX] != 0 {
				wallRect.X = float32(checkX)
				wallRect.Y = float32(checkY)
				if CheckCollisionCircleRec(p, playerRadius, wallRect) {
					return false
				}
			}
		}
	}
	return true
}

func castRay(mapData []int, mapWidth int, pos Vector2, rayDir Vector2) (wallType int, side int, distance float32, wallX float32) {
	mapX := int(pos.X)
	mapY := int(pos.Y)

	deltaDistX := float32(math.Abs(1 / float64(rayDir.X)))
	deltaDistY := float32(math.Abs(1 / float64(rayDir.Y)))

	var stepX, stepY int
	var sideDistX, sideDistY float32

	if rayDir.X < 0 {
		stepX = -1
		sideDistX = (pos.X - float32(mapX)) * deltaDistX
	} else {
		stepX = 1
		sideDistX = (float32(mapX) + 1.0 - pos.X) * deltaDistX
	}
	if rayDir.Y < 0 {
		stepY = -1
		sideDistY = (pos.Y - float32(mapY)) * deltaDistY
	} else {
		stepY = 1
		sideDistY = (float32(mapY) + 1.0 - pos.Y) * deltaDistY
	}

	// DDA Loop
	for {
		if sideDistX < sideDistY {
			sideDistX += deltaDistX
			mapX += stepX
			side = 0 // Hit on x-side
		} else {
			sideDistY += deltaDistY
			mapY += stepY
			side = 1 // Hit on y-side
		}

		// Check bounds and wall hit
		if mapX < 0 || mapX >= mapWidth || mapY < 0 || mapY >= mapHeight {
			break
		}
		wallType = mapData[mapY*mapWidth+mapX]
		if wallType != 0 {
			break
		}
	}

	if side == 0 {
		distance = (float32(mapX) - pos.X + (1-float32(stepX))/2) / rayDir.X
	} else {
		distance = (float32(mapY) - pos.Y + (1-float32(stepY))/2) / rayDir.Y
	}

	hitPoint := Vector2Add(pos, Vector2Scale(rayDir, distance))
	if side == 0 {
		wallX = hitPoint.Y - float32(math.Floor(float64(hitPoint.Y)))
	} else {
		wallX = hitPoint.X - float32(math.Floor(float64(hitPoint.X)))
	}

	return wallType, side, distance, wallX
}

func rotate(angle float32) {
	oldDir := dir
	dir.X = oldDir.X*float32(math.Cos(float64(angle))) - oldDir.Y*float32(math.Sin(float64(angle)))
	dir.Y = oldDir.X*float32(math.Sin(float64(angle))) + oldDir.Y*float32(math.Cos(float64(angle)))

	oldPlane := plane
	plane.X = oldPlane.X*float32(math.Cos(float64(angle))) - oldPlane.Y*float32(math.Sin(float64(angle)))
	plane.Y = oldPlane.X*float32(math.Sin(float64(angle))) + oldPlane.Y*float32(math.Cos(float64(angle)))
}
