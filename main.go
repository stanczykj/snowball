package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
)

var previousMoves [2]string

func init() {
	previousMoves = [2]string{"L", "F"}
}

func main() {
	port := "8080"
	if v := os.Getenv("PORT"); v != "" {
		port = v
	}
	http.HandleFunc("/", handler)

	log.Printf("starting server on port :%s", port)
	err := http.ListenAndServe(":"+port, nil)
	log.Fatalf("http listen error: %v", err)
}

func handler(w http.ResponseWriter, req *http.Request) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Fprint(w, "T")
		}
	}()

	if req.Method == http.MethodGet {
		fmt.Fprint(w, "Let the battle begin!")
		return
	}

	var v ArenaUpdate
	defer req.Body.Close()
	d := json.NewDecoder(req.Body)
	d.DisallowUnknownFields()
	if err := d.Decode(&v); err != nil {
		log.Printf("WARN: failed to decode ArenaUpdate in response body: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	resp := play(v)
	fmt.Fprint(w, resp)
}

func play(input ArenaUpdate) (response string) {
	log.Printf("IN: %#v", input)
	arenaMap := make([][]string, input.Arena.Dimensions[0])
	for y := range arenaMap {
		arenaMap[y] = make([]string, input.Arena.Dimensions[1])
	}
	for k, v := range input.Arena.State {
		arenaMap[v.X][v.Y] = k
	}
	playerState := input.Arena.State[input.Links.Self.Href]
	arenaMap[playerState.X][playerState.Y] = ""
	if cords := canBeShot(playerState, arenaMap, &input); cords != nil {
		return run(playerState, cords, arenaMap)
	}
	if canShoot(playerState, arenaMap) {
		return "T"
	}

	return nextMove(playerState, input.Arena.Dimensions)
}

func canShoot(playerState PlayerState, arenaMap [][]string) bool {
	if playerState.Direction == "N" || playerState.Direction == "S" {
		change := -1
		if playerState.Direction == "S" {
			change = 1
		}
		for i := 1; i < 4; i++ {
			y := clamp(playerState.Y+(i*change), 0, len(arenaMap[0])-1)
			if arenaMap[playerState.X][y] != "" {
				return true
			}
		}
	}
	if playerState.Direction == "W" || playerState.Direction == "E" {
		change := -1
		if playerState.Direction == "E" {
			change = 1
		}
		for i := 1; i < 4; i++ {
			x := clamp(playerState.X+(i*change), 0, len(arenaMap)-1)
			if arenaMap[x][playerState.Y] != "" {
				return true
			}
		}
	}
	return false
}

func canBeShot(playerState PlayerState, arenaMap [][]string, arenaUpdate *ArenaUpdate) []int {
	if !playerState.WasHit {
		return nil
	}
	areaWidth := len(arenaMap) - 1
	arenaHeight := len(arenaMap[0]) - 1
	cords := make([]int, 2)
	for i := 1; i < 4; i++ {
		if x := clamp(playerState.X-i, 0, areaWidth); arenaMap[x][playerState.Y] != "" {
			enemy := arenaMap[x][playerState.Y]
			if arenaUpdate.Arena.State[enemy].Direction == "E" {
				cords[0] = x
				cords[1] = playerState.Y
				return cords
			}
		}
		if x := clamp(playerState.X+i, 0, areaWidth); arenaMap[x][playerState.Y] != "" {
			enemy := arenaMap[x][playerState.Y]
			if arenaUpdate.Arena.State[enemy].Direction == "W" {
				cords[0] = x
				cords[1] = playerState.Y
				return cords
			}
		}
		if y := clamp(playerState.Y-i, 0, arenaHeight); arenaMap[playerState.X][y] != "" {
			enemy := arenaMap[playerState.X][y]
			if arenaUpdate.Arena.State[enemy].Direction == "S" {
				cords[0] = playerState.X
				cords[1] = y
				return cords
			}
		}
		if y := clamp(playerState.Y+i, 0, arenaHeight); arenaMap[playerState.X][y] != "" {
			enemy := arenaMap[playerState.X][y]
			if arenaUpdate.Arena.State[enemy].Direction == "N" {
				cords[0] = playerState.X
				cords[1] = y
				return cords
			}
		}
	}
	return nil
}

func nextMove(playerState PlayerState, arenaDimensions []int) string {
	var move string
	if playerState.X-moveMargin <= 0 && playerState.Direction == "W" {
		move = rotateDirection
	} else if playerState.X+moveMargin >= arenaDimensions[0] && playerState.Direction == "E" {
		move = rotateDirection
	} else if playerState.Y-moveMargin <= 0 && playerState.Direction == "N" {
		move = rotateDirection
	} else if playerState.Y-moveMargin >= arenaDimensions[1] && playerState.Direction == "S" {
		move = rotateDirection
	} else if previousMoves[0] == "F" {
		if playerState.X <= moveMargin {
			if playerState.Direction == "S" {
				move = "L"
			} else if playerState.Direction == "N" {
				move = "R"
			}
		} else if playerState.X >= arenaDimensions[0]-moveMargin {
			if playerState.Direction == "S" {
				move = "R"
			} else if playerState.Direction == "N" {
				move = "L"
			}
		} else if playerState.Y <= moveMargin {
			if playerState.Direction == "W" {
				move = "L"
			} else if playerState.Direction == "E" {
				move = "R"
			}
		} else if playerState.Y >= arenaDimensions[1]-moveMargin {
			if playerState.Direction == "W" {
				move = "R"
			} else if playerState.Direction == "E" {
				move = "L"
			}
		} else if previousMoves[1] == "L" {
			move = "R"
		} else {
			move = "L"
		}
	} else {
		move = "F"
	}
	previousMoves[1] = previousMoves[0]
	previousMoves[0] = move
	return move
}

func run(playerState PlayerState, attackerCoords []int, arenaMap [][]string) string {
	var move string
	if attackerCoords[0] == playerState.X {
		if playerState.Direction == "W" || playerState.Direction == "E" {
			var newX int
			if playerState.Direction == "W" {
				newX = clamp(playerState.X-1, 0, len(arenaMap))
			} else {
				newX = clamp(playerState.X+1, 0, len(arenaMap))
			}
			if arenaMap[newX][playerState.Y] == "" {
				move = "F"
			} else {
				move = rotateDirection
			}
		} else {
			move = rotateDirection
		}
	} else if attackerCoords[1] == playerState.Y {
		if playerState.Direction == "S" || playerState.Direction == "N" {
			var newY int
			if playerState.Direction == "S" {
				newY = clamp(playerState.Y+1, 0, len(arenaMap[0]))
			} else {
				newY = clamp(playerState.Y-1, 0, len(arenaMap[0]))
			}
			if arenaMap[playerState.X][newY] == "" {
				move = "F"
			} else {
				move = rotateDirection
			}
		} else {
			move = rotateDirection
		}
	}
	previousMoves[1] = previousMoves[0]
	previousMoves[0] = move
	return move
}

func clamp(number int, min int, max int) int {
	if number < min {
		return min
	}
	if number > max {
		return max
	}
	return number
}

const (
	rotateDirection = "L"
	moveMargin      = 0
)
